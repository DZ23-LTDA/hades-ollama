package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// COMPARAR VERSÕES DE SKILL (E-648) — o §9 do Paperclip pede «comparar revisões».
//
// O PADRÃO JÁ EXISTIA EM ARTEFACTOS (`ArtefactoCompararVersoes`, com `ErrVersaoInexistente` e
// `ErrBuscaSemTenant`). Aqui segue-se o mesmo desenho para SKILLS, sem o copiar: o que interessa
// ver entre versões não é tudo, e o que fica de fora é a decisão importante.

// SkillMudanca é uma diferença num campo entre duas versões.
type SkillMudanca struct {
	Campo string `json:"field"`
	De    string `json:"from"`
	Para  string `json:"to"`
}

// SkillDiferencaVersoes é o resultado de comparar duas versões arquivadas.
type SkillDiferencaVersoes struct {
	SkillID  string         `json:"skill_id"`
	De       string         `json:"from"`
	Para     string         `json:"to"`
	Mudancas []SkillMudanca `json:"changes"`
	// Integridade fica DITA À PARTE: o hash e a assinatura não são «conteúdo de trabalho», são a
	// prova de que o conteúdo é o que diz ser. Misturá-los com a descrição faria a diferença mais
	// importante passar despercebida no meio das outras.
	ConteudoMudou   bool `json:"content_changed"`
	AssinaturaMudou bool `json:"signature_changed"`
}

// lerManifestoArquivado lê UMA versão arquivada, com a MESMA validação de travessia do restauro.
//
// Foi extraído do `RestoreSkillVersionForOrganization` de propósito: duas cópias de uma verificação
// de caminho divergem, e a que fica a menos não se nota (a lição do E-626, onde um filtro de
// inquilino duplicado era o risco). *A validação tem de estar num sítio só.*
func (s *ContextStore) lerManifestoArquivado(id, versao string) (SkillManifest, error) {
	if !idDeSkillSeguro(id) {
		return SkillManifest{}, errors.New("skill id is invalid")
	}
	if !versaoDeSkillSegura(versao) {
		return SkillManifest{}, errors.New("skill version is invalid")
	}
	if s.root == "" {
		return SkillManifest{}, errors.New("skill history requires a store root")
	}
	base := filepath.Join(s.root, "skills", id+".versions")
	caminho := filepath.Join(base, versao+".json")
	// DEFESA EM PROFUNDIDADE: validação e travessia podem divergir.
	if relativo, err := filepath.Rel(base, caminho); err != nil || strings.HasPrefix(relativo, "..") {
		return SkillManifest{}, errors.New("skill version path escapes the history folder")
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return SkillManifest{}, fmt.Errorf("ler versao arquivada: %w", err)
	}
	var manifesto SkillManifest
	decoder := json.NewDecoder(bytes.NewReader(dados))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifesto); err != nil {
		return SkillManifest{}, fmt.Errorf("versao arquivada invalida: %w", err)
	}
	if manifesto.ID != id || manifesto.Version != versao {
		return SkillManifest{}, errors.New("versao arquivada nao corresponde ao pedido")
	}
	return manifesto, nil
}

// CompararVersoesDeSkill compara duas versões ARQUIVADAS da mesma skill.
//
// O QUE COMPARA: `description`, `scopes`, `tools` e `category` — o que muda o que a skill FAZ.
// O QUE NÃO COMPARA COMO CONTEÚDO: `trusted` e `enabled`, que são estado derivado pelo servidor e
// não propriedades do manifesto; mostrá-los como «mudança» sugeriria que uma versão antiga é mais
// confiável do que é — ***restaurar não promove confiança (E-631), e a comparação não pode sugerir
// o contrário.***
//
// O escopo de organização é aplicado pelo mesmo caminho do restauro: a organização de quem pergunta
// entra na leitura, e uma skill que não é dela não é comparável.
func (s *ContextStore) CompararVersoesDeSkill(organizationID, id, de, para string) (SkillDiferencaVersoes, error) {
	s.mu.Lock()
	atual, existe := s.skills[id]
	s.mu.Unlock()
	if !existe {
		return SkillDiferencaVersoes{}, fmt.Errorf("skill %q: %w", id, ErrVersaoInexistente)
	}
	if organizationID != "" && !pluginOwnedByOrganization(atual.OrganizationID, organizationID) {
		// A mesma mensagem de «não existe» das outras leituras: distinguir seria uma pista.
		return SkillDiferencaVersoes{}, fmt.Errorf("skill %q: %w", id, ErrVersaoInexistente)
	}
	// A VERSÃO ACTIVA NÃO ESTÁ NO ARQUIVO — só as SUPERADAS ficam lá. Comparar a revisão actual com
	// uma antiga é o caso MAIS útil, e sem isto falharia com «não existe»: *um erro que mente sobre
	// a razão.* O manifesto vivo serve quando a versão pedida é a activa.
	manifestoDaVersao := func(versao string) (SkillManifest, error) {
		if versao == atual.Version {
			return atual, nil
		}
		return s.lerManifestoArquivado(id, versao)
	}
	manifestoDe, err := manifestoDaVersao(de)
	if err != nil {
		return SkillDiferencaVersoes{}, fmt.Errorf("versao %q: %w", de, ErrVersaoInexistente)
	}
	manifestoPara, err := manifestoDaVersao(para)
	if err != nil {
		return SkillDiferencaVersoes{}, fmt.Errorf("versao %q: %w", para, ErrVersaoInexistente)
	}

	diferenca := SkillDiferencaVersoes{
		SkillID:         id,
		De:              de,
		Para:            para,
		ConteudoMudou:   manifestoDe.ContentSHA256 != manifestoPara.ContentSHA256,
		AssinaturaMudou: manifestoDe.Signature != manifestoPara.Signature || manifestoDe.SigningKeyID != manifestoPara.SigningKeyID,
	}
	acrescentar := func(campo, antigo, novo string) {
		if antigo != novo {
			diferenca.Mudancas = append(diferenca.Mudancas, SkillMudanca{Campo: campo, De: antigo, Para: novo})
		}
	}
	acrescentar("description", manifestoDe.Description, manifestoPara.Description)
	acrescentar("category", manifestoDe.Category, manifestoPara.Category)
	acrescentar("scopes", strings.Join(ordenadas(manifestoDe.Scopes), ","), strings.Join(ordenadas(manifestoPara.Scopes), ","))
	acrescentar("tools", strings.Join(ordenadas(manifestoDe.Tools), ","), strings.Join(ordenadas(manifestoPara.Tools), ","))
	return diferenca, nil
}

// ordenadas devolve uma cópia ordenada: comparar listas por ORDEM acusaria mudanças que não mudam
// o que a skill faz (a mesma permissão escrita noutra ordem é a mesma permissão).
func ordenadas(valores []string) []string {
	copia := append([]string(nil), valores...)
	sort.Strings(copia)
	return copia
}
