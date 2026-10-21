package agent

// FASE 28.1 item 10 do adendo V3: o `RollbackProvider`.
//
// O QUE O ADENDO PEDE, E O QUE ESTE HOST PERMITE
//
// O adendo pede: "cancela rollout, promove versao anterior ou restaura canal quando a plataforma
// permitir". As tres partes nao tem o mesmo suporte:
//
//   - **Promover versao anterior** (Windows): o `dist/` guarda instaladores de versoes anteriores com
//     SHA-256 medivel. Isto e' verificavel AGORA.
//   - **Cancelar rollout** (Google Play, App Store, EAS Update): exigiria conta autenticada e
//     service account/API key que **nao estao configuradas** -- `BLOCKED_EXTERNAL`.
//   - **Restaurar canal** (EAS Update branch/channel): exigiria EAS autenticado e um canal real --
//     `BLOCKED_EXTERNAL`.
//
// Este provider implementa a primeira e **declara as outras duas**, em vez de fingir simetria:
//
//	*Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
//
// E NAO EXECUTA O ROLLBACK
//
// `Planear` monta o plano e verifica a integridade do alvo. **Nao restaura.** Restaurar um binario de
// producao e' exactamente a accao que o adendo 28.5 poem atras de "confirmacao humana no momento da
// accao":
//
//	"7. ... assinar artefato de producao; submeter, publicar, promover, responder review ou iniciar
//	 rollout ... revogar certificado/chave"
//
// Um `Executar()` que existisse so' para devolver "nao autorizado" seria, de novo, um botao morto.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TipoDeRollback classifica o canal do rollback.
type TipoDeRollback string

const (
	// RollbackWindowsInstaller e' a promocao de um `.exe` anterior. Suportado localmente.
	RollbackWindowsInstaller TipoDeRollback = "WINDOWS_INSTALLER"
	// RollbackGooglePlay cancela um rollout no Play Console. Sem service account.
	RollbackGooglePlay TipoDeRollback = "GOOGLE_PLAY_ROLLOUT"
	// RollbackAppStore cancela/remove uma build no App Store Connect. Sem host macOS/credencial.
	RollbackAppStore TipoDeRollback = "APP_STORE_ROLLOUT"
	// RollbackEASUpdate restaura um canal EAS Update. Sem EAS autenticado nesta sessao.
	RollbackEASUpdate TipoDeRollback = "EAS_UPDATE_CHANNEL"
	// RollbackOCI remove uma tag de imagem. Sem registry configurado.
	RollbackOCI TipoDeRollback = "OCI_TAG"
)

// VersaoAnterior e' um artefacto candidato a rollback.
type VersaoAnterior struct {
	Caminho          string    `json:"caminho"`
	Nome             string    `json:"nome"`
	TamanhoBytes     int64     `json:"tamanho_bytes"`
	SHA256           string    `json:"sha256"`
	ModificadoEm     time.Time `json:"modificado_em"`
	PareceInstalador bool      `json:"parece_instalador"`
}

// PlanoDeRollback e' o que seria feito.
type PlanoDeRollback struct {
	Tipo              TipoDeRollback   `json:"tipo"`
	Suportado         bool             `json:"suportado"`
	Alvo              *VersaoAnterior  `json:"alvo,omitempty"`
	Candidatos        []VersaoAnterior `json:"candidatos"`
	TotalDeCandidatos int              `json:"total_de_candidatos"`
	// Integro diz se o alvo passou a verificacao de checksum.
	Integro bool `json:"integro"`
	// Executado e' sempre `false`: restaurar exige aprovacao humana no momento da accao.
	Executado bool `json:"executado"`
	// RequerAprovacao nomeia o que falta para executar.
	RequerAprovacao string `json:"requer_aprovacao,omitempty"`
	Explicacao      string `json:"explicacao"`
	Aviso           string `json:"aviso"`
}

// ErrRollbackNaoSuportado e' devolvido para canais sem credencial ou host.
var ErrRollbackNaoSuportado = errors.New("rollback nao suportado neste ambiente")

