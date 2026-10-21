package agent

// FASE 28.1 item 7 e adendo 28.5 do adendo V3: o `CredentialProvider`.
//
// A REGRA QUE DEFINE ESTE FICHEIRO
//
// O adendo 28.13 e' explicito:
//
//	"Nunca leia ou imprima o conteudo privado das chaves para 'provar' que existem. Valide por
//	 metadados, fingerprint, ferramenta oficial e operacao criptografica controlada."
//
// Este provider **abre certificados publicos** (`x509` sabe le-los) e **apenas faz `os.Stat`** em
// tudo o que pode conter segredo: `.p12`, `.jks`, `.keystore`, `.p8`, `.dpapi`, e o proprio
// `.mobileprovision` (que e' um CMS binario com o perfil dentro).
//
//	*Nao leia o segredo para provar que ele existe: o tamanho e a data ja' provam que o ficheiro
//	esta' la', e a leitura e' o risco.*
//
// PORQUE ISSO E' UMA DECISAO E NAO UMA LIMITACAO
//
// Um inventario que abrisse o `.p12` para ler o sujeito teria, no processo do servidor, a chave
// privada em memoria. O objectivo deste provider e' responder "o que existe, de que tipo, serve para
// que ambiente, e quando expira" -- e nenhuma dessas perguntas exige a chave.
//
// O QUE ELE NAO FAZ
//
// Nao valida a senha. Nao exporta nada. Nao copia nada. Nao diz se o `.p12` corresponde ao `.cer` --
// isso exige abrir o `.p12`, e fica declarado como NAO VERIFICADO em vez de ser afirmado.
//
//	*Aprovado significa "passou os gates que consegui aplicar", nao "todos os gates existiram".*

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TipoDeCredencial classifica um ficheiro do kit.
type TipoDeCredencial string

const (
	// CredencialCertificadoX509 e' um certificado publico. Pode ser lido.
	CredencialCertificadoX509 TipoDeCredencial = "CERTIFICADO_X509"
	// CredencialPerfilDeProvisionamento e' um `.mobileprovision` (CMS binario). NAO e' aberto.
	CredencialPerfilDeProvisionamento TipoDeCredencial = "PERFIL_DE_PROVISIONAMENTO"
	// CredencialChavePrivada e' uma chave (`.p8`, `.pem` com chave). NAO e' aberta.
	CredencialChavePrivada TipoDeCredencial = "CHAVE_PRIVADA"
	// CredencialKeystore e' um `.jks`/`.keystore`/`.p12`. NAO e' aberto.
	CredencialKeystore TipoDeCredencial = "KEYSTORE"
	// CredencialSenhaProtegida e' um segredo cifrado por DPAPI. NAO e' aberto.
	CredencialSenhaProtegida TipoDeCredencial = "SENHA_PROTEGIDA"
	// CredencialChecksums e' um manifesto de hashes. Publico.
	CredencialChecksums TipoDeCredencial = "CHECKSUMS"
	// CredencialDocumentacao e' texto de apoio. Publico.
	CredencialDocumentacao TipoDeCredencial = "DOCUMENTACAO"
	// CredencialDesconhecida e' um ficheiro que este provider nao sabe classificar.
	CredencialDesconhecida TipoDeCredencial = "DESCONHECIDA"
)

// conteudoLido diz se o tipo permite abrir o ficheiro.
//
// A funcao existe para que a regra fique num so' lugar: se amanha' alguem acrescentar um tipo, tem de
// decidir explicitamente se ele e' legivel.
func conteudoLido(tipo TipoDeCredencial) bool {
	switch tipo {
	case CredencialCertificadoX509, CredencialChecksums, CredencialDocumentacao:
		return true
	default:
		// Tudo o que pode conter segredo cai aqui, incluindo `DESCONHECIDA`: na duvida, nao se abre.
		return false
	}
}

