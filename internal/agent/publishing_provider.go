package agent

// FASE 28.1 item 6 e adendo 28.7 do adendo V3: o `PublishingProvider`.
//
// A AUTORIZACAO VIGENTE PROIBE PUBLICAR, E ISSO DEFINE O DESENHO
//
//	"Esta fase nao autoriza publicar nada agora. A autorizacao vigente continua proibindo release,
//	 tag, loja, marketplace e publicacao externa."
//
//	"9. ... submeter, publicar, promover, responder review ou iniciar rollout"
//	   -- acao que exige "confirmacao humana no momento da acao"
//
// Portanto este provider implementa **`publish.preflight`** -- o nome que o proprio adendo 28.7 da'
// -- e **nao** implementa `publish.submit`. O adendo lista os dois, e a diferenca entre eles e' a
// diferenca entre verificar e agir.
//
//	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
//
// O QUE O PREFLIGHT RESPONDE, E O QUE ELE NAO RESPONDE
//
// Responde: o canal existe nesta maquina? a credencial necessaria esta' presente? o artefacto
// candidato existe e tem checksum? o gate de release foi satisfeito?
//
// **Nao responde** se a publicacao vai correr bem: isso exigiria publicar.
//
//	*Um preflight nao e' uma publicacao. E' a lista do que falta antes de pedir autorizacao.*
//
// E O `LOCAL_ONLY` NAO E' UM CANAL DE PUBLICACAO
//
// O adendo 26.13 e' explicito: *"\"Publicar localmente\" nao pode ser apresentado como deploy
// publico."* Por isso `LOCAL_ONLY` existe no catalogo com `publico: false` e `pronto` so' significa
// que o artefacto esta' no disco.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanalDePublicacao identifica onde um artefacto seria publicado.
type CanalDePublicacao string

const (
	// PublicacaoLocal e' o disco desta maquina. NAO e' deploy publico.
	PublicacaoLocal CanalDePublicacao = "LOCAL_ONLY"
	// PublicacaoGooglePlay e' o Play Console. Exige service account.
	PublicacaoGooglePlay CanalDePublicacao = "GOOGLE_PLAY"
	// PublicacaoAppStore e' o App Store Connect. Exige macOS e API key.
	PublicacaoAppStore CanalDePublicacao = "APP_STORE"
	// PublicacaoEASUpdate e' um canal EAS Update. Exige EAS autenticado e canal real.
	PublicacaoEASUpdate CanalDePublicacao = "EAS_UPDATE"
	// PublicacaoGitHubRelease e' uma release no GitHub. Exige token com permissao de release.
	PublicacaoGitHubRelease CanalDePublicacao = "GITHUB_RELEASE"
	// PublicacaoOCI e' um registry de imagens. Exige registry configurado.
	PublicacaoOCI CanalDePublicacao = "OCI_REGISTRY"
)

// CondicaoDoPreflight e' uma das verificacoes.
type CondicaoDoPreflight struct {
	Nome       string `json:"nome"`
	Satisfeita bool   `json:"satisfeita"`
	Detalhe    string `json:"detalhe"`
}

// PreflightDePublicacao e' o veredicto sobre UM canal.
type PreflightDePublicacao struct {
	Canal     CanalDePublicacao     `json:"canal"`
	Publico   bool                  `json:"publico"`
	Pronto    bool                  `json:"pronto"`
	Condicoes []CondicaoDoPreflight `json:"condicoes"`
	// RequerAprovacao e' sempre verdadeiro para canal publico: publicar exige autorizacao humana.
	RequerAprovacao bool `json:"requer_aprovacao"`
	// Publicado e' sempre `false`: nao ha' `submit` neste provider.
	Publicado  bool   `json:"publicado"`
	Explicacao string `json:"explicacao"`
	Aviso      string `json:"aviso"`
}

