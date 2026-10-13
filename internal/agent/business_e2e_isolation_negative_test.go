package agent

import (
	"strings"
	"testing"
	"time"
)

// Estágio 32 — A JORNADA E2E REPROVA QUANDO HÁ VAZAMENTO ENTRE TENANTS (E-287).
//
// O código JÁ trata a direção negativa (`if !IsolationProven { Verdict = FALHOU }`), mas só havia teste do caminho
// felizista. Um guarda sem teste negativo é uma promessa: se alguém inverter a condição, a suíte continua verde.
//
// COMO O VAZAMENTO É PROVOCADO SEM INVENTAR NADA: a jornada recebe a organização a certificar e uma SEGUNDA
// organização usada para tentar ler. Passando a MESMA organização nos dois papéis, toda leitura "cruzada" passa a
// ser bem-sucedida — que é exatamente a condição de vazamento. Não é simulação: é a leitura de fato acontecendo.
//
// Propriedades travadas:
//  1. SEM vazamento (organizações distintas): IsolationProven = true e veredito aprovado;
//  2. COM vazamento (mesma organização nos dois papéis): IsolationProven = FALSE e veredito FALHOU;
//  3. o relatório NOMEIA o recurso que vazou (não basta dizer "falhou").
func TestJornadaE2EReprovaQuandoHaVazamentoEntreTenants(t *testing.T) {
	now := time.Now().UTC()

	// 1. Organizações distintas: sem leitura cruzada, jornada aprovada.
	runtimeLimpo, _ := businessRuntime(t)
	limpa := DefaultBusinessJourneyInput(LocalOrganizationID, "org-vizinhanca")
	relatorioLimpo, err := runtimeLimpo.CertifyBusinessJourney(limpa, now)
	if err != nil {
		t.Fatalf("jornada limpa: %v", err)
	}
	if !relatorioLimpo.IsolationProven {
		t.Fatalf("organizações distintas não podem produzir vazamento: %+v", relatorioLimpo.Isolation)
	}
	if relatorioLimpo.Verdict != BusinessVerdictOK {
		t.Fatalf("jornada sem vazamento precisa ser aprovada, veio %q", relatorioLimpo.Verdict)
	}

	// 2 e 3. Mesma organização nos dois papéis: as leituras cruzadas ACONTECEM. A jornada vai além de reprovar no
	// relatório: ela RECUSA CERTIFICAR, devolvendo erro que nomeia o problema. Era mais forte do que o teste supunha
	// — e é isso que se descobre escrevendo a direção negativa em vez de presumi-la.
	runtimeVazando, _ := businessRuntime(t)
	vazando := DefaultBusinessJourneyInput(LocalOrganizationID, LocalOrganizationID)
	relatorioVazando, err := runtimeVazando.CertifyBusinessJourney(vazando, now)
	if err == nil {
		t.Fatalf("leitura cruzada entre tenants PRECISA impedir a certificação (verdict=%q, isolation=%+v)",
			relatorioVazando.Verdict, relatorioVazando.Isolation)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "isola") {
		t.Fatalf("o erro precisa NOMEAR o isolamento: %v", err)
	}
	// Se algo foi devolvido junto do erro, NUNCA pode vir aprovado.
	if relatorioVazando.Verdict == BusinessVerdictOK {
		t.Fatal("vazamento entre tenants NÃO pode resultar em veredito aprovado, nem junto de erro")
	}
	// A tentativa de leitura cruzada precisa ficar registrada e NOMEADA: "falhou" sem dizer o quê não orienta correção.
	if len(relatorioVazando.Isolation) > 0 {
		nomeou, visivel := false, false
		for _, checagem := range relatorioVazando.Isolation {
			if checagem.Resource != "" {
				nomeou = true
			}
			if checagem.VisibleToOtherTenant {
				visivel = true
			}
		}
		if !nomeou || !visivel {
			t.Fatalf("leitura cruzada bem-sucedida precisa aparecer nomeada e visível: %+v", relatorioVazando.Isolation)
		}
	}
}
