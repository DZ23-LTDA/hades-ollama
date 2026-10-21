package agent

import (
	"errors"
	"sync"
	"testing"
)

// POSITIVO base: um pare novo nao bloqueia -- e' o estado de quem esta' a trabalhar.
func TestPareNovoNaoBloqueia(t *testing.T) {
	p := NovoPareDeEmergencia()
	if p.Bloqueia() {
		t.Fatal("um pare novo nao pode bloquear")
	}
	if e := p.Estado(); e.Acionado {
		t.Fatalf("estado inicial tem de ser desarmado: %+v", e)
	}
}

// NEGATIVO: sem motivo nao ha' pare -- nem alteracao de estado.
func TestPareSemMotivoRecusaENaoMudaEstado(t *testing.T) {
	p := NovoPareDeEmergencia()
	if err := p.Acionar("", "operador"); !errors.Is(err, ErrPareSemMotivo) {
		t.Fatalf("motivo vazio devia dar ErrPareSemMotivo, veio %v", err)
	}
	if p.Bloqueia() {
		t.Fatal("um pare recusado nao pode deixar a execucao bloqueada")
	}
	if e := p.Estado(); e.Tentativas != 0 {
		t.Fatalf("uma tentativa recusada nao conta como tentativa de pare, veio %d", e.Tentativas)
	}
}

// POSITIVO: com motivo, aciona e bloqueia, e o estado guarda motivo, autor e instante.
func TestPareAcionadoBloqueiaEGuardaMotivo(t *testing.T) {
	p := NovoPareDeEmergencia()
	quando := p.agora
	if err := p.Acionar("vazamento de dados entre tenants", "operador"); err != nil {
		t.Fatalf("devia acionar: %v", err)
	}
	if !p.Bloqueia() {
		t.Fatal("acionado tem de bloquear (fail-closed)")
	}
	e := p.Estado()
	if e.Motivo != "vazamento de dados entre tenants" || e.PorQuem != "operador" {
		t.Fatalf("estado incompleto: %+v", e)
	}
	if e.Quando.IsZero() || quando == nil {
		t.Fatal("o instante do pare tem de ficar registado")
	}
}

// O PRIMEIRO MOTIVO VENCE: acionar outra vez conta tentativa mas nao reescreve a causa.
func TestSegundoAcionamentoNaoReescreveAMotivo(t *testing.T) {
	p := NovoPareDeEmergencia()
	if err := p.Acionar("causa original", "operador"); err != nil {
		t.Fatal(err)
	}
	if err := p.Acionar("suposicao posterior", "outro"); err != nil {
		t.Fatal(err)
	}
	e := p.Estado()
	if e.Motivo != "causa original" {
		t.Fatalf("a causa original tem de vencer, veio %q", e.Motivo)
	}
	if e.Tentativas != 2 {
		t.Fatalf("as tentativas tem de ser contadas (2), veio %d", e.Tentativas)
	}
}

// Rearmar exige justificacao -- voltar a ligar e' a accao perigosa.
func TestRearmarExigeJustificacao(t *testing.T) {
	p := NovoPareDeEmergencia()
	if err := p.Acionar("teste", "operador"); err != nil {
		t.Fatal(err)
	}
	if err := p.Rearmar("operador", ""); !errors.Is(err, ErrPareSemMotivo) {
		t.Fatalf("rearmar sem justificacao devia dar ErrPareSemMotivo, veio %v", err)
	}
	if !p.Bloqueia() {
		t.Fatal("um rearme recusado NAO pode desbloquear")
	}
	if err := p.Rearmar("operador", "causa corrigida no commit X"); err != nil {
		t.Fatalf("com justificacao devia rearmar: %v", err)
	}
	if p.Bloqueia() {
		t.Fatal("depois de rearmado nao pode bloquear")
	}
	e := p.Estado()
	if e.RearmadoPor != "operador" || e.Justificacao == "" || e.RearmadoQuando.IsZero() {
		t.Fatalf("o rearme tem de ficar registado: %+v", e)
	}
	// O motivo original NAO se perde com o rearme.
	if e.Motivo != "teste" {
		t.Fatalf("o motivo original tem de sobreviver ao rearme, veio %q", e.Motivo)
	}
}

// Rearmar o que nao esta' parado e' no-op, e nao cria registo falso de rearme.
func TestRearmarQuandoNaoEstaParadoENoOp(t *testing.T) {
	p := NovoPareDeEmergencia()
	if err := p.Rearmar("operador", "so' por precaucao"); err != nil {
		t.Fatalf("devia ser no-op sem erro: %v", err)
	}
	if e := p.Estado(); e.RearmadoPor != "" {
		t.Fatalf("nao devia registar rearme do que nao estava parado: %+v", e)
	}
}

// Concorrencia: acionar e consultar de varias goroutines nao rebenta nem perde o bloqueio.
func TestPareConcorrente(t *testing.T) {
	p := NovoPareDeEmergencia()
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Acionar("corrida", "goroutine"); err != nil {
				t.Errorf("acionar concorrente: %v", err)
			}
			_ = p.Bloqueia()
		}()
	}
	wg.Wait()
	if !p.Bloqueia() {
		t.Fatal("depois de 50 acionamentos o pare tem de bloquear")
	}
	if e := p.Estado(); e.Tentativas != 50 {
		t.Fatalf("50 tentativas deviam ser contadas, veio %d", e.Tentativas)
	}
}
