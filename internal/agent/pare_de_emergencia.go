package agent

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrPareSemMotivo e' devolvido quando se tenta acionar ou rearmar sem motivo/justificacao.
//
// *Um pare que nao diz porque parou nao pode ser auditado nem revertido -- e' indistinguivel de uma
// avaria.*
var ErrPareSemMotivo = errors.New("pare de emergencia exige motivo")

// ErrExecucaoParada e' devolvido ao executor quando o pare esta' acionado.
//
// E' distinto de `ErrPareSemMotivo`: um diz *porque nao se pode parar sem motivo*, o outro diz
// *a execucao esta' parada por este motivo*. Confundir os dois apagaria a causa de quem le o erro.
var ErrExecucaoParada = errors.New("execucao parada pelo pare de emergencia")

// EstadoDoPare e' o retrato do pare num instante.
type EstadoDoPare struct {
	Acionado       bool      `json:"acionado"`
	Motivo         string    `json:"motivo,omitempty"`
	PorQuem        string    `json:"por_quem,omitempty"`
	Quando         time.Time `json:"quando,omitempty"`
	Tentativas     int       `json:"tentativas"`
	RearmadoPor    string    `json:"rearmado_por,omitempty"`
	Justificacao   string    `json:"justificacao,omitempty"`
	RearmadoQuando time.Time `json:"rearmado_quando,omitempty"`
}

// PareDeEmergencia e' o travao de mao do lado do dominio.
//
// PORQUE ISTO EXISTE, E O QUE NAO E'
//
// A linha C2 do adendo pede «botao de emergencia» para execucao autonoma, e a medicao do `E-1483`
// mostrou que ele **nao existe**: `ComputerUse` e' um componente a instalar, nao um perfil com travao.
// Esta primitiva e' a parte que se pode escrever sem inventar capacidade: o ESTADO e as regras de
// transicao. Nao ha' botao, rota nem UI -- e por isso ninguem a pode confundir com o botao.
//
// AS REGRAS, E PORQUE CADA UMA
//
//  1. **fail-closed**: acionado bloqueia; nao ha' caminho em que "acionado" signifique "talvez".
//  2. **motivo obrigatorio**: um pare sem motivo nao pode ser auditado nem explicado a quem chega
//     depois.
//  3. **o primeiro motivo vence**: acionar duas vezes nao reescreve a causa; conta tentativas. A
//     causa original e' a que interessa a quem investiga.
//  4. **rearmar exige justificacao e autor**: voltar a ligar e' a accao perigosa, nao a de parar.
type PareDeEmergencia struct {
	mu    sync.Mutex
	estao EstadoDoPare
	agora func() time.Time
}

// NovoPareDeEmergencia cria o pare DESARMADO (nao bloqueia).
func NovoPareDeEmergencia() *PareDeEmergencia {
	return &PareDeEmergencia{agora: time.Now}
}

// Acionar para a execucao. Motivo vazio devolve ErrPareSemMotivo e NAO altera o estado.
func (p *PareDeEmergencia) Acionar(motivo, porQuem string) error {
	if p == nil {
		return ErrPareSemMotivo
	}
	if motivo == "" {
		return ErrPareSemMotivo
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.estao.Tentativas++
	if !p.estao.Acionado {
		p.estao.Acionado = true
		p.estao.Motivo = motivo
		p.estao.PorQuem = porQuem
		p.estao.Quando = p.agora().UTC()
	}
	// Ja' acionado: o primeiro motivo vence; a tentativa fica contada.
	return nil
}

// Bloqueia diz se a execucao deve ser recusada. E' a unica pergunta que o resto do codigo faz.
func (p *PareDeEmergencia) Bloqueia() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.estao.Acionado
}

// Estado devolve a copia do retrato (nunca o ponteiro interno).
func (p *PareDeEmergencia) Estado() EstadoDoPare {
	if p == nil {
		return EstadoDoPare{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.estao
}

// Rearmar volta a ligar a execucao. Exige justificacao e autor.
func (p *PareDeEmergencia) Rearmar(porQuem, justificacao string) error {
	if p == nil {
		return ErrPareSemMotivo
	}
	if justificacao == "" {
		return ErrPareSemMotivo
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.estao.Acionado {
		// Rearmar o que nao esta' parado e' um no-op explicito, nao um erro: a intencao e' "estar
		// ligado", e ja' esta'.
		return nil
	}
	p.estao.Acionado = false
	p.estao.RearmadoPor = porQuem
	p.estao.Justificacao = justificacao
	p.estao.RearmadoQuando = p.agora().UTC()
	return nil
}

// AutorizarExecucao e' o ponto que os EXECUTORES chamam antes de fazer alguma coisa.
//
// Devolve `ErrExecucaoParada` com o motivo registado, para que o erro que chega ao utilizador diga
// porque parou em vez de "operacao recusada".
func (p *PareDeEmergencia) AutorizarExecucao(acao string) error {
	if p == nil || !p.Bloqueia() {
		return nil
	}
	estado := p.Estado()
	quando := ""
	if !estado.Quando.IsZero() {
		quando = " desde " + estado.Quando.Format("2006-01-02T15:04:05Z")
	}
	return fmt.Errorf("%w: %s%s (acao: %s)", ErrExecucaoParada, estado.Motivo, quando, acao)
}
