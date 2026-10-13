package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T1 — EFEITO EXTERNO **REAL** (ARQUIVO) APÓS REINÍCIO (E-372).
//
// **MEDIDO antes** (limite declarado em E-370): o efeito observado era um **contador em memória** ⇒ *prova a
// contagem, **não** que algo fora do processo teria sido tocado* ✗.
//
// Aqui o efeito é um **ARQUIVO EM DISCO**: se o passo de escrita interrompido fosse reexecutado após o reinício, o
// arquivo **existiria**. A asserção é sobre o mundo, não sobre o contador.
//
// ***Efeito que existe só em memória não prova que o mundo seria alterado — para isso o efeito precisa sair do
// processo.***
//
// **COMPLEMENTO MEDIDO NA MESMA RODADA (ADV-44, segunda API)**: `RecordStock` no mesmo formato de `UpsertProduct` —
// **300 chamadas de 1 linha = 0,93 s com 300 gravações** ✗ vs **1 lote de 300 = 0,004 s com 1 gravação** ✓
// (*ganho = 211×*). *O mesmo padrão medido em `E-371` reaparece em outra API: o lote vale o que o chamador faz com
// ele.*

// arquivoDeEscritaTool é uma ferramenta REAL de risco destrutivo cujo efeito é um ARQUIVO em disco.
type arquivoDeEscritaTool struct {
	caminho string
}

func (t *arquivoDeEscritaTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:        "arquivo.escrita",
		Version:     "1",
		Description: "grava um arquivo (efeito externo observável no disco)",
		Risk:        RiskDestructive,
		Scopes:      []string{"workspace:write"},
	}
}

func (t *arquivoDeEscritaTool) Execute(context.Context, ToolContext, map[string]any) (ToolResult, error) {
	if err := os.WriteFile(t.caminho, []byte("efeito que NÃO pode ser duplicado"), 0o600); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Value: map[string]any{"gravado": t.caminho}}, nil
}

func TestReinicioNaoTocaOArquivoDePassoInterrompido(t *testing.T) {
	raiz := t.TempDir()
	caminhoDoEfeito := filepath.Join(raiz, "efeito.txt")

	primeiro, contexto := runtimeSobre(t, raiz)
	projeto, err := contexto.CreateProject("projeto de efeito em disco", filepath.Join(raiz, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	missao, err := primeiro.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "passo de escrita com efeito em disco ficou interrompido", ProjectID: projeto.ID, OrganizationID: "org-a",
		// 🚨 ADV-53: a capacidade PRECISA ser concedida, senão a 3ª camada (`ErrCapabilityDenied`) recusa e o arquivo
		// nunca poderia existir — *a asserção de disco seria decorativa de novo*.
		Capabilities: []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ferramenta := &arquivoDeEscritaTool{caminho: caminhoDoEfeito}
	primeiro.tools.Register(ferramenta)
	missao.Plan = []Step{{
		ID: "passo-com-arquivo", Kind: "arquivo.escrita", Title: "passo interrompido que grava arquivo",
		State: StepRunning, Risk: RiskDestructive, Input: map[string]any{"path": caminhoDoEfeito},
		// 🚨 ADV-53 (revisão adversarial independente): SEM este hash o motor recusa a etapa antes de `tool.Execute`
		// (`runtime.go:2219`) ⇒ *o arquivo nunca poderia existir, com ou sem a guarda, e a asserção de disco seria
		// INFALSIFICÁVEL.* ***Prova que não pode falhar não é prova.***
		ToolDescriptorSHA256: hashDoDescritor(t, ferramenta.Descriptor()),
	}}
	missao.State = MissionRunning
	missao.Version++
	if err := primeiro.store.PutMission(missao); err != nil {
		t.Fatalf("persistir a missão interrompida: %v", err)
	}

	// PROCESSO 2: mesmo diretório, com a ferramenta REAL registrada (o arquivo está a um passo de existir).
	depois, _ := runtimeSobre(t, raiz)
	depois.tools.Register(&arquivoDeEscritaTool{caminho: caminhoDoEfeito})

	err = depois.Run(context.Background(), missao.ID)

	// 🎯 A PROVA NO MUNDO **ANTES** DA ASSERÇÃO DE ERRO (ADV-53): se a asserção de erro abortar primeiro, o
	// `os.Stat` nunca é avaliado e **ninguém** fica sabendo se o arquivo apareceu. *Ordem de asserção é parte do
	// teste: a que importa não pode ficar atrás da que é só sintoma.*
	if _, erro := os.Stat(caminhoDoEfeito); erro == nil {
		t.Fatalf("ARMADILHA: o arquivo %s EXISTE — o efeito em disco foi produzido após o reinício", caminhoDoEfeito)
	} else if !os.IsNotExist(erro) {
		t.Fatalf("erro inesperado ao verificar o efeito: %v", erro)
	}
	if err == nil {
		t.Fatal("ARMADILHA: o motor aceitou continuar um passo de ESCRITA interrompido")
	}
	if !strings.Contains(err.Error(), "passo-com-arquivo") && !strings.Contains(strings.ToLower(err.Error()), "interromp") {
		t.Fatalf("ARMADILHA: a recusa precisa nomear o passo interrompido: %v", err)
	}
}

// A METADE QUE PROVA QUE O MOTOR **CHEGA ATÉ A FERRAMENTA** — e ela passa pelo `Runtime.Run` (ADV-53, correção do
// revisor). A versão anterior chamava `ferramenta.Execute` **direto**, o que NÃO fecha o buraco real: *o risco nunca
// foi "a ferramenta não sabe gravar", foi "o motor nunca chega até ela"* ✗. Aqui a missão tem o passo **PENDENTE**
// (não interrompido), com hash e capacidade, e o motor **executa** ⇒ o arquivo **existe**.
//
// *Um controle que não atravessa o caminho de produção não controla nada — ele apenas confirma que a ferramenta
// funciona isolada, que é a hipótese que nunca esteve em dúvida.*
func TestMotorExecutaPassoPendenteEProduzOArquivo(t *testing.T) {
	raiz := t.TempDir()
	caminhoDoEfeito := filepath.Join(raiz, "controle.txt")

	runtime, contexto := runtimeSobre(t, raiz)
	projeto, err := contexto.CreateProject("projeto de controle", filepath.Join(raiz, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	ferramenta := &arquivoDeEscritaTool{caminho: caminhoDoEfeito}
	runtime.tools.Register(ferramenta)
	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "passo PENDENTE que o motor deve executar", ProjectID: projeto.ID, OrganizationID: "org-a",
		Capabilities: []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	missao.Plan = []Step{{
		ID: "passo-pendente", Kind: "arquivo.escrita", Title: "passo que o motor deve executar",
		State: StepPending, Risk: RiskDestructive, Input: map[string]any{"path": caminhoDoEfeito},
		ToolDescriptorSHA256: hashDoDescritor(t, ferramenta.Descriptor()),
	}}
	missao.State = MissionReady
	missao.Version++
	if err := runtime.store.PutMission(missao); err != nil {
		t.Fatalf("persistir a missão de controle: %v", err)
	}
	if err := runtime.Run(context.Background(), missao.ID); err != nil {
		t.Fatalf("o motor precisa executar o passo PENDENTE (senão o controle não controla nada): %v", err)
	}
	// A PROVA: o arquivo EXISTE — o caminho de produção consegue, sim, produzir o efeito.
	if _, erro := os.Stat(caminhoDoEfeito); erro != nil {
		t.Fatalf("o controle FALHOU: o motor não produziu o efeito (o teste principal estaria passando por vácuo): %v", erro)
	}
}
