package agent

import (
	"strings"
	"testing"
)

// Testes da integração do handoff no Context Bootstrap (§4): o contrato entra no documento
// canónico E no preâmbulo que vai ao modelo — as duas coisas, porque só a primeira deixaria
// a transferência de contexto a acontecer em JSON que ninguém lê.

func bootstrapBase() ContextBootstrapInput {
	return ContextBootstrapInput{
		Version:        "test",
		OrganizationID: "org-a",
		ProjectID:      "proj-1",
		Objective:      "entregar a compactação",
	}
}

func TestBootstrapSemHandoffNaoInventaSeccao(t *testing.T) {
	b, err := BuildContextBootstrap(bootstrapBase())
	if err != nil {
		t.Fatal(err)
	}
	if b.Handoff != "" {
		t.Fatalf("apareceu handoff sem ter sido dado: %q", b.Handoff)
	}
	// E o preâmbulo não ganha secção nenhuma: um cabeçalho vazio faria o modelo procurar
	// um handoff que não existe.
	if strings.Contains(b.SystemMessage(), MarcadorDeHandoff) {
		t.Fatal("o preâmbulo anuncia um handoff inexistente")
	}
}

func TestBootstrapComHandoffSaiNoDocumentoENoPreambulo(t *testing.T) {
	h, err := HandoffDaMissao(missaoDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	in := bootstrapBase()
	in.MissionID = "m-1"
	in.Handoff = &h

	b, err := BuildContextBootstrap(in)
	if err != nil {
		t.Fatal(err)
	}
	// (1) NO DOCUMENTO.
	if !strings.Contains(b.Handoff, MarcadorDeHandoff) {
		t.Fatalf("o documento não traz o bloco marcado: %q", b.Handoff)
	}
	// (2) NO PREÂMBULO — é isto que faz a transferência chegar ao modelo.
	preambulo := b.SystemMessage()
	if !strings.Contains(preambulo, MarcadorDeHandoff) {
		t.Fatal("o handoff está no JSON mas NÃO no preâmbulo: não chega ao modelo")
	}
	// (3) E O CONTEÚDO ÚTIL VIAJA: o objetivo e o passo que espera decisão humana.
	if !strings.Contains(preambulo, "entregar a compactação com recuperação provada") {
		t.Fatal("o objetivo da missão não está no preâmbulo")
	}
	if !strings.Contains(preambulo, "EXIGE APROVAÇÃO") {
		t.Fatal("o passo que exige aprovação não chega ao modelo")
	}
	// (4) COMO DADO, NUNCA COMO INSTRUÇÃO (§7-E): o marcador precede o conteúdo.
	iMarcador := strings.Index(b.Handoff, MarcadorDeHandoff)
	iObjetivo := strings.Index(b.Handoff, "entregar a compactação")
	if iMarcador < 0 || iObjetivo < 0 || iMarcador > iObjetivo {
		t.Fatal("o conteúdo aparece antes do marcador de dado")
	}
}

// O TETO É O DO PRÓPRIO CONTRATO: um handoff gigante não pode inchar o documento nem o
// preâmbulo. E o corte vem DECLARADO — nunca em silêncio.
func TestBootstrapRespeitaOTetoDoHandoff(t *testing.T) {
	h := Handoff{
		Objetivo:  "objetivo",
		Pendentes: []string{},
	}
	for i := range 400 {
		h.Pendentes = append(h.Pendentes, "pendência número "+strconvItoa(i))
	}
	in := bootstrapBase()
	in.Handoff = &h
	b, err := BuildContextBootstrap(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Handoff, "CORTADO") {
		t.Fatalf("o corte não foi declarado no documento (tamanho %d)", len(b.Handoff))
	}
	if !strings.Contains(b.SystemMessage(), "CORTADO") {
		t.Fatal("o aviso de corte não chegou ao preâmbulo")
	}
}