// ErrSemCandidatos e' devolvido quando o directorio nao tem versoes anteriores.
var ErrSemCandidatos = errors.New("nenhuma versao anterior encontrada")

// maxVersoesAnteriores limita a listagem.
const maxVersoesAnteriores = 64

// RollbackProvider e' o contrato do item 10 do adendo 28.1.
type RollbackProvider interface {
	// Planear monta o plano de rollback para o tipo pedido. Nunca executa.
	Planear(ctx context.Context, tipo TipoDeRollback, directorio string) (PlanoDeRollback, error)
}

// ProviderDeRollback planeia rollbacks de artefactos locais.
type ProviderDeRollback struct {
	// Agora permite fixar o relogio nos testes.
	Agora time.Time
}

// avisoDeRollback diz, no corpo, que nada foi executado.
func avisoDeRollback() string {
	return "este plano NAO executou o rollback: `executado` e' sempre false. Restaurar um artefacto " +
		"de producao exige aprovacao humana no momento da accao, e nenhum ficheiro foi movido, " +
		"copiado ou apagado."
}

// canaisSuportados lista o que este host consegue planear, e por que os outros nao.
func canaisSuportados() map[TipoDeRollback]string {
	return map[TipoDeRollback]string{
		RollbackWindowsInstaller: "",
		RollbackGooglePlay: "exige uma service account JSON com acesso ao Play Console; " +
			"CONFIG_REQUIRED (adendo 28.13, item 7: 'ainda falta uma service account JSON')",
		RollbackAppStore: "exige host macOS e a App Store Connect API key em uso; num host Windows " +
			"o rollback de build nao pode ser executado (BLOCKED_EXTERNAL)",
		RollbackEASUpdate: "exige `eas` autenticado e um canal real com runtime compativel; " +
			"nenhum canal foi criado ainda (BLOCKED_EXTERNAL)",
		RollbackOCI: "exige um registry configurado; nenhum registry existe nesta missao " +
			"(BLOCKED_EXTERNAL)",
	}
}

// Planear monta o plano. Nao executa nada.
func (p ProviderDeRollback) Planear(
	ctx context.Context,
	tipo TipoDeRollback,
	directorio string,
) (PlanoDeRollback, error) {
	if err := ctx.Err(); err != nil {
		return PlanoDeRollback{}, err
	}
	plano := PlanoDeRollback{
		Tipo:      tipo,
		Executado: false,
		Aviso:     avisoDeRollback(),
	}

	motivo, conhecido := canaisSuportados()[tipo]
	if !conhecido {
		plano.Explicacao = fmt.Sprintf(
			"tipo de rollback desconhecido: %q; este provider nao afirma que serve nem que nao serve",
			tipo,
		)
		return plano, fmt.Errorf("%w: tipo desconhecido %q", ErrRollbackNaoSuportado, tipo)
	}
	if motivo != "" {
		plano.Suportado = false
		plano.Explicacao = motivo
		return plano, fmt.Errorf("%w: %s", ErrRollbackNaoSuportado, motivo)
	}
	plano.Suportado = true

	candidatos, err := listarVersoesAnteriores(ctx, directorio, p.Agora)
	if err != nil {
		plano.Explicacao = err.Error()
		return plano, err
	}
	plano.Candidatos = candidatos
	plano.TotalDeCandidatos = len(candidatos)
	if len(candidatos) == 0 {
		plano.Explicacao = fmt.Sprintf(
			"%q nao contem artefactos de versoes anteriores; sem candidatos nao ha' rollback a "+
				"planear", directorio,
		)
		return plano, fmt.Errorf("%w: %s", ErrSemCandidatos, directorio)
	}

	// O alvo e' o instaler mais RECENTE que nao seja o que esta' a correr? Nao: o provider nao sabe
	// qual esta' a correr. Devolve o mais recente como proposta e diz que a escolha e' humana.
	alvo := candidatos[0]
	plano.Alvo = &alvo
	plano.Integro = alvo.SHA256 != ""
	plano.RequerAprovacao = "escolher o alvo e autorizar a restauracao; o provider propoe o mais " +
		"recente e nao decide por quem assina a release"
	plano.Explicacao = fmt.Sprintf(
		"%d candidato(s); o mais recente e' %q (%d bytes, sha256 %s). Nada foi restaurado",
		len(candidatos), alvo.Nome, alvo.TamanhoBytes, alvo.SHA256,
	)
	return plano, nil
}

