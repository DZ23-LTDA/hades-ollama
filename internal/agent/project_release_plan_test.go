package agent

// FASE 28.6: testes do elo entre ProjectDetector e ResolverDeToolchain.
//
// A LACUNA QUE ESTES TESTES FECHAM
//
// A E-1245 provou o detector com 15 testes e nada o chamava. Estes testes provam a CHAMADA:
//
//	*Um detector que ninguem consulta e' um teste com um nome melhor.*
//
// O que e' travado:
//
//   1. o alvo vem do PROJECTION, nao de um parametro;
//   2. cada alvo detectado tem um plano (ou um erro explicito), nunca e' omitido;
//   3. projection sem alvos da' `SemAlvos`, nao sucesso silencioso;
//   4. a ordem e' deterministica;
//   5. blockers de alvos diferentes sao deduplicados no resumo;
//   6. `MotivosLegiveis` explica cada exigencia nao saudavel;
//   7. os avisos do detector chegam ao plano do projection;
//   8. raiz invalida propaga o erro, nao devolve plano vazio;
//   9. o projection Go do HADES produz plano real (auto-teste).

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanoParaProjetoGo(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module exemplo\n\ngo 1.24\n")

	resolver := NovoResolverDeToolchain()
	plano, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("PlanoParaProjeto: %v", err)
	}
	if plano.SemAlvos {
		t.Fatal("projection Go devia declarar alvos")
	}
	if len(plano.Alvos) == 0 {
		t.Fatal("nenhum alvo planado")
	}
	// Todo alvo detectado tem de aparecer, pronto ou bloqueado -- nunca omitido.
	if len(plano.Alvos) != len(plano.Projeto.Alvos) {
		t.Fatalf("alvos planados=%d, detectados=%d", len(plano.Alvos), len(plano.Projeto.Alvos))
	}
	total := len(plano.AlvosProntos) + len(plano.AlvosBloqueados)
	if total != len(plano.Alvos) {
		t.Fatalf("prontos+bloqueados=%d != alvos=%d", total, len(plano.Alvos))
	}
	t.Logf("go: prontos=%v bloqueados=%v explicacao=%q",
		plano.AlvosProntos, plano.AlvosBloqueados, plano.Explicacao)
}

// O alvo vem do PROJECTION. Um repo Node nao produz plano de Windows-installer por magia.
func TestPlanoParaProjetoAlvoVemDoProjeto(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"site"}`)
	escreverFicheiroDeTeste(t, raiz, "pnpm-lock.yaml", "")

	resolver := NovoResolverDeToolchain()
	plano, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("PlanoParaProjeto: %v", err)
	}
	// Node puro declara `web`; nao declara `android`.
	for _, a := range plano.Alvos {
		if a.Alvo == "android" {
			t.Fatalf("alvo android apareceu num projection Node sem Expo/Android: %v", plano.Alvos)
		}
	}
	achouWeb := false
	for _, a := range plano.Alvos {
		if a.Alvo == "web" {
			achouWeb = true
		}
	}
	if !achouWeb {
		t.Fatalf("alvo web nao planado; veio %v", plano.Alvos)
	}
}

// Projection sem alvos reconheciveis: `SemAlvos`, nunca sucesso mudo.
func TestPlanoParaProjetoSemAlvos(t *testing.T) {
	raiz := t.TempDir()
	// Python nao tem alvo de instalador no catalogo actual.
	escreverFicheiroDeTeste(t, raiz, "pyproject.toml", "[project]\nname=\"x\"\n")

	resolver := NovoResolverDeToolchain()
	plano, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("PlanoParaProjeto: %v", err)
	}
	if !plano.SemAlvos {
		t.Fatalf("projection Python devia dar SemAlvos; alvos=%v", plano.Alvos)
	}
	if len(plano.Alvos) != 0 {
		t.Fatalf("SemAlvos com alvos planados: %v", plano.Alvos)
	}
	if plano.Explicacao == "" {
		t.Fatal("SemAlvos sem explicacao")
	}
}

// Determinismo: duas chamadas, mesma ordem.
func TestPlanoParaProjetoDeterministico(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module x\n")
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"web"}`)

	resolver := NovoResolverDeToolchain()
	primeiro, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("primeira: %v", err)
	}
	for i := range 2 {
		seguinte, err := resolver.PlanoParaProjeto(context.Background(), raiz)
		if err != nil {
			t.Fatalf("chamada %d: %v", i, err)
		}
		if len(seguinte.Alvos) != len(primeiro.Alvos) {
			t.Fatalf("chamada %d: alvos variaram", i)
		}
		for j := range seguinte.Alvos {
			if seguinte.Alvos[j].Alvo != primeiro.Alvos[j].Alvo {
				t.Fatalf("chamada %d: ordem mudou em %d: %q vs %q", i, j,
					seguinte.Alvos[j].Alvo, primeiro.Alvos[j].Alvo)
			}
		}
		for j := range seguinte.AlvosProntos {
			if j >= len(primeiro.AlvosProntos) || seguinte.AlvosProntos[j] != primeiro.AlvosProntos[j] {
				t.Fatalf("chamada %d: ordem de prontos mudou", i)
			}
		}
	}
}

