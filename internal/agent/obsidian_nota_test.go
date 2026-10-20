package agent

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func metaDeTeste(escopo EscopoDeMemoria) MetaDaNota {
	return MetaDaNota{
		MemoryID:   "mem-0001",
		Escopo:     escopo,
		Project:    "hades",
		Author:     "dz23",
		CreatedAt:  "2026-10-19T00:00:00Z",
		UpdatedAt:  "2026-10-19T00:00:00Z",
		Confidence: "alta",
	}
}

func fatoDeTeste() Memoria {
	return Memoria{
		ID:       "mem-0001",
		Rotulo:   MemoriaFatoVerificado,
		Conteudo: "E-1433 mediu 129 PASS na familia de memoria.",
		Fonte:    "go test ./internal/agent -run Memor",
	}
}

// Sem escopo nao ha nota. E' a negacao por defeito: nao existe escopo "implicito" a inferir do caminho.
func TestNotaDeMemoriaRecusaSemEscopo(t *testing.T) {
	_, err := MontarNota(metaDeTeste(""), fatoDeTeste())
	if !errors.Is(err, ErrEscopoAusente) {
		t.Fatalf("esperava ErrEscopoAusente, veio %v", err)
	}
}

// Escopo fora da lista fechada nao e' aceito por ser "parecido".
func TestNotaDeMemoriaRecusaEscopoDesconhecido(t *testing.T) {
	_, err := MontarNota(metaDeTeste("PROJETO"), fatoDeTeste())
	if !errors.Is(err, ErrEscopoDesconhecido) {
		t.Fatalf("esperava ErrEscopoDesconhecido, veio %v", err)
	}
}

// O adaptador NAO tem regra de rotulo propria: quem recusa FACT_VERIFIED sem fonte e' `Memoria.Validar`.
// Se este teste passar a falhar, alguem duplicou a regra e as duas podem divergir.
func TestNotaDeMemoriaDefereValidacaoDoRotulo(t *testing.T) {
	m := fatoDeTeste()
	m.Fonte = ""
	_, err := MontarNota(metaDeTeste(EscopoProject), m)
	if !errors.Is(err, ErrPromocaoSemProva) {
		t.Fatalf("esperava ErrPromocaoSemProva (de Memoria.Validar), veio %v", err)
	}
}

func TestNotaDeMemoriaRoundTrip(t *testing.T) {
	original := fatoDeTeste()
	meta := metaDeTeste(EscopoProject)

	dados, err := MontarNota(meta, original)
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}

	meta2, volta, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if volta.ID != original.ID || volta.Rotulo != original.Rotulo || volta.Fonte != original.Fonte {
		t.Fatalf("identidade perdida: %#v", volta)
	}
	if strings.TrimSpace(volta.Conteudo) != strings.TrimSpace(original.Conteudo) {
		t.Fatalf("conteudo perdido: %q", volta.Conteudo)
	}
	if meta2.Escopo != meta.Escopo {
		t.Fatalf("escopo perdido: %q", meta2.Escopo)
	}
	if meta2.ContentHash != HashDoConteudo(original.Conteudo) {
		t.Fatalf("content_hash nao corresponde ao corpo: %q", meta2.ContentHash)
	}
}

