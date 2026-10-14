package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// E-539 (Estagio 47 - Universal Artifact Drive): a triade completa - **versionar · comparar · restaurar**.
//
// O `E-537` entregou a busca e o **CAS** (`ProximaVersaoDeArtefacto`, o `expectedVersion` do `builder`). Faltavam as
// outras duas pernas. *Sem elas, «versionamento» e' um numero a subir num campo: nao se compara nada, nem se volta
// atras.*
//
// ***AS DUAS REGRAS QUE GOVERNAM ESTE FICHEIRO - E A SEGUNDA E' A QUE IMPORTA.***
//
//  1. **Comparar exige as DUAS versoes.** *Nao existe diferenca contra uma versao que nao existe.* E, mais fino:
//     **`MesmoConteudo` so' e' verdadeiro com `SHA256` igual** - *o tamanho nao decide conteudo. Dois textos de 40 bytes
//     sao o mesmo tamanho e podem nao partilhar uma letra; dizer «e' igual» por tamanho seria um falso negativo a espera
//     de acontecer.*
//
//  2. **Restaurar NAO reescreve a historia.** *Restaurar a versao 2 quando se esta' na 5 produz uma **versao 6** com o
//     conteudo da 2 - e **deixa a 5 onde esta'**.* ⇒ *esta e' a diferenca entre um sistema de versoes e um `UPDATE`: num,
//     o passado permanece consultavel; no outro, e' substituido e ninguem sabe o que houve.* *Um botao de restaurar que
//     apaga o que la' estava e' um botao que destroi precisamente aquilo que se ia recuperar.*
var (
	// ErrVersaoInexistente recusa comparar ou restaurar uma versao que nao esta' no indice.
	ErrVersaoInexistente = errors.New("artifact version does not exist")
	// ErrRestauroConflitante recusa restaurar quando a versao corrente ja' nao e' a esperada - *o CAS, na restauracao.*
	ErrRestauroConflitante = errors.New("artifact restore conflicts with the current version")
)

// ArtefactoDiferencaVersoes e' o que muda entre duas versoes. *Nao diz «igual» sem o provar pelo hash.*
type ArtefactoDiferencaVersoes struct {
	De            int    `json:"de"`
	Para          int    `json:"para"`
	MesmoConteudo bool   `json:"mesmo_conteudo"`
	MesmoTamanho  bool   `json:"mesmo_tamanho"`
	MudouTipo     bool   `json:"mudou_tipo"`
	TamanhoDe     int64  `json:"tamanho_de"`
	TamanhoPara   int64  `json:"tamanho_para"`
	MediaTypeDe   string `json:"media_type_de,omitempty"`
	MediaTypePara string `json:"media_type_para,omitempty"`
	// Razao explica a conclusao em texto, *para que um `false` nao obrigue a adivinhar o que mudou.*
	Razao string `json:"razao"`
}

// versaoDeArtefacto procura uma versao especifica, **dentro do tenant e da identidade**.
//
// *O tenant entra aqui e nao no chamador: uma funcao que aceita um `ArtifactManifest` sem confirmar o tenant e' uma
// funcao que pode ser chamada com o manifesto errado - e este e' o terceiro sitio, depois do `E-531` e do `E-537`, onde
// a mesma fronteira se afirma.*
func versaoDeArtefacto(indice []ArtifactManifest, tenantID, projectID, path string, versao int) (ArtifactManifest, bool) {
	for _, m := range indice {
		if !strings.EqualFold(strings.TrimSpace(m.TenantID), strings.TrimSpace(tenantID)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(m.ProjectID), strings.TrimSpace(projectID)) {
			continue
		}
		if strings.TrimSpace(m.Path) != strings.TrimSpace(path) {
			continue
		}
		if m.Version == versao {
			return m, true
		}
	}
	return ArtifactManifest{}, false
}