// Raiz invalida propaga o erro; nao devolve plano vazio como se fosse um projection sem alvos.
func TestPlanoParaProjetoRaizInvalida(t *testing.T) {
	resolver := NovoResolverDeToolchain()
	if _, err := resolver.PlanoParaProjeto(
		context.Background(), filepath.Join(t.TempDir(), "nao-existe")); err == nil {
		t.Fatal("raiz inexistente devia dar erro")
	}
	if _, err := resolver.PlanoParaProjeto(context.Background(), ""); err == nil {
		t.Fatal("raiz vazia devia dar erro")
	}
}

// Os avisos do detector chegam ao plano do projection: quem le' o plano tem de ver o que a
// deteccao nao conseguiu ver.
func TestPlanoParaProjetoLevaAvisosDoDetector(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"dependencies":{"expo":"~53"}}`)
	if err := os.MkdirAll(filepath.Join(raiz, "android"), 0o700); err != nil {
		t.Fatalf("mkdir android: %v", err)
	}

	resolver := NovoResolverDeToolchain()
	plano, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("PlanoParaProjeto: %v", err)
	}
	if len(plano.Avisos) == 0 {
		t.Fatal("avisos do detector nao chegaram ao plano")
	}
}

// `ResumoDeBlockers` deduplica: o mesmo `go` em falta nao pode aparecer uma vez por alvo.
func TestResumoDeBlockersDeduplica(t *testing.T) {
	plano := PlanoDoProjeto{
		Alvos: []PlanoPorAlvo{
			{Alvo: "windows", Plano: CapabilityPlan{Blockers: []string{"go", "inno-setup"}}},
			{Alvo: "linux", Plano: CapabilityPlan{Blockers: []string{"go", "docker"}}},
		},
	}
	got := plano.ResumoDeBlockers()
	esperado := []string{"docker", "go", "inno-setup"}
	if len(got) != len(esperado) {
		t.Fatalf("resumo=%v; esperado %v", got, esperado)
	}
	for i := range esperado {
		if got[i] != esperado[i] {
			t.Fatalf("resumo=%v; esperado %v", got, esperado)
		}
	}
}

// Um alvo com erro de plano tem de aparecer no resumo, nao desaparecer.
func TestResumoDeBlockersIncluiErroDePlano(t *testing.T) {
	plano := PlanoDoProjeto{
		Alvos: []PlanoPorAlvo{
			{Alvo: "oci", ErroDePlano: "alvo desconhecido"},
		},
	}
	got := plano.ResumoDeBlockers()
	if len(got) != 1 || got[0] != "oci: alvo desconhecido" {
		t.Fatalf("resumo=%v", got)
	}
}

// `MotivosLegiveis` explica cada exigencia nao saudavel, com nome e motivo.
func TestMotivosLegiveis(t *testing.T) {
	plano := PlanoDoProjeto{
		Alvos: []PlanoPorAlvo{
			{Alvo: "windows", Plano: CapabilityPlan{Probes: []ToolProbe{
				{
					Requisito: ToolRequirement{Nome: "go", Motivo: "compilar o binario"},
					Estado:    ToolHealthy,
				},
				{
					Requisito: ToolRequirement{Nome: "inno-setup", Motivo: "compilar o instalador"},
					Estado:    ToolAbsent,
					Detalhe:   "nao encontrado no PATH",
				},
			}}},
		},
	}
	got := plano.MotivosLegiveis()
	if len(got) != 1 {
		t.Fatalf("motivos=%v; esperado so' o ausente", got)
	}
	// A sonda saudavel nao entra.
	for _, m := range got {
		if m == "windows: go -- compilar o binario" {
			t.Fatalf("sonda saudavel entrou: %v", got)
		}
	}
	if !stringContains(got[0], "inno-setup") || !stringContains(got[0], "compilar o instalador") {
		t.Fatalf("motivo sem nome ou razao: %q", got[0])
	}
}

// Auto-teste: o proprio repo HADES.
func TestPlanoParaProjetoDoRepoHades(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.mod")); err != nil {
		t.Skipf("nao estou dentro do repo HADES (%v)", err)
	}
	resolver := NovoResolverDeToolchain()
	plano, err := resolver.PlanoParaProjeto(context.Background(), raiz)
	if err != nil {
		t.Fatalf("plano do repo HADES: %v", err)
	}
	if plano.SemAlvos {
		t.Fatal("repo HADES sem alvos")
	}
	t.Logf("repo HADES: alvos=%v prontos=%v bloqueados=%v",
		func() []string {
			var n []string
			for _, a := range plano.Alvos {
				n = append(n, a.Alvo)
			}
			return n
		}(), plano.AlvosProntos, plano.AlvosBloqueados)
}
