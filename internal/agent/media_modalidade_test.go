package agent

import (
	"errors"
	"strings"
	"testing"
)

// E-535 (Estagio 45, o `E-430` da matriz) - a recusa de modalidade, ANTES de tocar em disco ou rede.
//
// ***O QUE ESTE TESTE PROTEGE.*** Nao protege uma funcao: protege uma ORDEM. A recusa tem de acontecer **depois** de
// saber que modelo e **antes** de qualquer disco ou rede. ⇒ por isso o teste exercita o caminho com um `MediaManager`
// que **nao tem provider nenhum** - *se a recusa nao viesse primeiro, ele tentaria a rede e falharia por outro motivo,
// e o teste passaria a medir a coisa errada.*

func TestRecusarModalidadeDeMidiaFalhaFechado(t *testing.T) {
	casos := []struct {
		nome       string
		modalidade string
		modelo     string
		esperaErro bool
		porque     string
	}{
		{"imagem com modelo", "image", "dall-e-3", false, "*o caso legitimo tem de passar*"},
		{"video com modelo", "video", "veo-3", false, "*idem*"},
		{"imagem SEM modelo", "image", "", true, "*nao existe caminho que chame o provider com modelo vazio*"},
		{"video SEM modelo", "video", "", true, "*e' o `VideoModel` nao configurado - o caso mais provavel de todos*"},
		{"video com ESPACOS", "video", "   ", true, "*espacos nao sao um modelo*"},
		{"video para modelo TEXTUAL", "video", "llama3.1:8b", true, "*E-430 no seu exemplo literal*"},
		{"video para modelo de CHAT", "video", "gpt-4o", true, "*um modelo de chat nao gera video*"},
		{"imagem para modelo de VIDEO", "image", "veo-3", true, "*e o inverso tambem: a recusa nao e' so' de um lado*"},
	}
	for _, c := range casos {
		err := recusarModalidadeDeMidia(c.modalidade, c.modelo)
		if c.esperaErro && err == nil {
			t.Fatalf("ARMADILHA: %s (%s/%q) foi ACEITE - %s", c.nome, c.modalidade, c.modelo, c.porque)
		}
		if !c.esperaErro && err != nil {
			t.Fatalf("ARMADILHA: %s (%s/%q) foi RECUSADO (%v) - %s", c.nome, c.modalidade, c.modelo, err, c.porque)
		}
	}
}

func TestRecusarModalidadeDeMidiaDistingueAusenciaDeIncoerencia(t *testing.T) {
	// *Os dois erros existem separados porque as duas causas sao diferentes - e quem depura precisa de saber qual e'.*
	semModelo := recusarModalidadeDeMidia("video", "")
	if !errors.Is(semModelo, ErrMediaModelRequired) {
		t.Fatalf("ausencia de modelo devia dar ErrMediaModelRequired, veio %v", semModelo)
	}
	incoerente := recusarModalidadeDeMidia("video", "llama3.1:8b")
	if !errors.Is(incoerente, ErrMediaModalityMismatch) {
		t.Fatalf("modelo incoerente devia dar ErrMediaModalityMismatch, veio %v", incoerente)
	}
	// E a mensagem diz QUAL modelo, para nao obrigar a adivinhar.
	if !strings.Contains(incoerente.Error(), "llama3.1:8b") {
		t.Fatalf("a recusa nao nomeia o modelo recusado: %v", incoerente)
	}
}

// TestGenerateVideoRecusaAntesDeTocarNaRede e' a prova da ORDEM: com um provider vazio, a recusa tem de vir primeiro.
func TestGenerateVideoRecusaAntesDeTocarNaRede(t *testing.T) {
	// *Nenhum provider configurado:* se a recusa nao fosse a primeira coisa depois do fallback, o codigo tentaria a
	// rede e devolveria um erro de rede - e o teste estaria a medir a coisa errada.
	m := &MediaManager{}
	_, err := m.GenerateVideo(t.Context(), t.TempDir(), "um clipe curto", "")
	if err == nil {
		t.Fatal("ARMADILHA: GenerateVideo sem VideoModel configurado NAO recusou - *e' o sucesso falso que o E-430 proibe*")
	}
	if !errors.Is(err, ErrMediaModelRequired) {
		t.Fatalf("ARMADILHA: a recusa nao veio primeiro: %v - *se isto for um erro de rede, a ordem esta' errada*", err)
	}
}

func TestGenerateImageRecusaAntesDeTocarNaRede(t *testing.T) {
	m := &MediaManager{}
	_, err := m.GenerateImage(t.Context(), t.TempDir(), "uma imagem", "")
	if err == nil {
		t.Fatal("ARMADILHA: GenerateImage sem ImageModel configurado NAO recusou")
	}
	if !errors.Is(err, ErrMediaModelRequired) {
		t.Fatalf("ARMADILHA: a recusa nao veio primeiro: %v", err)
	}
}