// Determinismo: mesma entrada, mesmos bytes. Se isto quebrar, o hash da nota deixa de significar alguma coisa.
func TestMontarNotaEhDeterminista(t *testing.T) {
	a, err := MontarNota(metaDeTeste(EscopoGoal), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	b, err := MontarNota(metaDeTeste(EscopoGoal), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("duas montagens da mesma entrada deram bytes diferentes")
	}
}

// A discriminacao que interessa: reanexar a MESMA memory_id nao acrescenta bytes nem os altera.
func TestAnexoIdempotenteNaoDuplicaNemReescreve(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	m := fatoDeTeste()

	primeiro, anexou, err := AnexarNotaIdempotente(nil, meta, m)
	if err != nil || !anexou {
		t.Fatalf("primeira anexacao devia ter anexado: anexou=%v err=%v", anexou, err)
	}

	// Mesma chave, corpo DIFERENTE: a idempotencia nao pode depender do corpo.
	m2 := m
	m2.Conteudo = "outro texto completamente diferente"

	segundo, anexou2, err := AnexarNotaIdempotente(primeiro, meta, m2)
	if err != nil {
		t.Fatalf("segunda anexacao: %v", err)
	}
	if anexou2 {
		t.Fatalf("reanexar a mesma memory_id nao podia anexar")
	}
	if !bytes.Equal(primeiro, segundo) {
		t.Fatalf("reanexar a mesma memory_id alterou os bytes")
	}
	if n := strings.Count(string(segundo), "memory_id: mem-0001"); n != 1 {
		t.Fatalf("esperava 1 ocorrencia da memory_id, encontrei %d", n)
	}
}

// memory_id diferente anexa de verdade -- senao o teste acima passaria por o append estar sempre quebrado.
func TestAnexoDeIdDiferenteAcrescenta(t *testing.T) {
	primeiro, _, err := AnexarNotaIdempotente(nil, metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("preparacao: %v", err)
	}
	outra := fatoDeTeste()
	outra.ID = "mem-0002"
	meta := metaDeTeste(EscopoProject)
	meta.MemoryID = "mem-0002"

	segundo, anexou, err := AnexarNotaIdempotente(primeiro, meta, outra)
	if err != nil || !anexou {
		t.Fatalf("nova memory_id devia anexar: anexou=%v err=%v", anexou, err)
	}
	if len(segundo) <= len(primeiro) {
		t.Fatalf("os bytes nao cresceram")
	}
}

// Campo fora do schema e' recusado. Um campo ignorado em silencio e' um campo que o leitor julga ter lido.
func TestLerNotaRecusaCampoDesconhecido(t *testing.T) {
	dados := []byte("---\nmemory_id: mem-0001\nschema_version: 1\nscope: PROJECT\ncampo_inventado: x\n---\ncorpo\n")
	_, _, err := LerNota(dados)
	if !errors.Is(err, ErrCampoDesconhecido) {
		t.Fatalf("esperava ErrCampoDesconhecido, veio %v", err)
	}
}

func TestLerNotaRecusaFrontmatterAusente(t *testing.T) {
	_, _, err := LerNota([]byte("texto solto sem cabecalho\n"))
	if !errors.Is(err, ErrNotaSemFrontmatter) {
		t.Fatalf("esperava ErrNotaSemFrontmatter, veio %v", err)
	}
}

// author e' quem escreveu a nota; aprovador e' quem assinou a decisao. Se o autor virasse aprovador na leitura,
// uma decisao sem assinatura ganharia assinatura ao passar pelo disco.
func TestAutorNaoViraAprovador(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	meta.Author = "quem-escreveu"
	m := Memoria{ID: "mem-0002", Rotulo: MemoriaProposta, Conteudo: "proposta sem aprovador"}
	dados, err := MontarNota(meta, m)
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	_, volta, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if volta.Aprovador != "" {
		t.Fatalf("o autor da nota virou aprovador da decisao: %q", volta.Aprovador)
	}
}

// O hash gravado tem de bater com o corpo lido. Se nao batesse (por causa do "\n" final), toda a leitura
// pareceria uma edicao humana -- um alarme que toca sempre nao avisa de nada.
func TestIntegridadeDaNotaAceitaLeituraLimpa(t *testing.T) {
	dados, err := MontarNota(metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	meta, m, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	if err := IntegridadeDaNota(meta, m); err != nil {
		t.Fatalf("leitura limpa acusada de adulteracao: %v", err)
	}
}

// Edicao humana no corpo tem de ser DETETADA -- e nao apagada.
func TestIntegridadeDaNotaDetectaEdicaoHumana(t *testing.T) {
	dados, err := MontarNota(metaDeTeste(EscopoProject), fatoDeTeste())
	if err != nil {
		t.Fatalf("MontarNota: %v", err)
	}
	meta, m, err := LerNota(dados)
	if err != nil {
		t.Fatalf("LerNota: %v", err)
	}
	m.Conteudo = m.Conteudo + "linha acrescentada a mao no Obsidian\n"
	if err := IntegridadeDaNota(meta, m); !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("esperava ErrConteudoAlterado, veio %v", err)
	}
}

// Nota sem content_hash nao pode passar por integra: ausencia de prova nao e' prova.
func TestIntegridadeDaNotaRecusaSemHash(t *testing.T) {
	meta := metaDeTeste(EscopoProject)
	meta.ContentHash = ""
	if err := IntegridadeDaNota(meta, fatoDeTeste()); !errors.Is(err, ErrConteudoAlterado) {
		t.Fatalf("esperava ErrConteudoAlterado, veio %v", err)
	}
}