// PublishingProvider e' o contrato do item 6 do adendo 28.1.
//
// Tem UM metodo: `Preflight`. Nao ha' `Publicar()` -- nem para devolver "nao autorizado".
type PublishingProvider interface {
	// Preflight verifica as condicoes para publicar num canal. Nao publica.
	Preflight(ctx context.Context, canal CanalDePublicacao, caminhoDoArtefacto, caminhoDaCredencial string) (PreflightDePublicacao, error)
}

// ProviderDePublicacao faz o preflight dos canais.
type ProviderDePublicacao struct{}

// avisoDePreflight diz, no corpo, que nada foi publicado.
func avisoDePreflight() string {
	return "este preflight NAO publicou nada: `publicado` e' sempre false. Publicar, submeter, " +
		"promover ou iniciar rollout exige autorizacao humana especifica no momento da accao, e " +
		"este provider nao tem `submit`. `LOCAL_ONLY` nao e' deploy publico."
}

// descricaoDoCanal devolve o que o canal exige e se e' publico.
func descricaoDoCanal(canal CanalDePublicacao) (publico bool, requisito string, conhecido bool) {
	switch canal {
	case PublicacaoLocal:
		return false, "", true
	case PublicacaoGooglePlay:
		return true, "service account JSON com acesso ao Play Console", true
	case PublicacaoAppStore:
		return true, "host macOS com Xcode e App Store Connect API key em uso", true
	case PublicacaoEASUpdate:
		return true, "EAS autenticado e um canal real com runtime compativel", true
	case PublicacaoGitHubRelease:
		return true, "token com permissao de release no repositorio", true
	case PublicacaoOCI:
		return true, "registry OCI configurado e autenticado", true
	default:
		return false, "", false
	}
}

// Preflight verifica as condicoes e devolve o veredicto.
func (ProviderDePublicacao) Preflight(
	ctx context.Context,
	canal CanalDePublicacao,
	caminhoDoArtefacto string,
	caminhoDaCredencial string,
) (PreflightDePublicacao, error) {
	if err := ctx.Err(); err != nil {
		return PreflightDePublicacao{}, err
	}
	resultado := PreflightDePublicacao{
		Canal:     canal,
		Publicado: false,
		Aviso:     avisoDePreflight(),
	}

	publico, requisito, conhecido := descricaoDoCanal(canal)
	resultado.Publico = publico
	if !conhecido {
		resultado.Explicacao = fmt.Sprintf(
			"canal desconhecido: %q; este provider nao afirma que serve nem que nao serve", canal,
		)
		return resultado, fmt.Errorf("%w: canal %q", ErrRollbackNaoSuportado, canal)
	}

	// Canal publico exige aprovacao humana, sempre -- mesmo que o preflight passe inteiro.
	if publico {
		resultado.RequerAprovacao = true
	}

	// 1. O artefacto existe e e' integro?
	resultado.Condicoes = append(resultado.Condicoes, verificarArtefactoDoPreflight(caminhoDoArtefacto))

	// 2. A credencial existe? Para canal local nao ha' credencial.
	if publico {
		resultado.Condicoes = append(resultado.Condicoes,
			verificarCredencialDoPreflight(caminhoDaCredencial, requisito))
	}

	// 3. O canal esta' disponivel nesta maquina?
	resultado.Condicoes = append(resultado.Condicoes, CondicaoDoPreflight{
		Nome:       "canal disponivel",
		Satisfeita: !publico, // canal publico NUNCA esta' "disponivel" para executar nesta missao
		Detalhe:    detalheDoCanal(publico, requisito),
	})

	// O veredicto: pronto apenas se TODAS as condicoes passarem.
	resultado.Pronto = true
	var falhas []string
	for _, condicao := range resultado.Condicoes {
		if !condicao.Satisfeita {
			resultado.Pronto = false
			falhas = append(falhas, condicao.Nome+": "+condicao.Detalhe)
		}
	}
	if resultado.Pronto {
		if publico {
			resultado.Explicacao = "todas as condicoes verificaveis passaram; publicar exige " +
				"autorizacao humana e este provider nao a executa"
		} else {
			resultado.Explicacao = "o artefacto existe e e' integro; `LOCAL_ONLY` significa que esta' " +
				"no disco, NAO que foi publicado"
		}
		return resultado, nil
	}
	resultado.Explicacao = "preflight NAO passou: " + strings.Join(falhas, "; ")
	return resultado, nil
}

