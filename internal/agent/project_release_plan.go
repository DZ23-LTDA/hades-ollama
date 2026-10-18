package agent

// FASE 28.6 do adendo V3: O ELO ENTRE O DETECTOR E O RESOLVER.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// A E-1245 criou o `ProjectDetector` e provou-o com 15 testes. Mas nada o chamava: era o *quinto*
// caso, neste trabalho, de codigo provado por teste e nao usado pelo produto.
//
//	*Um detector que ninguem consulta e' um teste com um nome melhor.*
//
// O QUE FALTAVA, EM CONCRETO
//
//	ProjectDetector.Detectar()          "o que este projection PEDE"       (existia)
//	ResolverDeToolchain.Plano(ctx,alvo) "o que falta para este alvo"     (existia)
//
// O que nao existia era **a chamada entre os dois**. E sem ela o `CapabilityPlan` continuava a
// nascer de um alvo escrito a mao -- que era exactamente o problema que a E-1245 identificou.
//
// O QUE ESTE FICHEIRO ACRESCENTA, E O QUE RECUSA FAZER
//
// Acrescenta `PlanoParaProjeto`: detecta, filtra os alvos que o catalogo conhece, e devolve **um
// plano por alvo** mais o plano do projection.
//
// Recusa-se a:
//   - inventar alvos que o detector nao declarou;
//   - esconder que um alvo esta' bloqueado;
//   - devolver "tudo pronto" quando nao ha' alvos (isso e' `SemAlvos`, nao sucesso).
//
//	*Um plano vazio apresentado como sucesso e' a mesma mentira que um card clicavel que termina em
//	tela vazia.*

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PlanoPorAlvo e' o plano de UM alvo, com a proveniencia de onde o alvo veio.
type PlanoPorAlvo struct {
	// Alvo e' o nome do alvo no catalogo (windows, android, ...).
	Alvo string `json:"alvo"`
	// Plano e' o `CapabilityPlan` do resolver para este alvo.
	Plano CapabilityPlan `json:"plano"`
	// ErroDePlano diz porque o plano nao pode ser construido, quando aplicavel.
	//
	//	*Um alvo que nao consegue nem produzir um plano e' informacao, nao um item para omitir.*
	ErroDePlano string `json:"erro_de_plano,omitempty"`
}

// PlanoDoProjeto e' o resultado de `PlanoParaProjeto`.
type PlanoDoProjeto struct {
	// Projeto e' o que foi detectado. Sem isto o plano nao e' auditavel.
	Projeto ProjetoDetectado `json:"projeto"`
	// Alvos sao os planos por alvo, em ordem do nome do alvo.
	Alvos []PlanoPorAlvo `json:"alvos"`
	// AlvosProntos sao os alvos cujo plano esta' completo (sem blockers).
	AlvosProntos []string `json:"alvos_prontos"`
	// AlvosBloqueados sao os alvos com pelo menos um blocker.
	AlvosBloqueados []string `json:"alvos_bloqueados"`
	// SemAlvos diz que o projection nao declarou nenhum alvo que o catalogo conheca. Nao e' sucesso.
	SemAlvos bool `json:"sem_alvos"`
	// TrabalhoIndependente e' a uniao do trabalho que pode avancar sem desbloquear nada.
	TrabalhoIndependente []string `json:"trabalho_independente,omitempty"`
	// Avisos vem do detector e dos planos, sem duplicados.
	Avisos []string `json:"avisos,omitempty"`
	// Explicacao diz em uma frase o que o resultado significa.
	Explicacao string `json:"explicacao"`
}