// Credencial e' o que o inventario sabe sobre um ficheiro.
//
// Nenhum campo pode conter material secreto. O `FingerprintSHA256` e' do CONTEUDO do certificado
// publico, nao de uma chave.
type Credencial struct {
	CaminhoRelativo string           `json:"caminho_relativo"`
	Tipo            TipoDeCredencial `json:"tipo"`
	TamanhoBytes    int64            `json:"tamanho_bytes"`
	ModificadoEm    time.Time        `json:"modificado_em"`
	// ConteudoLido diz se este provider abriu o ficheiro. `false` significa que so' fez `os.Stat`.
	ConteudoLido bool `json:"conteudo_lido"`
	// FingerprintSHA256 e' o hash do DER do certificado, quando lido. Vazio nos outros tipos.
	FingerprintSHA256 string `json:"fingerprint_sha256,omitempty"`
	Sujeito           string `json:"sujeito,omitempty"`
	Emissor           string `json:"emissor,omitempty"`
	ValidoDe          string `json:"valido_de,omitempty"`
	ValidoAte         string `json:"valido_ate,omitempty"`
	Expirado          bool   `json:"expirado,omitempty"`
	// Nota explica o que NAO foi verificado, quando aplicavel.
	Nota string `json:"nota,omitempty"`
}

// InventarioDeCredenciais e' o resultado de varrer um kit.
type InventarioDeCredenciais struct {
	Raiz              string       `json:"raiz"`
	MedidoEm          time.Time    `json:"medido_em"`
	Credenciais       []Credencial `json:"credenciais"`
	Total             int          `json:"total"`
	Lidos             int          `json:"lidos"`
	NaoLidos          int          `json:"nao_lidos"`
	Certificados      int          `json:"certificados"`
	Expirados         int          `json:"expirados"`
	SegredosPresentes []string     `json:"segredos_presentes"`
	// NaoLegiveis conta o que NAO pode ser medido. Nunca e' omitido do resultado.
	NaoLegiveis int `json:"nao_legiveis"`
	// Avisos regista o que falhou, para o inventario nao mentir por omissao:
	//
	//	*Um erro que nao aborta nao e' um erro que nao aconteceu.*
	Avisos []string `json:"avisos,omitempty"`
	Aviso  string   `json:"aviso"`
}

// ErrKitInexistente e' devolvido quando a raiz nao existe ou nao e' um directorio.
var ErrKitInexistente = fmt.Errorf("raiz do kit de credenciais inacessivel")

// maxFicheirosDoKit limita a varredura.
//
//	*Uma varredura de arvore disparada por HTTP sem limite e' uma pausa que o cliente pode pedir ao
//	servidor e que o servidor nao pode recusar depois de comecar.*
const maxFicheirosDoKit = 512

// maxBytesDeCertificado limita a leitura de um certificado publico. Um `.cer` real tem poucos KB.
const maxBytesDeCertificado = 64 * 1024

// CredentialProvider e' o contrato do item 7 do adendo 28.1.
type CredentialProvider interface {
	// Inventariar percorre a raiz e devolve o que existe, SEM abrir o que pode ser segredo.
	Inventariar(ctx context.Context, raiz string) (InventarioDeCredenciais, error)
}

// ProviderDeCredenciais inventaria um kit local com metadados seguros.
type ProviderDeCredenciais struct {
	// Agora permite fixar o relogio nos testes. Zero significa `time.Now()`.
	Agora time.Time
}