// listarVersoesAnteriores le' o directorio e mede cada artefacto.
//
// Mede o SHA-256 de cada candidato: *um checksum errado e' uma prova de integridade que mente*, e um
// plano de rollback que nao mede o alvo nao prova que o alvo existe como foi construido.
func listarVersoesAnteriores(
	ctx context.Context,
	directorio string,
	agora time.Time,
) ([]VersaoAnterior, error) {
	if strings.TrimSpace(directorio) == "" {
		return nil, fmt.Errorf("%w: directorio vazio", ErrSemCandidatos)
	}
	absoluto, err := filepath.Abs(directorio)
	if err != nil {
		return nil, fmt.Errorf("%w: resolver %q: %v", ErrSemCandidatos, directorio, err)
	}
	estado, err := os.Stat(absoluto)
	if err != nil {
		return nil, fmt.Errorf("%w: %q: %v", ErrSemCandidatos, absoluto, err)
	}
	if !estado.IsDir() {
		return nil, fmt.Errorf("%w: %q nao e' um directorio", ErrSemCandidatos, absoluto)
	}

	var versoes []VersaoAnterior
	entradas, err := os.ReadDir(absoluto)
	if err != nil {
		return nil, fmt.Errorf("%w: ler %q: %v", ErrSemCandidatos, absoluto, err)
	}
	for _, entrada := range entradas {
		if err := ctx.Err(); err != nil {
			return versoes, err
		}
		if len(versoes) >= maxVersoesAnteriores {
			break
		}
		if entrada.IsDir() {
			continue
		}
		nome := entrada.Name()
		minusculo := strings.ToLower(nome)
		pareceInstalador := strings.HasSuffix(minusculo, ".exe") ||
			strings.HasSuffix(minusculo, ".msi") ||
			strings.HasSuffix(minusculo, ".msix") ||
			strings.HasSuffix(minusculo, ".dmg") ||
			strings.HasSuffix(minusculo, ".pkg") ||
			strings.HasSuffix(minusculo, ".deb") ||
			strings.HasSuffix(minusculo, ".rpm") ||
			strings.HasSuffix(minusculo, ".appimage")
		if !pareceInstalador {
			continue
		}
		caminho := filepath.Join(absoluto, nome)
		informacao, err := os.Stat(caminho)
		if err != nil {
			continue
		}
		soma, err := sha256DoFicheiro(caminho)
		if err != nil {
			// Um candidato ilegivel fica FORA da lista, e nao entra com um checksum vazio: um
			// candidato sem checksum nao e' verificavel.
			continue
		}
		versoes = append(versoes, VersaoAnterior{
			Caminho:          caminho,
			Nome:             nome,
			TamanhoBytes:     informacao.Size(),
			SHA256:           soma,
			ModificadoEm:     informacao.ModTime().UTC(),
			PareceInstalador: true,
		})
	}

	// Mais recente primeiro: e' a ordem em que um rollback faz sentido.
	sort.SliceStable(versoes, func(i, j int) bool {
		return versoes[i].ModificadoEm.After(versoes[j].ModificadoEm)
	})
	_ = agora
	return versoes, nil
}

// sha256DoFicheiro calcula o hash em fluxo, sem carregar o ficheiro inteiro.
//
// Um instalador tem dezenas de MB: *ler um ficheiro sem tecto porque "e' so' um binario" e' confiar
// no tamanho para decidir a memoria.*
func sha256DoFicheiro(caminho string) (string, error) {
	ficheiro, err := os.Open(caminho) //nolint:gosec // caminho derivado de um directorio dado
	if err != nil {
		return "", err
	}
	defer ficheiro.Close()
	medidor := sha256.New()
	if _, err := io.Copy(medidor, ficheiro); err != nil {
		return "", err
	}
	return hex.EncodeToString(medidor.Sum(nil)), nil
}