// ArtefactoCompararVersoes compara duas versoes de um artefacto, **no mesmo tenant**.
func ArtefactoCompararVersoes(indice []ArtifactManifest, tenantID, projectID, path string, de, para int) (ArtefactoDiferencaVersoes, error) {
	if strings.TrimSpace(tenantID) == "" {
		return ArtefactoDiferencaVersoes{}, ErrBuscaSemTenant
	}
	a, okA := versaoDeArtefacto(indice, tenantID, projectID, path, de)
	if !okA {
		return ArtefactoDiferencaVersoes{}, fmt.Errorf("%w: versao %d de %q", ErrVersaoInexistente, de, path)
	}
	b, okB := versaoDeArtefacto(indice, tenantID, projectID, path, para)
	if !okB {
		return ArtefactoDiferencaVersoes{}, fmt.Errorf("%w: versao %d de %q", ErrVersaoInexistente, para, path)
	}

	d := ArtefactoDiferencaVersoes{
		De: de, Para: para,
		TamanhoDe: a.Size, TamanhoPara: b.Size,
		MediaTypeDe: a.MediaType, MediaTypePara: b.MediaType,
		// ***`MesmoConteudo` SO' PELO HASH.*** *O tamanho nao decide conteudo: dois ficheiros de 1024 bytes podem nao
		// partilhar um unico byte.*
		MesmoConteudo: a.SHA256 == b.SHA256 && strings.TrimSpace(a.SHA256) != "",
		MesmoTamanho:  a.Size == b.Size,
		MudouTipo:     !strings.EqualFold(a.MediaType, b.MediaType),
	}
	switch {
	case d.MesmoConteudo:
		d.Razao = "mesmo conteudo (sha256 igual)"
	case d.MesmoTamanho:
		d.Razao = "tamanho igual mas CONTEUDO DIFERENTE (sha256 distinto) - o tamanho nao decide conteudo"
	default:
		d.Razao = "conteudo diferente"
	}
	return d, nil
}

// ArtefactoRestaurarVersao devolve um **novo** manifesto que repoe o conteudo de `restaurar`, na versao seguinte.
//
// ***NAO ALTERA O INDICE, E NAO APAGA NADA.*** *Devolve o manifesto a gravar; quem chama decide onde o guardar.* ⇒ *e' a
// assinatura que torna impossivel «restaurar» por substituicao: nao ha' parametro que aceite um indice para mutar.*
//
// `esperado` e' o CAS: a versao corrente que quem grava **julga** ser a actual. *Um `esperado` desactualizado recusa -
// sem isso, duas restauracoes concorrentes perdem uma, e alguem fica a achar que repôs o que nao repôs.*
func ArtefactoRestaurarVersao(indice []ArtifactManifest, tenantID, projectID, path string, restaurar, esperado int, quando time.Time) (ArtifactManifest, error) {
	if strings.TrimSpace(tenantID) == "" {
		return ArtifactManifest{}, ErrBuscaSemTenant
	}
	alvo, ok := versaoDeArtefacto(indice, tenantID, projectID, path, restaurar)
	if !ok {
		return ArtifactManifest{}, fmt.Errorf("%w: versao %d de %q", ErrVersaoInexistente, restaurar, path)
	}

	// O CAS, reutilizando a funcao do E-537 - *nao reimplemento a regra de conflito num segundo sitio.*
	proxima, err := ProximaVersaoDeArtefacto(indice, tenantID, projectID, path, esperado)
	if err != nil {
		return ArtifactManifest{}, fmt.Errorf("%w: %v", ErrRestauroConflitante, err)
	}

	// O novo manifesto: **conteudo do alvo**, identidade NOVA e versao `proxima`.
	novo := alvo
	novo.ID = fmt.Sprintf("%s-restaurado-%d", strings.TrimSpace(alvo.ID), proxima)
	novo.Version = proxima
	if !quando.IsZero() {
		novo.CreatedAt = quando
	}
	// *E a ACL do alvo e' preservada - restaurar nao e' uma forma de alargar permissoes.*
	return novo, nil
}

// ArtefactoHistorico devolve as versoes de um artefacto, **da mais recente para a mais antiga**.
//
// *Existe para que a UI nao tenha de ordenar nem de filtrar por tenant - a mesma fronteira, no quarto sitio.*
func ArtefactoHistorico(indice []ArtifactManifest, tenantID, projectID, path string) []ArtifactManifest {
	versoes := make([]ArtifactManifest, 0, 4)
	for _, m := range indice {
		if !strings.EqualFold(strings.TrimSpace(m.TenantID), strings.TrimSpace(tenantID)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(m.ProjectID), strings.TrimSpace(projectID)) {
			continue
		}
		if strings.TrimSpace(m.Path) != strings.TrimSpace(path) {
			continue
		}
		versoes = append(versoes, m)
	}
	for i := 1; i < len(versoes); i++ {
		for j := i; j > 0 && versoes[j].Version > versoes[j-1].Version; j-- {
			versoes[j], versoes[j-1] = versoes[j-1], versoes[j]
		}
	}
	return versoes
}
