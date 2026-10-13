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
	if err == nil {
		t.Fatal("ARMADILHA: o motor aceitou continuar um passo de ESCRITA interrompido")
	}
	if !strings.Contains(err.Error(), "passo-com-arquivo") && !strings.Contains(strings.ToLower(err.Error()), "interromp") {
		t.Fatalf("ARMADILHA: a recusa precisa nomear o passo interrompido: %v", err)
	}
	// A PROVA NO MUNDO: o arquivo NÃO pode existir.
	if _, erro := os.Stat(caminhoDoEfeito); erro == nil {
		t.Fatalf("ARMADILHA: o arquivo %s EXISTE — o efeito em disco foi produzido após o reinício", caminhoDoEfeito)
	} else if !os.IsNotExist(erro) {
		t.Fatalf("erro inesperado ao verificar o efeito: %v", erro)
	}
}

// A METADE QUE PROVA QUE O EFEITO **SABE** ACONTECER: sem ela, o teste acima poderia passar porque a ferramenta
// nunca grava nada.
func TestFerramentaDeArquivoGravaQuandoExecutada(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "controle.txt")
	ferramenta := &arquivoDeEscritaTool{caminho: caminho}
	if _, err := ferramenta.Execute(context.Background(), ToolContext{}, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("o efeito em disco precisa acontecer quando a ferramenta executa: %v", err)
	}
	if !strings.Contains(string(dados), "NÃO pode ser duplicado") {
		t.Fatalf("conteúdo inesperado no arquivo de controle: %q", string(dados))
	}
}
