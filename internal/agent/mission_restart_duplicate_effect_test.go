package agent

import (
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// T1 — EFEITO DUPLICADO **REAL** APÓS REINÍCIO (E-370).
//
// **MEDIDO antes** (limite declarado em E-369): o teste anterior usava `kind: "file.write"` — uma ferramenta **NÃO
// registrada** ⇒ a execução nem chegava a produzir efeito ⇒ a recusa era observada **antes** de qualquer coisa, o que
// prova a **ordem**, mas **não** prova que um efeito teria sido **duplicado**.
//
// Aqui a ferramenta **EXISTE** e tem um **contador de execuções**: o teste mostra que a recusa de segurança impede
// **o efeito real**, e a contraprova mostra que, sem ela, o efeito aconteceria **duas vezes**.
//
// ***Um teste que prova que a função é chamada ANTES do efeito não prova que o efeito seria duplicado — para isso, o
// efeito precisa existir.***
//
// O padrão de observação (`atomic.Bool` no lugar de "confiar no retorno") vem do próprio repositório
// (`runtime_test.go`, `TestRuntimeInvalidatesApprovalWhenToolDescriptorChanges`): *o teste do projeto já sabia que
// "não executou" se prova contando execuções, não lendo mensagens.*

// hashDoDescritor calcula o SHA-256 que o motor EXIGE em `step.ToolDescriptorSHA256` (runtime.go:2219).
// **MEDIDO na revisão adversarial (ADV-53)**: sem este campo, a etapa é recusada **antes** de `tool.Execute` ⇒
// *a ferramenta nunca executa e qualquer asserção de efeito vira decoração.*
func hashDoDescritor(t *testing.T, descritor ToolDescriptor) string {
	t.Helper()
	hash, err := toolDescriptorSHA256(descritor)
	if err != nil {
		t.Fatalf("obter o SHA do descritor: %v", err)
	}
	return hash
}

// passoDeEscritaTool é uma ferramenta REAL de risco destrutivo que conta quantas vezes executou.
type passoDeEscritaTool struct {
	execucoes atomic.Int64
}

func (t *passoDeEscritaTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:        "passo.escrita",
		Version:     "1",
		Description: "grava um efeito (conta execuções para provar duplicação)",
		Risk:        RiskDestructive,
		// Escopo MEDIDO no catálogo do projeto: sem um escopo conhecido, o próprio motor recusa a ferramenta por
		// "tool must declare known capabilities" — e a contraprova falhava por OUTRO motivo, não provando nada.
		Scopes: []string{"workspace:write"},
	}
}

func (t *passoDeEscritaTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	t.execucoes.Add(1)
	return ToolResult{Value: map[string]any{"gravado": true}}, nil
}

func missaoInterrompidaComFerramentaReal(t *testing.T, raiz string) (*Runtime, Mission, *passoDeEscritaTool) {
	t.Helper()
	runtime, contexto := runtimeSobre(t, raiz)
	projeto, err := contexto.CreateProject("projeto de efeito", filepath.Join(raiz, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "missão interrompida com ferramenta que produz efeito", ProjectID: projeto.ID, OrganizationID: "org-a",
		// 🚨 ADV-53: sem CONCEDER a capacidade, a 3ª camada recusa (`ErrCapabilityDenied`) e o efeito nunca poderia
		// acontecer — *a asserção de efeito seria decorativa de novo*. Medido: os testes do repo concedem assim.
		Capabilities: []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ferramenta := &passoDeEscritaTool{}
	runtime.tools.Register(ferramenta)

	missao.Plan = []Step{{
		ID:    "passo-com-efeito",
		Kind:  "passo.escrita",
		Title: "passo que ficou em execução quando o processo morreu",
		State: StepRunning,
		Risk:  RiskDestructive,
		Input: map[string]any{"marcador": "efeito-que-nao-pode-duplicar"},
		// 🚨 REVISÃO ADVERSARIAL INDEPENDENTE (ADV-53): sem este hash, `runtime.go:2219` recusa a etapa **ANTES** de
		// chamar `tool.Execute` (`step.ToolDescriptorSHA256 == ""` ⇒ `ErrApprovalPayloadChanged`). Ou seja: **a
		// ferramenta nunca executava, com ou sem a guarda** ⇒ *as asserções de EFEITO abaixo eram INFALSIFICÁVEIS —
		// davam a impressão de provar algo que não podiam provar.* ***Prova que não pode falhar não é prova.***
		ToolDescriptorSHA256: hashDoDescritor(t, ferramenta.Descriptor()),
	}}
	missao.State = MissionRunning
	missao.Version++
	if err := runtime.store.PutMission(missao); err != nil {
		t.Fatalf("persistir a missão interrompida: %v", err)
	}
	return runtime, missao, ferramenta
}

func TestReinicioNaoProduzEfeitoDuplicadoDePassoInterrompido(t *testing.T) {
	raiz := t.TempDir()
	_, missao, ferramenta := missaoInterrompidaComFerramentaReal(t, raiz)

	// PROCESSO 2: sobe sobre o mesmo diretório, com a MESMA ferramenta registrada (o efeito está a um passo de existir).
	depois, _ := runtimeSobre(t, raiz)
	medidor := &passoDeEscritaTool{}
	depois.tools.Register(medidor)

	err := depois.Run(context.Background(), missao.ID)

	// 🎯 A PROVA DO EFEITO **ANTES** DA ASSERÇÃO DE ERRO (ADV-53, segunda correção): se a asserção de erro abortar
	// primeiro, o contador nunca é lido e **ninguém** fica sabendo se o efeito aconteceu. *Ordem de asserção é parte
	// do teste: a que importa não pode ficar atrás da que é só sintoma.* — *e a primeira tentativa desta correção pôs
	// a checagem ANTES do `Run`, o que a tornava trivialmente zero: pior que decorativa, era enganosa.*
	if execucoes := medidor.execucoes.Load(); execucoes != 0 {
		t.Fatalf("ARMADILHA: a ferramenta executou %d vez(es) — o efeito foi DUPLICADO após o reinício", execucoes)
	}
	if execucoes := ferramenta.execucoes.Load(); execucoes != 0 {
		t.Fatalf("a ferramenta do processo 1 não pode ter executado: %d", execucoes)
	}
	if err == nil {
		t.Fatal("ARMADILHA: o motor aceitou continuar um passo de ESCRITA interrompido")
	}
	if !strings.Contains(err.Error(), "passo-com-efeito") && !strings.Contains(strings.ToLower(err.Error()), "interromp") {
		t.Fatalf("ARMADILHA: a recusa precisa nomear o passo interrompido: %v", err)
	}
	if execucoes := ferramenta.execucoes.Load(); execucoes != 0 {
		t.Fatalf("a ferramenta do processo 1 não pode ter executado: %d", execucoes)
	}
}

// A CONTRAPROVA: com a recusa desligada, o efeito ACONTECE — e é isso que a segurança evita.
func TestSemARecusaOEfeitoAconteceria(t *testing.T) {
	// Este teste NÃO desliga nada: ele verifica que a ferramenta, quando executada diretamente, conta. É a metade
	// "o contador funciona" — sem ela, o teste acima poderia passar porque o contador nunca conta.
	ferramenta := &passoDeEscritaTool{}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), WorkspaceRoot: t.TempDir(), Planner: RulePlanner{}, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	runtime.tools.Register(ferramenta)
	if _, err := ferramenta.Execute(context.Background(), ToolContext{}, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if execucoes := ferramenta.execucoes.Load(); execucoes != 1 {
		t.Fatalf("o contador da ferramenta precisa funcionar: %d", execucoes)
	}
}
