package agent

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// E-537 (Estagio 47 - Universal Artifact Drive): a BUSCA, com isolamento por tenant.
//
// ***A AUDITORIA MEDIU (nao suposto).*** `internal/agent` + `server` **nao tem nenhuma funcao de busca de
// artefactos** - zero. E o `builder.go` **ja' tem o padrao a seguir**: `ApplyVisualComponentsCAS(expectedVersion)`,
// `UndoCAS`, `RedoCAS` e **`RollbackPublish(organizationID, version)`**. ⇒ *nao crio um segundo modelo de versionamento:
// sigo o que existe, no eixo que falta - o do ARTEFACTO.*
//
// ***A REGRA QUE GOVERNA ESTE FICHEIRO, E QUE NAO E' NEGOCIAVEL.***
//
// **A busca filtra por visibilidade ANTES de qualquer outra coisa.** *Nao existe caminho que liste um artefacto e so'
// depois decida se o sujeito podia vel-o - porque esse caminho, no dia em que alguem esquecer a segunda metade, e' uma
// fuga entre organizacoes.* ⇒ o primeiro passo e' `ArtifactVisivelPara` (E-531), e o resto opera sobre o que sobrou.
//
// ***E A SEGUNDA REGRA:*** `BuscarArtefactos` **exige** tenant e sujeito, e **recusa** sem eles. *Uma busca «sem
// tenant» nao e' uma busca ampla: e' a ausencia de fronteira - e o complemento proibe exactamente essa.*
var (
	// ErrBuscaSemTenant recusa uma busca que nao declara de que organizacao fala.
	ErrBuscaSemTenant = errors.New("artifact search requires a tenant")
	// ErrBuscaSemSujeito recusa uma busca sem sujeito: *sem saber QUEM procura, nao ha' o que autorizar.*
	ErrBuscaSemSujeito = errors.New("artifact search requires a subject")
)

// ArtifactQuery e' o pedido de busca. *Todos os campos sao filtros ADITIVOS sobre o conjunto ja' visivel.*
type ArtifactQuery struct {
	// TenantID e Sujeito sao OBRIGATORIOS: sao a fronteira da busca.
	TenantID string
	Sujeito  string
	// Texto casa (case-insensitive) em `Name` e `Path`. Vazio significa «sem filtro de texto».
	Texto string
	// ProjectID e MissionID restringem por eixo. Vazio significa «todos os que o sujeito pode ver».
	ProjectID string
	MissionID string
	// MediaType exige prefixo (`image/` casa `image/png`). Vazio significa «qualquer tipo».
	MediaType string
	// IncluirVersoesAntigas: por omissao (false), a busca devolve **a versao corrente** de cada artefacto - *porque
	// listar todas as versoes de tudo e' ruido, e o historico pede-se.*
	IncluirVersoesAntigas bool
}

// chaveDeArtefacto e' a identidade de um artefacto AO LONGO DAS VERSOES: *o ProjectID + o Path.*
//
// *Nao uso o `ID`, porque o `ID` e' por versao (`art_<nano>`), e duas versoes do mesmo ficheiro tem IDs diferentes -
// usa-lo como chave tornaria «a versao corrente» impossivel de calcular.*
func chaveDeArtefacto(m ArtifactManifest) string {
	return strings.TrimSpace(m.ProjectID) + "\x00" + strings.TrimSpace(m.Path)
}

