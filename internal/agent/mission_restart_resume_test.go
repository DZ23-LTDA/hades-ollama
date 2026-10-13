package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// T1 — RE-EXECUÇÃO APÓS REINÍCIO: O QUE O MOTOR **RECUSA** REFAZER (E-369).
//
// **MEDIDO antes**: o V2.2 pede no T1 *"estado recuperado após reinício"* e, além disso, que o trabalho **continue de
// onde parou**. Em E-368 eu provei a **recuperação do estado** e declarei que **continuar** era outra propriedade.
//
// A LEITURA DO MOTOR MOSTROU QUE A PROPRIEDADE JÁ EXISTE — e é de **segurança**, não de conveniência:
//
//	if step := interruptedNonReadStep(mission, r.tools); step != nil { … }
//
// Um passo que ficou `StepRunning` quando o processo morreu e cujo risco **não é leitura** é recusado: ***reexecutar
// um passo de escrita depois de um crash pode DUPLICAR o efeito*** (um pagamento, um envio, um arquivo sobrescrito).
// Já um passo de **leitura** interrompido não tem esse perigo.
//
// ***A distinção entre "pode continuar" e "não pode repetir" é a parte do reinício que evita o estrago.***

func missaoComPassoInterrompido(t *testing.T, raiz string, risco RiskClass) (*Runtime, Mission) {
	t.Helper()
	runtime, contexto := runtimeSobre(t, raiz)
	projeto, err := contexto.CreateProject("projeto de retomada", filepath.Join(raiz, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "missão interrompida no meio de um passo", ProjectID: projeto.ID, OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Monta o cenário REAL de interrupção: o passo está `StepRunning` porque o processo morreu executando-o.
	missao.Plan = []Step{{
		ID:    "passo-interrompido",
		Kind:  "file.write",
		Title: "passo que ficou em execução",
		State: StepRunning,
		Risk:  risco,
		Input: map[string]any{"path": "saida.txt", "content": "efeito que NÃO pode ser duplicado"},
	}}
	missao.State = MissionRunning
	// MEDIDO: PutMission é otimista — sem incrementar a versão ele recusa com 'mission version conflict'.
	missao.Version++
	if err := runtime.store.PutMission(missao); err != nil {
		t.Fatalf("persistir a missão interrompida: %v", err)
	}
	return runtime, missao
}

func TestReinicioNaoReexecutaPassoDeEscritaInterrompido(t *testing.T) {
	raiz := t.TempDir()
	_, missao := missaoComPassoInterrompido(t, raiz, RiskDestructive)

	// PROCESSO 2: sobe sobre o mesmo diretório e tenta continuar a missão.
	depois, _ := runtimeSobre(t, raiz)
	err := depois.Run(context.Background(), missao.ID)

	recuperada, errGet := depois.store.GetMission(missao.ID)
	if errGet != nil {
		t.Fatalf("a missão precisa continuar existindo: %v", errGet)
	}
	// ARMADILHA: um passo de ESCRITA interrompido não pode voltar a rodar sozinho.
	if err == nil {
		t.Fatalf("ARMADILHA: o motor reexecutou um passo de ESCRITA interrompido sem recusa (estado final %q)", recuperada.State)
	}
	// ARMADILHA MEDIDA — E ELA PEGOU UM DEFEITO NO MEU PRÓPRIO TESTE: exigir apenas "err != nil" fazia o teste
	// passar por QUALQUER erro, inclusive "tool não registrada", que nada tem a ver com a recusa de segurança.
	// A contraprova (desligar `interruptedNonReadStep`) deixava o teste VERDE — ou seja, ele não discriminava nada.
	// *Teste que passa pelo motivo errado é pior que teste ausente: ele dá cobertura sem dar garantia.*
	if !strings.Contains(err.Error(), "passo-interrompido") && !strings.Contains(strings.ToLower(err.Error()), "interromp") {
		t.Fatalf("ARMADILHA: a recusa precisa nomear o passo interrompido — 'err != nil' sozinho não prova nada: %v", err)
	}
	// E o passo NÃO pode ficar marcado como executado com sucesso.
	for _, passo := range recuperada.Plan {
		if passo.ID == "passo-interrompido" && passo.State == StepSucceeded {
			t.Fatalf("ARMADILHA: passo de escrita interrompido foi marcado como CONCLUÍDO: %+v", passo)
		}
	}
}

func TestReinicioNaoRecusaPassoDeLeituraInterrompido(t *testing.T) {
	// O CONTRAPONTO: leitura interrompida não tem efeito colateral, então a recusa de segurança não se aplica.
	// A asserção é PRECISA: não quero saber se a execução terminou (isso depende de ferramentas); quero saber que a
	// recusa **não** é a de "passo não-leitura interrompido".
	raiz := t.TempDir()
	runtime, missao := missaoComPassoInterrompido(t, raiz, RiskRead)
	interrompidos := interruptedNonReadStep(missao, runtime.tools)
	if interrompidos != nil {
		t.Fatalf("ARMADILHA: passo de LEITURA interrompido foi classificado como não-leitura: %+v", interrompidos)
	}

	depois, _ := runtimeSobre(t, raiz)
	err := depois.Run(context.Background(), missao.ID)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "interromp") &&
		!strings.Contains(strings.ToLower(err.Error()), "leitura") {
		t.Fatalf("ARMADILHA: a recusa de passo interrompido foi aplicada a um passo de LEITURA: %v", err)
	}
	t.Logf("passo de leitura interrompido: o motor seguiu (erro de execução é admissível, recusa de segurança não): %v", err)
}