// detalheDoCanal explica a disponibilidade sem inventar.
func detalheDoCanal(publico bool, requisito string) string {
	if !publico {
		return "canal local: o artefacto fica no disco desta maquina"
	}
	return "canal publico: exige " + requisito + "; a autorizacao vigente proibe publicar nesta " +
		"missao, por isso o canal nao e' declarado disponivel"
}

// verificarArtefactoDoPreflight mede o artefacto candidato.
func verificarArtefactoDoPreflight(caminho string) CondicaoDoPreflight {
	condicao := CondicaoDoPreflight{Nome: "artefacto existe e e' integro"}
	if strings.TrimSpace(caminho) == "" {
		condicao.Detalhe = "nenhum artefacto indicado; sem artefacto nao ha' o que publicar"
		return condicao
	}
	absoluto, err := filepath.Abs(caminho)
	if err != nil {
		condicao.Detalhe = fmt.Sprintf("resolver %q: %v", caminho, err)
		return condicao
	}
	estado, err := os.Stat(absoluto)
	if err != nil {
		condicao.Detalhe = fmt.Sprintf("o artefacto %q nao existe", absoluto)
		return condicao
	}
	if estado.IsDir() {
		condicao.Detalhe = fmt.Sprintf("%q e' um directorio", absoluto)
		return condicao
	}
	if estado.Size() == 0 {
		condicao.Detalhe = fmt.Sprintf("%q tem 0 bytes", absoluto)
		return condicao
	}
	soma, err := sha256DoFicheiro(absoluto)
	if err != nil {
		condicao.Detalhe = fmt.Sprintf("medir %q: %v", absoluto, err)
		return condicao
	}
	condicao.Satisfeita = true
	condicao.Detalhe = fmt.Sprintf("%d bytes, sha256 %s", estado.Size(), soma)
	return condicao
}

// verificarCredencialDoPreflight verifica a EXISTENCIA da credencial, sem a abrir.
//
//	*O valor privado nao passa por este processo: a existencia e o tamanho bastam para o preflight.*
func verificarCredencialDoPreflight(caminho, requisito string) CondicaoDoPreflight {
	condicao := CondicaoDoPreflight{Nome: "credencial presente"}
	if strings.TrimSpace(caminho) == "" {
		condicao.Detalhe = "nenhuma credencial indicada; o canal exige " + requisito
		return condicao
	}
	absoluto, err := filepath.Abs(caminho)
	if err != nil {
		condicao.Detalhe = fmt.Sprintf("resolver %q: %v", caminho, err)
		return condicao
	}
	estado, err := os.Stat(absoluto)
	if err != nil {
		condicao.Detalhe = fmt.Sprintf("a credencial %q nao existe", absoluto)
		return condicao
	}
	if estado.IsDir() {
		condicao.Detalhe = fmt.Sprintf("%q e' um directorio", absoluto)
		return condicao
	}
	condicao.Satisfeita = true
	// Sem nome de ficheiro e sem conteudo: so' o tamanho.
	condicao.Detalhe = fmt.Sprintf("presente (%d bytes); conteudo NAO lido", estado.Size())
	return condicao
}

// CanaisDePublicacao lista o catalogo completo, com a classificacao honesta.
func CanaisDePublicacao() map[CanalDePublicacao]bool {
	return map[CanalDePublicacao]bool{
		PublicacaoLocal:         false,
		PublicacaoGooglePlay:    true,
		PublicacaoAppStore:      true,
		PublicacaoEASUpdate:     true,
		PublicacaoGitHubRelease: true,
		PublicacaoOCI:           true,
	}
}
