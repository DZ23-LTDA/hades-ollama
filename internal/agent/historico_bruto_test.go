package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Testes do histórico bruto imutável (§2 do complemento universal de memória).
// Nenhum deles chama modelo, rede ou relógio de negócio.

func anexar(t *testing.T, h *HistoricoBruto, papel, conteudo string) RegistroDeHistorico {
	t.Helper()
	reg, err := h.Anexar(RegistroDeHistorico{Papel: papel, Conteudo: conteudo, ProjetoID: "p1"})
	if err != nil {
		t.Fatalf("anexar %q: %v", conteudo, err)
	}
	return reg
}

func TestHistoricoBrutoAcrescentaEDevolveCadeiaValida(t *testing.T) {
	h, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Fechar()

	primeiro := anexar(t, h, "user", "primeira")
	anexar(t, h, "assistant", "segunda")
	terceiro := anexar(t, h, "user", "terceira")

	if primeiro.Seq != 1 || primeiro.HashAnterior != "" {
		t.Fatalf("o primeiro registo tem de ser a ORIGEM da cadeia: %+v", primeiro)
	}
	if terceiro.Seq != 3 || terceiro.HashAnterior == "" {
		t.Fatalf("terceiro registo mal encadeado: %+v", terceiro)
	}
	registos, err := h.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 3 {
		t.Fatalf("LerTudo = %d registos, esperado 3", len(registos))
	}
	if err := VerificarCadeia(registos); err != nil {
		t.Fatalf("cadeia íntegra reprovada: %v", err)
	}
	// O chamador NÃO decide o Seq nem o hash: quem decide é o histórico.
	if registos[1].HashAnterior != registos[0].Hash {
		t.Fatal("o segundo registo não aponta para o hash do primeiro")
	}
}

func TestHistoricoBrutoRetomaAAposReabrir(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "hist.jsonl")
	h, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	anexar(t, h, "user", "antes da queda")
	anexar(t, h, "assistant", "ainda antes")
	if err := h.Fechar(); err != nil {
		t.Fatal(err)
	}

	// REABERTURA: é o que acontece quando o processo reinicia no meio de uma missão.
	reaberto, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer reaberto.Fechar()
	terceiro := anexar(t, reaberto, "user", "depois da queda")
	if terceiro.Seq != 3 {
		t.Fatalf("a retomada reiniciou a numeração: seq=%d (esperado 3)", terceiro.Seq)
	}
	registos, err := reaberto.LerTudo()
	if err != nil {
		t.Fatal(err)
	}
	if len(registos) != 3 {
		t.Fatalf("depois de reabrir, LerTudo = %d (o histórico anterior tem de sobreviver)", len(registos))
	}
	if err := VerificarCadeia(registos); err != nil {
		t.Fatalf("a cadeia não sobreviveu à reabertura: %v", err)
	}
	if registos[2].HashAnterior != registos[1].Hash {
		t.Fatal("a cadeia não foi continuada a partir do que estava em disco")
	}
}

func TestHistoricoBrutoDetetaAlteracaoDoConteudo(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "hist.jsonl")
	h, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	anexar(t, h, "user", "original")
	anexar(t, h, "assistant", "resposta")
	anexar(t, h, "user", "fim")
	h.Fechar()

	// ADULTERAÇÃO: altera-se o conteúdo de uma linha antiga, mantendo o hash gravado.
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	adulterado := strings.Replace(string(bruto), `"conteudo":"original"`, `"conteudo":"ADULTERADO"`, 1)
	if adulterado == string(bruto) {
		t.Fatal("o teste não conseguiu adulterar: o formato do ficheiro mudou")
	}
	if err := os.WriteFile(caminho, []byte(adulterado), 0o600); err != nil {
		t.Fatal(err)
	}

	reaberto, err := AbrirHistoricoBruto(caminho)
	if err == nil {
		reaberto.Fechar()
		t.Fatal("o histórico ACEITOU uma linha adulterada: a cadeia não está a ser conferida na abertura")
	}
	if !strings.Contains(err.Error(), "alterado") {
		t.Fatalf("recusou, mas por outro motivo: %v", err)
	}
}

func TestHistoricoBrutoRecusaLinhaCorrompida(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "hist.jsonl")
	h, err := AbrirHistoricoBruto(caminho)
	if err != nil {
		t.Fatal(err)
	}
	anexar(t, h, "user", "válida")
	h.Fechar()
	arquivo, err := os.OpenFile(caminho, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := arquivo.WriteString("{isto não é json}\n"); err != nil {
		t.Fatal(err)
	}
	arquivo.Close()

	// Uma linha má NÃO pode ser saltada em silêncio: um histórico que ignora o que não
	// entende não é auditável.
	if _, err := AbrirHistoricoBruto(caminho); err == nil {
		t.Fatal("abriu um histórico com linha corrompida")
	} else if !strings.Contains(err.Error(), "corrompida") {
		t.Fatalf("motivo inesperado: %v", err)
	}
}

func TestHistoricoBrutoRecusaRegistosInvalidos(t *testing.T) {
	h, err := AbrirHistoricoBruto(filepath.Join(t.TempDir(), "hist.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Anexar(RegistroDeHistorico{Conteudo: "sem papel"}); err == nil {
		t.Fatal("aceitou registo sem papel")
	}
	if _, err := AbrirHistoricoBruto("   "); err == nil {
		t.Fatal("aceitou caminho vazio")
	}
	// `Quando` vazio é preenchido pelo histórico (o chamador não é obrigado a saber as horas).
	reg := anexar(t, h, "user", "com hora implícita")
	if reg.Quando.IsZero() {
		t.Fatal("o histórico não preencheu Quando")
	}
	if err := h.Fechar(); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Anexar(RegistroDeHistorico{Papel: "user"}); err == nil {
		t.Fatal("aceitou anexar a um histórico fechado")
	}
	if err := h.Fechar(); err != nil {
		t.Fatalf("Fechar duas vezes devia ser inofensivo: %v", err)
	}
	_ = time.Now
}
