package agent

import (
	"testing"
)

// E-472 — **O PAYLOAD DA APROVAÇÃO, CAMPO A CAMPO: 8 CAMPOS E UMA PRÉ-CONDIÇÃO, NÃO "5 CAMADAS".**
//
// **A CORRECÇÃO DO PRÓPRIO REGISTRO**: eu vinha escrevendo *"as 5 camadas do `ErrApprovalPayloadChanged`"* ✗ — e
// **medi**: `approvalPayloadSHA256` (`workspace_write.go:27-55`) hasheia **8 campos** (`ID`, `Kind`, `Title`, `Input`,
// `Risk`, `RequiresApproval`, `ToolDescriptorSHA256`, `ToolConfigSHA256`) e tem **uma pré-condição** (*a redacção vem
// **primeiro***). ⇒ ***o "5" era uma **memória**, não uma medição*** ⇒ *é **exactamente** o que o **E-453** condenou:
// «contagem sem data é a forma mais convincente de afirmação sem lastro» — **e eu repeti o erro em nove rodadas de
// relatórios** ✓.
//
// **E A PEÇA QUE NINGUÉM MENCIONAVA**: a **redacção vem ANTES do hash** ✓ — *se o `Input` contém dado sensível, a
// aprovação **não pode ser criada de todo*** (`errors.New("approval payload contains sensitive data that cannot be
// persisted safely")`) ⇒ ***é uma camada que nenhum dos meus relatórios citava, e é a primeira.***
//
// ⇒ este teste exercita **cada um dos 8 campos**: *mudar qualquer um tem de mudar o hash* — *se um campo não entrar no
// hash, a aprovação fica ligada a um payload que já não é o aprovado, e a reconferência não vê.*
func TestPayloadDaAprovacaoPorCampo(t *testing.T) {
	base := Step{
		ID:                   "passo-1",
		Kind:                 "workspace.write",
		Title:                "escrever ficheiro",
		Input:                map[string]any{"path": "nota.txt", "content": "conteudo limpo"},
		Risk:                 RiskWrite,
		RequiresApproval:     true,
		ToolDescriptorSHA256: "desc-base",
		ToolConfigSHA256:     "cfg-base",
	}
	referencia, err := approvalPayloadSHA256(base)
	if err != nil {
		t.Fatalf("hash de referência: %v", err)
	}

	// Cada mutação: *um campo de cada vez, e o hash TEM de mudar.*
	mutacoes := []struct {
		nome  string
		aplic func(passo *Step)
	}{
		{"ID", func(p *Step) { p.ID = "passo-2" }},
		{"Kind", func(p *Step) { p.Kind = "outro.kind" }},
		{"Title", func(p *Step) { p.Title = "outro titulo" }},
		{"Input", func(p *Step) { p.Input = map[string]any{"path": "outro.txt", "content": "conteudo limpo"} }},
		{"Risk", func(p *Step) { p.Risk = RiskDestructive }},
		{"RequiresApproval", func(p *Step) { p.RequiresApproval = false }},
		{"ToolDescriptorSHA256", func(p *Step) { p.ToolDescriptorSHA256 = "desc-outra" }},
		{"ToolConfigSHA256", func(p *Step) { p.ToolConfigSHA256 = "cfg-outra" }},
	}
	for _, mutacao := range mutacoes {
		passo := base
		// O mapa é por referência: *copiar o struct não copia o `Input`.*
		passo.Input = map[string]any{"path": "nota.txt", "content": "conteudo limpo"}
		mutacao.aplic(&passo)
		alterado, err := approvalPayloadSHA256(passo)
		if err != nil {
			t.Fatalf("campo %s: %v", mutacao.nome, err)
		}
		if alterado == referencia {
			t.Fatalf("ARMADILHA: mudar `%s` **NAO** mudou o hash da aprovacao — *o campo nao entra no payload, entao a "+
				"aprovacao fica ligada a um payload que ja' nao e' o aprovado, e a reconferencia nao ve'*", mutacao.nome)
		}
	}

	// E a PRÉ-CONDIÇÃO, que é a primeira camada: *input com dado sensível NÃO gera hash.*
	sensivel := base
	sensivel.Input = map[string]any{"path": "nota.txt", "content": "chave sk-abcdefghijklmnopqrstuvwxyz012345"}
	if _, err := approvalPayloadSHA256(sensivel); err == nil {
		t.Fatal("ARMADILHA: um input com dado sensivel gerou hash — *a redacao tem de vir ANTES, e a aprovacao nao pode " +
			"ser criada de todo*")
	}

	// E o determinismo: *o mesmo passo tem de dar o mesmo hash* — *sem isto, a reconferência acusaria sempre.*
	repetido, err := approvalPayloadSHA256(base)
	if err != nil {
		t.Fatalf("repeticao: %v", err)
	}
	if repetido != referencia {
		t.Fatal("o MESMO passo deu hashes diferentes — *a reconferencia acusaria sempre e nada executaria*")
	}
}