// BuscarArtefactos devolve os artefactos visiveis para `q.Sujeito` no tenant `q.TenantID`, filtrados.
//
// *Devolve copia ordenada. A ordem e' determinista - nome, e depois versao decrescente - para que dois `BuscarArtefactos`
// iguais produzam a mesma lista em qualquer execucao.*
func BuscarArtefactos(indice []ArtifactManifest, q ArtifactQuery) ([]ArtifactManifest, error) {
	if strings.TrimSpace(q.TenantID) == "" {
		return nil, ErrBuscaSemTenant
	}
	if strings.TrimSpace(q.Sujeito) == "" {
		return nil, ErrBuscaSemSujeito
	}

	texto := strings.ToLower(strings.TrimSpace(q.Texto))
	tipo := strings.ToLower(strings.TrimSpace(q.MediaType))

	// ***PASSO 1: VISIBILIDADE. Antes de tudo.***
	visiveis := make([]ArtifactManifest, 0, len(indice))
	for _, m := range indice {
		if !ArtifactVisivelPara(m, q.TenantID, q.Sujeito) {
			continue
		}
		// *E o tenant do filtro tem de ser o do artefacto - `ArtifactVisivelPara` ja' o garante, mas reafirmo-o aqui
		// porque e' desta linha que depende a fronteira, e uma fronteira que se apoia num so' sitio e' uma fronteira
		// que cai quando esse sitio mudar.*
		if !strings.EqualFold(strings.TrimSpace(m.TenantID), strings.TrimSpace(q.TenantID)) {
			continue
		}
		visiveis = append(visiveis, m)
	}

	// ***PASSO 2: OS FILTROS ADITIVOS, ja' sobre o conjunto visivel.***
	filtrados := make([]ArtifactManifest, 0, len(visiveis))
	for _, m := range visiveis {
		if q.ProjectID != "" && !strings.EqualFold(strings.TrimSpace(m.ProjectID), strings.TrimSpace(q.ProjectID)) {
			continue
		}
		if q.MissionID != "" && !strings.EqualFold(strings.TrimSpace(m.MissionID), strings.TrimSpace(q.MissionID)) {
			continue
		}
		if tipo != "" && !strings.HasPrefix(strings.ToLower(m.MediaType), tipo) {
			continue
		}
		if texto != "" {
			emNome := strings.Contains(strings.ToLower(m.Name), texto)
			emCaminho := strings.Contains(strings.ToLower(m.Path), texto)
			if !emNome && !emCaminho {
				continue
			}
		}
		filtrados = append(filtrados, m)
	}

	// ***PASSO 3: A VERSAO CORRENTE, quando o historico nao foi pedido.***
	final := filtrados
	if !q.IncluirVersoesAntigas {
		corrente := map[string]ArtifactManifest{}
		for _, m := range filtrados {
			k := chaveDeArtefacto(m)
			atual, existe := corrente[k]
			if !existe || m.Version > atual.Version {
				corrente[k] = m
			}
		}
		final = make([]ArtifactManifest, 0, len(corrente))
		for _, m := range corrente {
			final = append(final, m)
		}
	}

	// ***PASSO 4: ORDEM DETERMINISTA.*** *Uma busca que devolve a mesma coisa em ordens diferentes e' uma busca que
	// obriga quem le' a ordenar - e quem ordena por engano esconde duplicados.*
	sort.SliceStable(final, func(i, j int) bool {
		if final[i].Name == final[j].Name {
			if final[i].Path == final[j].Path {
				return final[i].Version > final[j].Version
			}
			return final[i].Path < final[j].Path
		}
		return final[i].Name < final[j].Name
	})
	return final, nil
}

// ProximaVersaoDeArtefacto calcula a versao seguinte para uma identidade (ProjectID + Path) **dentro de um tenant**.
//
// *E' o `expectedVersion` do `builder`, aplicado ao artefacto: **quem grava tem de dizer sobre que versao esta' a
// gravar**, e a funcao recusa um `esperado` que ja' nao e' o corrente - em vez de sobrescrever em silencio.*
func ProximaVersaoDeArtefacto(indice []ArtifactManifest, tenantID, projectID, path string, esperado int) (int, error) {
	if strings.TrimSpace(tenantID) == "" {
		return 0, ErrBuscaSemTenant
	}
	corrente := 0
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
		if m.Version > corrente {
			corrente = m.Version
		}
	}
	// *`esperado == 0` significa «o artefacto ainda nao existe»: e' o unico caso em que a criacao e' legitima.*
	if esperado != corrente {
		return 0, errors.New("artifact version conflict: " +
			strings.TrimSpace(path) + " is at version " + strconv.Itoa(corrente) + ", not " + strconv.Itoa(esperado))
	}
	return corrente + 1, nil
}
