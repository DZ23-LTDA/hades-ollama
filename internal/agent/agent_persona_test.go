package agent

import (
	"errors"
	"strings"
	"testing"
)

func personaDeTeste() AgentPersona {
	return AgentPersona{
		ID:          "revisor-de-codigo",
		Slug:        "code-reviewer",
		Name:        "Revisor de Codigo",
		Version:     "1.2.0",
		Description: "Revê diffs antes de entravam no branch principal.",
		Division:    "engenharia",
		Category:    "review",
		Source:      "hades-catalogo",
		SourceRef:   "commit:abc123",
		License:     "MIT",
		Prompt:      "Leia o diff e aponte o defeito, não o estilo.",
		Capabilities: []string{
			"read-diff",
			"comment",
			"run-tests",
		},
		Scopes:           []string{"project", "goal"},
		Tools:            []string{"git", "test-runner"},
		RequiresApproval: []string{"post-comment-external"},
		Forbidden:        []string{"push", "merge", "force-push"},
	}
}

// *A mesma entrada canónica, na mesma versão, tem de produzir bytes idênticos -- senão a reconciliação por
// hash detecta a hora, não o drift.*
func TestRenderEhDeterministaByteAByte(t *testing.T) {
	p := personaDeTeste()
	primeiro, err := RenderHades(p)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := RenderHades(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(primeiro) != string(segundo) {
		t.Fatal("dois renders da mesma persona produziram bytes diferentes")
	}
	// E o mesmo para os outros formatos.
	for _, alvo := range []TargetFormat{TargetCodex, TargetClaudeCode} {
		a, err := Render(p, alvo)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Render(p, alvo)
		if err != nil {
			t.Fatal(err)
		}
		if string(a) != string(b) {
			t.Fatalf("%s: dois renders produziram bytes diferentes", alvo)
		}
	}
}

// *Ordem em que o autor escreveu a lista não é conteúdo: o conjunto é o mesmo.*
func TestHashNaoDependeDaOrdemDosConjuntos(t *testing.T) {
	a := personaDeTeste()
	b := personaDeTeste()
	b.Capabilities = []string{"run-tests", "read-diff", "comment"}
	b.Forbidden = []string{"force-push", "merge", "push"}
	b.Scopes = []string{"goal", "project"}

	ha, err := a.ContentHash()
	if err != nil {
		t.Fatal(err)
	}
	hb, err := b.ContentHash()
	if err != nil {
		t.Fatal(err)
	}
	if ha != hb {
		t.Fatalf("ordem diferente gerou hash diferente:\n  %s\n  %s", ha, hb)
	}

	// E uma capacidade A MAIS tem de gerar hash diferente.
	c := personaDeTeste()
	c.Capabilities = append(c.Capabilities, "delete-branch")
	hc, err := c.ContentHash()
	if err != nil {
		t.Fatal(err)
	}
	if hc == ha {
		t.Fatal("acrescentar uma capacidade NAO mudou o hash: o hash nao cobre o conteudo")
	}
}

func TestHashCcobreConteudoENaoFormato(t *testing.T) {
	// Duplicados não mudam o conjunto.
	a := personaDeTeste()
	b := personaDeTeste()
	b.Capabilities = append(b.Capabilities, "comment", "read-diff") // repetidos

	ha, _ := a.ContentHash()
	hb, _ := b.ContentHash()
	if ha != hb {
		t.Fatal("duplicar uma capacidade mudou o hash; o hash nao e' do conjunto")
	}

	// Mas mudar o prompt MUDA.
	c := personaDeTeste()
	c.Prompt += " Seja breve."
	hc, _ := c.ContentHash()
	if hc == ha {
		t.Fatal("mudar o prompt NAO mudou o hash")
	}
}

func TestValidateRecusaOQueNaoSeAudita(t *testing.T) {
	casos := []struct {
		nome   string
		mexer  func(*AgentPersona)
		trecho string
	}{
		{"sem id", func(p *AgentPersona) { p.ID = "" }, "id is required"},
		{"sem nome", func(p *AgentPersona) { p.Name = "" }, "name is required"},
		{"sem versao", func(p *AgentPersona) { p.Version = "" }, "version is required"},
		{"sem fonte", func(p *AgentPersona) { p.Source = "" }, "source is required"},
		{"sem licenca", func(p *AgentPersona) { p.License = "" }, "license is required"},
		// *Um id que reescreve o caminho e' um caminho, nao um id.*
		{"id com barra", func(p *AgentPersona) { p.ID = "a/b" }, "path separators"},
		{"id com traversal", func(p *AgentPersona) { p.ID = ".." }, "path separators or .."},
		{"id com backslash", func(p *AgentPersona) { p.ID = `a\b` }, "path separators"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			p := personaDeTeste()
			caso.mexer(&p)
			err := p.Validate()
			if err == nil {
				t.Fatal("esperava erro de validacao")
			}
			if !errors.Is(err, ErrPersonaInvalid) {
				t.Fatalf("erro nao e' ErrPersonaInvalid: %v", err)
			}
			if !strings.Contains(err.Error(), caso.trecho) {
				t.Fatalf("mensagem %q nao contem %q", err.Error(), caso.trecho)
			}
		})
	}
}