// classificar decide o tipo pelo NOME do ficheiro.
//
// Pelo nome e nao pelo conteudo, porque classificar pelo conteudo exigiria abrir -- que e' exactamente
// o que este provider recusa fazer nos segredos.
func classificar(nome string) TipoDeCredencial {
	minusculo := strings.ToLower(nome)
	switch {
	case strings.HasSuffix(minusculo, ".p12"), strings.HasSuffix(minusculo, ".pfx"),
		strings.HasSuffix(minusculo, ".jks"), strings.HasSuffix(minusculo, ".keystore"):
		return CredencialKeystore
	case strings.HasSuffix(minusculo, ".dpapi"):
		return CredencialSenhaProtegida
	case strings.HasSuffix(minusculo, ".p8"),
		strings.HasSuffix(minusculo, ".key"):
		return CredencialChavePrivada
	case strings.HasSuffix(minusculo, ".mobileprovision"),
		strings.HasSuffix(minusculo, ".provisionprofile"):
		return CredencialPerfilDeProvisionamento
	case strings.HasSuffix(minusculo, ".cer"), strings.HasSuffix(minusculo, ".crt"),
		strings.HasSuffix(minusculo, ".der"), strings.HasSuffix(minusculo, ".pem"):
		return CredencialCertificadoX509
	case strings.HasPrefix(minusculo, "sha256sums"), strings.HasSuffix(minusculo, ".sha256"):
		return CredencialChecksums
	case strings.HasSuffix(minusculo, ".txt"), strings.HasSuffix(minusculo, ".md"),
		strings.HasSuffix(minusculo, ".json"), strings.HasSuffix(minusculo, ".inf"),
		strings.HasSuffix(minusculo, ".csr"):
		return CredencialDocumentacao
	default:
		return CredencialDesconhecida
	}
}

// Inventariar percorre a raiz e devolve metadados seguros de cada ficheiro.
func (p ProviderDeCredenciais) Inventariar(
	ctx context.Context,
	raiz string,
) (InventarioDeCredenciais, error) {
	if err := ctx.Err(); err != nil {
		return InventarioDeCredenciais{}, err
	}
	absoluta, err := filepath.Abs(raiz)
	if err != nil {
		return InventarioDeCredenciais{}, fmt.Errorf("%w: resolver %q: %v", ErrKitInexistente, raiz, err)
	}
	info, err := os.Stat(absoluta)
	if err != nil {
		return InventarioDeCredenciais{}, fmt.Errorf("%w: %q: %v", ErrKitInexistente, absoluta, err)
	}
	if !info.IsDir() {
		return InventarioDeCredenciais{}, fmt.Errorf("%w: %q nao e' um directorio", ErrKitInexistente, absoluta)
	}

	agora := p.Agora
	if agora.IsZero() {
		agora = time.Now()
	}

	inventario := InventarioDeCredenciais{
		Raiz:     absoluta,
		MedidoEm: agora,
		Aviso: "este inventario NAO le' chaves privadas, keystores, senhas protegidas nem perfis de " +
			"provisionamento: para esses ficheiros so' o tamanho e a data sao medidos. Nao valida " +
			"senhas, nao confirma que o `.p12` corresponde ao `.cer`, e nao exporta nada.",
	}

	err = filepath.WalkDir(absoluta, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			// Um erro de leitura num subdirectorio nao aborta o inventario inteiro -- mas tambem nao
			// desaparece em silencio:
			//
			//	*Um erro que nao aborta nao e' um erro que nao aconteceu.*
			//
			// Fica registado em `Avisos` e o inventario continua. Devolver `nil` sem registar seria
			// um inventario que mente por omissao sobre o que nao conseguiu ver.
			inventario.Avisos = append(inventario.Avisos, fmt.Sprintf(
				"nao foi possivel entrar em %q: %v", caminho, err,
			))
			inventario.NaoLegiveis++
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entrada.IsDir() {
			return nil
		}
		if inventario.Total >= maxFicheirosDoKit {
			return filepath.SkipAll
		}
		relativo, err := filepath.Rel(absoluta, caminho)
		if err != nil {
			relativo = filepath.Base(caminho)
		}
		credencial := Credencial{
			CaminhoRelativo: relativo,
			Tipo:            classificar(entrada.Name()),
		}
		estado, err := os.Stat(caminho)
		if err != nil {
			// Mesma regra: registar e seguir, nunca engolir.
			inventario.Avisos = append(inventario.Avisos, fmt.Sprintf(
				"nao foi possivel medir %q: %v", relativo, err,
			))
			inventario.NaoLegiveis++
			return nil
		}
		credencial.TamanhoBytes = estado.Size()
		credencial.ModificadoEm = estado.ModTime().UTC()

		if conteudoLido(credencial.Tipo) {
			preencherComConteudo(&credencial, caminho, agora)
		} else {
			credencial.Nota = "conteudo NAO lido: este tipo pode conter segredo; apenas metadados"
			inventario.SegredosPresentes = append(inventario.SegredosPresentes, relativo)
		}

		inventario.Credenciais = append(inventario.Credenciais, credencial)
		inventario.Total++
		if credencial.ConteudoLido {
			inventario.Lidos++
		} else {
			inventario.NaoLidos++
		}
		// Conta apenas os certificados VALIDOS. Um `.cer` que nao e' um certificado fica no
		// inventario com o `Nota` a dizer por que, mas nao entra neste numero:
		//
		//	*Um relatorio que obriga a somar a mao e' um relatorio para quem ja' sabia a resposta.*
		if credencial.Tipo == CredencialCertificadoX509 && credencial.FingerprintSHA256 != "" {
			inventario.Certificados++
		}
		if credencial.Expirado {
			inventario.Expirados++
		}
		return nil
	})
	if err != nil {
		return inventario, err
	}

	sort.Slice(inventario.Credenciais, func(i, j int) bool {
		return inventario.Credenciais[i].CaminhoRelativo < inventario.Credenciais[j].CaminhoRelativo
	})
	sort.Strings(inventario.SegredosPresentes)
	return inventario, nil
}

