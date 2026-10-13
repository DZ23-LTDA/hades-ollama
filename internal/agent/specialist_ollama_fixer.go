package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// P0-E / T4 — FIXER COM MODELO LOCAL E OBSERVABILIDADE REAL (E-349).
//
// Fecha o `NAO_MEDIDO` declarado em E-348: até aqui o fixer do caminho T4 era **determinístico**; a execução com
// **modelo local** não tinha sido medida. Aqui o fixer fala com o **Ollama local** e — importante — **respeita o
// segmento**: se o modelo propuser arquivo fora do segmento, a proposta é **RECUSADA**.
//
// ***Confiar no modelo para respeitar fronteira é o mesmo que não ter fronteira.*** O segmento é conferido **no fixer**
// (antes de escrever) **e** no coordenador (antes de mesclar): duas barreiras, porque a segunda é tarde para evitar
// que um arquivo alheio já esteja alterado na worktree.
//
// OBSERVABILIDADE: quando um `*Runtime` é informado, o laço existente emite eventos (`mission.repair_attempt`, etc.) no
// armazenamento real. Sem `Runtime`, os eventos **não** são emitidos — e isso continua **declarado**, não escondido.

var (
	// ErrFixerOutOfSegment recusa proposta de arquivo fora do segmento do especialista.
	ErrFixerOutOfSegment = errors.New("fixer proposed a file outside its segment")
	// ErrFixerModelUnavailable declara ausência do modelo em vez de fingir correção.
	ErrFixerModelUnavailable = errors.New("fixer model is unavailable")
)

// OllamaSpecialistFixer conserta DENTRO da worktree usando o Ollama local.
type OllamaSpecialistFixer struct {
	BaseURL string
	Modelo  string
	Client  *http.Client
	// MaxBytes limita o tamanho de cada arquivo enviado no prompt (limite declarado, não escondido).
	MaxBytes int
}

// Name identifica o fixer nos registros.
func (f OllamaSpecialistFixer) Name() string { return "ollama-fixer" }

// Fix pede a correção ao modelo e escreve SOMENTE arquivos do segmento.
func (f OllamaSpecialistFixer) Fix(ctx context.Context, worktreeDir string, tarefa SpecialistTask, tentativa int, ultimaSaida string) error {
	if strings.TrimSpace(f.Modelo) == "" {
		return fmt.Errorf("%w: modelo não informado", ErrFixerModelUnavailable)
	}
	limite := f.MaxBytes
	if limite <= 0 {
		limite = 256 << 10
	}
	// O enunciado é o segmento do especialista + a saída REAL dos testes.
	permitidos := map[string]bool{}
	prompt := "Você vai corrigir um bug em Go. Responda SOMENTE com o conteúdo COMPLETO de cada arquivo corrigido, " +
		"dentro de blocos delimitados por <<<ARQUIVO nome>>> e <<<FIM>>>. Altere APENAS estes arquivos:\n"
	for _, arquivo := range tarefa.Segment {
		limpo := filepath.Clean(arquivo)
		permitidos[limpo] = true
		prompt += "- " + limpo + "\n"
	}
	prompt += "\nTentativa " + fmt.Sprint(tentativa) + ".\n"
	if strings.TrimSpace(ultimaSaida) != "" {
		prompt += "\nSaída dos testes que falharam:\n" + recortar(ultimaSaida, 6000) + "\n"
	}
	nomes := make([]string, 0, len(permitidos))
	for arquivo := range permitidos {
		nomes = append(nomes, arquivo)
	}
	sort.Strings(nomes)
	for _, arquivo := range nomes {
		conteudo, err := os.ReadFile(filepath.Join(worktreeDir, arquivo))
		if err != nil {
			continue
		}
		prompt += "\n=== " + arquivo + " ===\n" + recortar(string(conteudo), limite) + "\n"
	}
	if strings.TrimSpace(tarefa.Instruction) != "" {
		prompt += "\nInstrução: " + tarefa.Instruction + "\n"
	}

	resposta, err := OllamaGenerate(ctx, f.BaseURL, f.Modelo, prompt, f.Client)
	if err != nil {
		return err
	}
	proposta, err := ExtrairArquivosDaResposta(resposta)
	if err != nil {
		return err
	}
	if len(proposta) == 0 {
		return fmt.Errorf("%w: o modelo não devolveu nenhum bloco <<<ARQUIVO ...>>>", ErrFixerModelUnavailable)
	}
	// SEGMENTO CONFERIDO ANTES DE ESCREVER: proposta fora do segmento é RECUSA, não aviso.
	escritos := 0
	for nome, conteudo := range proposta {
		limpo := filepath.Clean(nome)
		if !permitidos[limpo] {
			return fmt.Errorf("%w: %q", ErrFixerOutOfSegment, nome)
		}
		if len(conteudo) > limite*8 {
			return fmt.Errorf("%w: arquivo %q acima do limite declarado", ErrSpecialistInvalid, nome)
		}
		caminho := filepath.Join(worktreeDir, limpo)
		if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
			return err
		}
		escritos++
	}
	if escritos == 0 {
		return fmt.Errorf("%w: nenhum arquivo do segmento foi escrito", ErrFixerModelUnavailable)
	}
	return nil
}

func recortar(texto string, limite int) string {
	if len(texto) <= limite {
		return texto
	}
	return texto[:limite] + "\n[...recortado...]"
}

// OllamaGenerate é a chamada CRUA ao Ollama local — usada pelo harness do T2 e pelo fixer do T4.
//
// ***Uma única implementação da chamada: dois lugares que falam com o modelo não podem divergir.***
func OllamaGenerate(ctx context.Context, baseURL, modelo, prompt string, cliente *http.Client) (string, error) {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	if strings.TrimSpace(modelo) == "" {
		return "", fmt.Errorf("%w: modelo não informado", ErrFixerModelUnavailable)
	}
	if cliente == nil {
		cliente = &http.Client{Timeout: 3 * time.Minute}
	}
	corpo, err := json.Marshal(map[string]any{"model": modelo, "prompt": prompt, "stream": false,
		"options": map[string]any{"temperature": 0}})
	if err != nil {
		return "", err
	}
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/generate", bytes.NewReader(corpo))
	if err != nil {
		return "", err
	}
	requisicao.Header.Set("Content-Type", "application/json")
	resposta, err := cliente.Do(requisicao)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFixerModelUnavailable, err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: HTTP %d", ErrFixerModelUnavailable, resposta.StatusCode)
	}
	var decodificada struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resposta.Body).Decode(&decodificada); err != nil {
		return "", err
	}
	return decodificada.Response, nil
}
