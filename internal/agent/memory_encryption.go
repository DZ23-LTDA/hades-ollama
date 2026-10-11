package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Criptografia da memória em repouso (atividade do estágio 15).
//
// PRINCÍPIO: nenhuma criptografia caseira. O primitivo é o MESMO já usado para
// credenciais de conector (`encryptCredential`): AES-256-GCM, chave derivada por
// SHA-256 de `OLLAMA_AGENT_CREDENTIAL_KEY`, nonce aleatório por registro. Inventar
// um segundo esquema criaria duas verdades sobre a mesma chave.
//
// DECISÕES DELIBERADAS:
//   - é OPT-IN: sem chave configurada o conteúdo continua em texto claro, como
//     antes. Exigir chave quebraria instalações existentes de uma vez;
//   - o TEXTO CLARO NÃO VAI PARA O DISCO quando a chave existe: `Content` fica
//     vazio no arquivo e o cifrado vai em `encrypted_content`;
//   - a LEITURA É FAIL-CLOSED: registro cifrado sem chave disponível **falha com
//     erro explícito** em vez de voltar vazio. Devolver vazio em silêncio seria
//     perda de dado disfarçada de memória vazia;
//   - TODAS as escritas passam por `writeMemoriesFile`: eram CINCO caminhos
//     gravando o mesmo arquivo, e proteger quatro deixaria um deles salvando texto
//     claro — o mesmo defeito de "irmão esquecido" que já corrigi três vezes.

const (
	// MemoryEncryptionAlgorithm identifica o esquema no relatório de capacidade.
	MemoryEncryptionAlgorithm = "AES-256-GCM"
	// MemoryEncryptionKeySource diz de onde vem a chave.
	MemoryEncryptionKeySource = "OLLAMA_AGENT_CREDENTIAL_KEY"
)

// MemoryEncryptionStatus é o estado REAL da criptografia em repouso, para o
// produto poder dizer a verdade em vez de sugerir que protege.
type MemoryEncryptionStatus struct {
	Configured bool   `json:"configured"`
	AtRest     bool   `json:"at_rest"`
	Algorithm  string `json:"algorithm,omitempty"`
	KeySource  string `json:"key_source,omitempty"`
	Guidance   string `json:"guidance,omitempty"`
}

// MemoryEncryption descreve honestamente se a memória é gravada cifrada.
func MemoryEncryption() MemoryEncryptionStatus {
	if _, err := credentialKey(); err != nil {
		return MemoryEncryptionStatus{
			Configured: false,
			AtRest:     false,
			Guidance: "Defina " + MemoryEncryptionKeySource + " (mínimo 16 caracteres) para gravar a " +
				"memória cifrada em repouso. Sem a chave, o conteúdo fica em texto claro no store local.",
		}
	}
	return MemoryEncryptionStatus{
		Configured: true,
		AtRest:     true,
		Algorithm:  MemoryEncryptionAlgorithm,
		KeySource:  MemoryEncryptionKeySource,
	}
}

// ErrMemoryEncryptionUnavailable indica leitura de memória cifrada sem chave.
var ErrMemoryEncryptionUnavailable = errors.New("memory content is encrypted and the key is unavailable")

func decryptMemoryContent(value string) (string, error) {
	key, err := credentialKey()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMemoryEncryptionUnavailable, err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid memory ciphertext: %w", err)
	}
	if len(sealed) < aead.NonceSize() {
		return "", errors.New("invalid memory ciphertext")
	}
	plaintext, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("memory ciphertext did not authenticate with the configured key")
	}
	return string(plaintext), nil
}

// memoriesForDisk devolve a cópia que vai para o disco. Com chave configurada,
// `Content` sai vazio e o cifrado vai em `encrypted_content` — no registro E no
// histórico: o texto anterior guardado em `Versions` é texto do usuário e vazaria
// em claro se ficasse de fora (foi o próprio teste que pegou isso).
func memoriesForDisk(memories []Memory) ([]Memory, error) {
	if !MemoryEncryption().Configured {
		return memories, nil
	}
	onDisk := make([]Memory, 0, len(memories))
	for _, memory := range memories {
		stored := memory
		if strings.TrimSpace(stored.Content) != "" {
			sealed, err := encryptCredential(stored.Content)
			if err != nil {
				return nil, fmt.Errorf("encrypt memory content: %w", err)
			}
			stored.EncryptedContent = sealed
			stored.Content = ""
		}
		if len(stored.Versions) > 0 {
			versions := make([]MemoryVersion, 0, len(stored.Versions))
			for _, version := range stored.Versions {
				if strings.TrimSpace(version.Content) != "" {
					sealed, err := encryptCredential(version.Content)
					if err != nil {
						return nil, fmt.Errorf("encrypt memory version: %w", err)
					}
					version.EncryptedContent = sealed
					version.Content = ""
				}
				versions = append(versions, version)
			}
			stored.Versions = versions
		}
		onDisk = append(onDisk, stored)
	}
	return onDisk, nil
}

// memoriesFromDisk devolve a cópia em memória, decifrando o que estiver cifrado.
func memoriesFromDisk(memories []Memory) ([]Memory, error) {
	loaded := make([]Memory, 0, len(memories))
	for _, memory := range memories {
		if strings.TrimSpace(memory.EncryptedContent) != "" {
			plaintext, err := decryptMemoryContent(memory.EncryptedContent)
			if err != nil {
				return nil, fmt.Errorf("memory %s: %w", memory.ID, err)
			}
			memory.Content = plaintext
			memory.EncryptedContent = ""
		}
		for index, version := range memory.Versions {
			if strings.TrimSpace(version.EncryptedContent) == "" {
				continue
			}
			plaintext, err := decryptMemoryContent(version.EncryptedContent)
			if err != nil {
				return nil, fmt.Errorf("memory %s revision %d: %w", memory.ID, version.Revision, err)
			}
			memory.Versions[index].Content = plaintext
			memory.Versions[index].EncryptedContent = ""
		}
		loaded = append(loaded, memory)
	}
	return loaded, nil
}

// writeMemoriesFile é o ÚNICO caminho de escrita do arquivo de memórias: cinco
// pontos gravavam o mesmo arquivo e qualquer um fora daqui salvaria texto claro.
func (s *ContextStore) writeMemoriesFile(projectID string, memories []Memory) error {
	onDisk, err := memoriesForDisk(memories)
	if err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(s.root, "memories", projectID+".json"), onDisk)
}