func TestRenderRecusaTargetDesconhecido(t *testing.T) {
	_, err := Render(personaDeTeste(), TargetFormat("figma"))
	if !errors.Is(err, ErrPersonaUnsupportedTarget) {
		t.Fatalf("esperava ErrPersonaUnsupportedTarget, obtive %v", err)
	}
	if FormatSupport(TargetFormat("figma")) != SupportUnsupported {
		t.Fatal("formato desconhecido nao declarado UNSUPPORTED")
	}
}

// *Uma capacidade que some no render e' uma permissao que o utilizador pensa ter pedido e nao pediu.*
func TestFormatoQueNaoRepresentaCapacidadeNaoAsInventa(t *testing.T) {
	// O Codex nao tem campo de escopos. O render NAO pode escrever "escopo" como se tivesse sido concedido.
	corpo, err := Render(personaDeTeste(), TargetCodex)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(corpo), "Escopos permitidos") {
		t.Fatal("o render Codex inventou uma seccao de escopos que o formato nao tem")
	}
	// Mas as ferramentas E os bloqueios tem de sobreviver.
	if !strings.Contains(string(corpo), "Ferramentas necessarias") {
		t.Fatal("o render Codex perdeu as ferramentas")
	}
	if !strings.Contains(string(corpo), "Proibido") {
		t.Fatal("o render Codex perdeu as accoes proibidas")
	}
}

// *Não avaliado e' um estado, e e' diferente de aprovado.*
func TestPersonaNaoAvaliadaDizIsso(t *testing.T) {
	corpo, err := RenderHades(personaDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(corpo), "NAO_AVALIADO") {
		t.Fatal("persona sem avaliacao nao declarou NAO_AVALIADO")
	}
	if strings.Contains(string(corpo), "APROVADO") {
		t.Fatal("o render declarou APROVADO sem avaliacao")
	}
}

func TestClassifyDrift(t *testing.T) {
	p := personaDeTeste()
	rendido, err := RenderHades(p)
	if err != nil {
		t.Fatal(err)
	}
	hashOrigem, err := p.ContentHash()
	if err != nil {
		t.Fatal(err)
	}
	entrada := PersonaLedgerEntry{
		PersonaID:    p.ID,
		Version:      p.Version,
		Target:       TargetHades,
		Destination:  "projeto/.hades/agents/revisor.md",
		Scope:        "project",
		SourceHash:   hashOrigem,
		RenderedHash: hashBytes(rendido),
	}

	casos := []struct {
		nome     string
		atual    []byte
		esperado []byte
		quer     DriftState
	}{
		{"igual ao que o HADES escreveu", rendido, rendido, DriftCurrent},
		{"ficheiro apagado", nil, rendido, DriftRemoved},
		// *Editado fora do HADES: actualizar por cima apagaria trabalho de uma pessoa.*
		{"editado a mao", []byte("conteudo diferente"), rendido, DriftLocallyModified},
		{"origem mudou, disco intacto", rendido, []byte("render novo"), DriftOutdated},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if got := ClassifyDrift(entrada, caso.atual, caso.esperado); got != caso.quer {
				t.Fatalf("estado=%s quer=%s", got, caso.quer)
			}
		})
	}

	// *Sem hashes nao ha' reconciliacao possivel. Dizer CURRENT seria inventar.*
	semHash := entrada
	semHash.RenderedHash = ""
	if got := ClassifyDrift(semHash, rendido, rendido); got != DriftForeign {
		t.Fatalf("registo sem hash classificado como %s; devia ser FOREIGN_UNMANAGED", got)
	}
}

// *Nunca "sobrescrever" por omissao: cada estado tem uma accao segura, e a de MODIFIED preserva a edicao.*
func TestDescribeDriftNuncaMandaSobrescrever(t *testing.T) {
	for _, estado := range []DriftState{
		DriftCurrent, DriftOutdated, DriftLocallyModified, DriftRemoved,
		DriftForeign, DriftConflicted, DriftBroken,
	} {
		texto := DescribeDrift(estado)
		if texto == "" {
			t.Fatalf("%s sem accao descrita", estado)
		}
		minusculo := strings.ToLower(texto)
		if strings.Contains(minusculo, "sobrescreva") || strings.Contains(minusculo, "force") {
			t.Fatalf("%s manda sobrescrever: %q", estado, texto)
		}
	}
	// E o estado de edicao local diz explicitamente para preservar.
	if !strings.Contains(DescribeDrift(DriftLocallyModified), "Preserve") {
		t.Fatal("MODIFIED nao manda preservar a edicao")
	}
	if !strings.Contains(DescribeDrift(DriftForeign), "NAO remova") {
		t.Fatal("FOREIGN nao manda nao remover")
	}
	// *O ultimo a escrever nao ganha por ser o ultimo.*
	if !strings.Contains(DescribeDrift(DriftConflicted), "uma pessoa") {
		t.Fatal("CONFLICTED nao manda resolver com uma pessoa")
	}
}
