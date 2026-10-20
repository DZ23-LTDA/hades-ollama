package agent

import (
	"errors"
	"strings"
	"testing"
)

func conversaDoGoal(pares ...string) []MensagemDaConversa {
	var out []MensagemDaConversa
	for i := 0; i+1 < len(pares); i += 2 {
		out = append(out, MensagemDaConversa{Papel: pares[i], Conteudo: pares[i+1]})
	}
	return out
}

// NEGATIVO: conversa vazia nao produz proposta -- inventar um goal a partir de nada e' pior do que nao
// responder.
func TestProporGoalRecusaConversaVazia(t *testing.T) {
	if _, err := ProporGoalDaConversa(nil); !errors.Is(err, ErrConversaVazia) {
		t.Fatalf("conversa nil devia dar ErrConversaVazia, veio %v", err)
	}
	if _, err := ProporGoalDaConversa(conversaDoGoal("user", "   ", "assistant", "\n")); !errors.Is(err, ErrConversaVazia) {
		t.Fatalf("conversa so' com espacos devia dar ErrConversaVazia, veio %v", err)
	}
}

// POSITIVO: a conversa nomeia metrica, alvo e periodo -> confianca alta, nada falta, e o motivo cita
// os tres achados.
func TestProporGoalComTudoQueAConversaDisse(t *testing.T) {
	c := conversaDoGoal(
		"user", "Quero reduzir a latência da API para 200ms neste trimestre.",
		"assistant", "Entendido: vou propor um goal de tempo de resposta.",
	)
	p, err := ProporGoalDaConversa(c)
	if err != nil {
		t.Fatalf("devia propor: %v", err)
	}
	if p.Goal.Metric != "tempo de resposta" {
		t.Fatalf("metrica devia ser 'tempo de resposta', veio %q", p.Goal.Metric)
	}
	if p.Goal.Target != 200 || p.Goal.Unit != "ms" {
		t.Fatalf("alvo devia ser 200 ms, veio %v %q", p.Goal.Target, p.Goal.Unit)
	}
	if p.Goal.Period != "trimestre" {
		t.Fatalf("periodo devia ser trimestre, veio %q", p.Goal.Period)
	}
	if len(p.Faltando) != 0 {
		t.Fatalf("nao devia faltar nada, faltou: %v", p.Faltando)
	}
	if p.Confianca != "alta" {
		t.Fatalf("confianca devia ser alta, veio %q", p.Confianca)
	}
	if !strings.Contains(p.Motivo, "tempo de resposta") || !strings.Contains(p.Motivo, "200") {
		t.Fatalf("o motivo tem de citar a metrica e o alvo: %q", p.Motivo)
	}
	if p.Goal.Title == "" {
		t.Fatal("o titulo nao pode ficar vazio quando ha' conversa")
	}
}

// A REGRA MAIS IMPORTANTE: o que a conversa NAO disse nao e' preenchido com um valor plausivel.
func TestProporGoalNaoInventaMetricaNemAlvo(t *testing.T) {
	c := conversaDoGoal("user", "Precisamos melhorar aquilo tudo, está ruim demais.")
	p, err := ProporGoalDaConversa(c)
	if err != nil {
		t.Fatalf("devia propor (mesmo sem dados): %v", err)
	}
	if p.Goal.Metric != "" {
		t.Fatalf("metrica NAO pode ser inventada, veio %q", p.Goal.Metric)
	}
	if p.Goal.Target != 0 {
		t.Fatalf("alvo NAO pode ser inventado, veio %v", p.Goal.Target)
	}
	if p.Goal.Period != "" {
		t.Fatalf("periodo NAO pode ser inventado, veio %q", p.Goal.Period)
	}
	for _, esperado := range []string{"metrica", "alvo", "periodo"} {
		if !contemString(p.Faltando, esperado) {
			t.Fatalf("%q devia estar em Faltando, veio %v", esperado, p.Faltando)
		}
	}
	if p.Confianca != "baixa" {
		t.Fatalf("confianca devia ser baixa, veio %q", p.Confianca)
	}
	if !strings.Contains(p.Motivo, "nada foi inferido") {
		t.Fatalf("o motivo tem de dizer que nada foi inferido: %q", p.Motivo)
	}
}

// E o goal incompleto e' recusado pelo store, que e' a segunda linha de defesa.
func TestGoalIncompletoERecusadoPeloStore(t *testing.T) {
	c := conversaDoGoal("user", "Quero reduzir a latência, mas ainda não sei para quanto.")
	p, err := ProporGoalDaConversa(c)
	if err != nil {
		t.Fatalf("devia propor: %v", err)
	}
	if p.Goal.Metric == "" {
		t.Fatal("a metrica devia ter sido encontrada nesta conversa")
	}
	if contemString(p.Faltando, "alvo") == false {
		t.Fatalf("o alvo devia faltar: %v", p.Faltando)
	}
	store, errStore := NewCompanyStore(t.TempDir())
	if errStore != nil {
		t.Fatalf("criar store: %v", errStore)
	}
	criada, errCriar := store.Create(Company{Name: "Empresa", OrganizationID: "org-teste"})
	if errCriar != nil {
		t.Fatalf("criar empresa: %v", errCriar)
	}
	if _, err := store.AddGoal(criada.ID, p.Goal); err != nil {
		t.Fatalf("este goal tem metrica e titulo, o store devia aceitar: %v", err)
	}
	// Sem metrica, o store tem de recusar -- e' a proteccao final contra goal inventado.
	vazio := p.Goal
	vazio.Metric = ""
	if _, err := store.AddGoal(criada.ID, vazio); err == nil {
		t.Fatal("o store tem de recusar goal sem metrica")
	}
}

func contemString(lista []string, procurado string) bool {
	for _, item := range lista {
		if strings.Contains(item, procurado) {
			return true
		}
	}
	return false
}