// preencherComConteudo le' um certificado publico e preenche os metadados.
//
// Se o ficheiro nao for um certificado (por exemplo um `.pem` que afinal e' outra coisa), a credencial
// fica com a nota do erro e NAO com um fingerprint inventado.
func preencherComConteudo(credencial *Credencial, caminho string, agora time.Time) {
	if credencial.Tipo != CredencialCertificadoX509 {
		credencial.ConteudoLido = true
		return
	}
	bruto, err := lerLimitado(caminho, maxBytesDeCertificado)
	if err != nil {
		credencial.Nota = fmt.Sprintf("nao foi possivel ler: %v", err)
		return
	}
	certificado, err := certificadoDeBytes(bruto)
	if err != nil {
		credencial.Nota = fmt.Sprintf("o ficheiro tem extensao de certificado mas nao e' um: %v", err)
		credencial.ConteudoLido = true
		return
	}
	soma := sha256.Sum256(certificado.Raw)
	credencial.ConteudoLido = true
	credencial.FingerprintSHA256 = hex.EncodeToString(soma[:])
	credencial.Sujeito = certificado.Subject.CommonName
	credencial.Emissor = certificado.Issuer.CommonName
	credencial.ValidoDe = certificado.NotBefore.UTC().Format(time.RFC3339)
	credencial.ValidoAte = certificado.NotAfter.UTC().Format(time.RFC3339)
	credencial.Expirado = agora.After(certificado.NotAfter)
}

// lerLimitado le' no maximo `limite` bytes.
//
//	*Ler um ficheiro sem tecto porque "e' um certificado" e' confiar na extensao para decidir o
//	tamanho.*
func lerLimitado(caminho string, limite int64) ([]byte, error) {
	ficheiro, err := os.Open(caminho) //nolint:gosec // o caminho vem de uma varredura da raiz dada
	if err != nil {
		return nil, err
	}
	defer ficheiro.Close()
	tampao := make([]byte, limite)
	lidos, err := ficheiro.Read(tampao)
	if err != nil && lidos == 0 {
		return nil, err
	}
	return tampao[:lidos], nil
}

// certificadoDeBytes aceita PEM ou DER.
func certificadoDeBytes(bruto []byte) (*x509.Certificate, error) {
	if bloco, _ := pem.Decode(bruto); bloco != nil && bloco.Type == "CERTIFICATE" {
		return x509.ParseCertificate(bloco.Bytes)
	}
	return x509.ParseCertificate(bruto)
}