// PlanoParaProjeto detecta o projection e constroi um plano por alvo.
//
// Este e' o metodo que faltava: sem argumento de alvo, porque o alvo vem do PROJECTION.
//
//	*Um preflight cujo alvo e' um parametro nao e' um preflight do projection: e' um preflight da
//	vontade de quem chamou.*
func (r *ResolverDeToolchain) PlanoParaProjeto(ctx context.Context, raiz string) (PlanoDoProjeto, error) {
	detector := ProjectDetector{Raiz: raiz, MaxProfundidade: r.ProfundidadeDeDeteccao}
	projeto, err := detector.Detectar()
	if err != nil {
		return PlanoDoProjeto{}, err
	}

	resultado := PlanoDoProjeto{
		Projeto:              projeto,
		Alvos:                []PlanoPorAlvo{},
		AlvosProntos:         []string{},
		AlvosBloqueados:      []string{},
		TrabalhoIndependente: []string{},
	}

	// Ordem deterministica: o detector ja' devolve os alvos ordenados, mas nao confiamos nisso
	// sem o dizer aqui -- a garantia tem de estar em quem consome.
	alvos := append([]string(nil), projeto.Alvos...)
	sort.Strings(alvos)

	visto := map[string]bool{}
	for _, alvo := range alvos {
		plano, errPlano := r.Plano(ctx, alvo)
		entrada := PlanoPorAlvo{Alvo: alvo, Plano: plano}
		if errPlano != nil {
			entrada.ErroDePlano = errPlano.Error()
			resultado.AlvosBloqueados = append(resultado.AlvosBloqueados, alvo)
		} else if plano.Pronto() {
			resultado.AlvosProntos = append(resultado.AlvosProntos, alvo)
		} else {
			resultado.AlvosBloqueados = append(resultado.AlvosBloqueados, alvo)
		}
		resultado.Alvos = append(resultado.Alvos, entrada)

		// Uniao do trabalho independente, sem duplicados: o mesmo `go install` nao precisa de
		// aparecer duas vezes por haver dois alvos.
		for _, t := range plano.TrabalhoIndependente {
			if visto[t] {
				continue
			}
			visto[t] = true
			resultado.TrabalhoIndependente = append(resultado.TrabalhoIndependente, t)
		}
	}

	// Avisos do detector primeiro: o que a deteccao nao conseguiu ver tem prioridade sobre a
	// estrategia.
	resultado.Avisos = append(resultado.Avisos, projeto.Avisos...)
	sort.Strings(resultado.AlvosProntos)
	sort.Strings(resultado.AlvosBloqueados)
	sort.Strings(resultado.TrabalhoIndependente)

	if len(projeto.Alvos) == 0 {
		resultado.SemAlvos = true
		resultado.Explicacao = "o projection nao declara nenhum alvo que o catalogo do HADES conheca; " +
			"nao ha' plano de build a construir"
		return resultado, nil
	}

	switch {
	case len(resultado.AlvosBloqueados) == 0:
		resultado.Explicacao = fmt.Sprintf(
			"todos os %d alvo(s) do projection estao' prontos nesta maquina",
			len(resultado.AlvosProntos))
	case len(resultado.AlvosProntos) == 0:
		resultado.Explicacao = fmt.Sprintf(
			"nenhum dos %d alvo(s) do projection esta' pronto; %d item(ns) de trabalho independente "+
				"podem avancar", len(resultado.AlvosBloqueados), len(resultado.TrabalhoIndependente))
	default:
		resultado.Explicacao = fmt.Sprintf(
			"%d alvo(s) pronto(s) e %d bloqueado(s); %d item(ns) de trabalho independente",
			len(resultado.AlvosProntos), len(resultado.AlvosBloqueados),
			len(resultado.TrabalhoIndependente))
	}
	return resultado, nil
}

// ResumoDeBlockers devolve os blockers de todos os alvos, sem duplicados, em ordem.
//
// Existe para o Mission Control poder mostrar UMA lista em vez de obrigar o utilizador a abrir
// cada alvo.
//
//	*Um relatorio que obriga a somar a mao e' um relatorio para quem ja' sabia a resposta.*
func (p PlanoDoProjeto) ResumoDeBlockers() []string {
	var todos []string
	for _, a := range p.Alvos {
		todos = append(todos, a.Plano.Blockers...)
		if a.ErroDePlano != "" {
			todos = append(todos, a.Alvo+": "+a.ErroDePlano)
		}
	}
	return ordenarSemDuplicados(todos)
}

// MotivosLegiveis devolve os motivos de cada exigencia nao saudavel, em texto.
//
// O `CapabilityPlan` guarda nomes de ferramentas; quem le' precisa do motivo. Sem isto o painel
// mostra "go" e o utilizador nao sabe se falta instalar, se a versao e' velha ou se o PATH esta'
// errado.
func (p PlanoDoProjeto) MotivosLegiveis() []string {
	var saida []string
	for _, a := range p.Alvos {
		for _, sonda := range a.Plano.Probes {
			if sonda.Estado.ResolveReady() {
				continue
			}
			// O motivo vem do REQUISITO (o que se esperava), o detalhe vem da SONDA (o que se
			// observou). Os dois juntos e' que explicam: *um estado sem motivo obriga quem le' a
			// adivinhar o que era suposto acontecer.*
			motivo := strings.TrimSpace(sonda.Requisito.Motivo)
			if detalhe := strings.TrimSpace(sonda.Detalhe); detalhe != "" {
				if motivo != "" {
					motivo += " (" + detalhe + ")"
				} else {
					motivo = detalhe
				}
			}
			if motivo == "" {
				motivo = "estado " + string(sonda.Estado)
			}
			saida = append(saida, a.Alvo+": "+sonda.Requisito.Nome+" -- "+motivo)
		}
	}
	return ordenarSemDuplicados(saida)
}

// ordenarSemDuplicados ordena e remove repeticoes.
func ordenarSemDuplicados(entrada []string) []string {
	visto := map[string]bool{}
	saida := make([]string, 0, len(entrada))
	for _, s := range entrada {
		if visto[s] {
			continue
		}
		visto[s] = true
		saida = append(saida, s)
	}
	sort.Strings(saida)
	return saida
}
