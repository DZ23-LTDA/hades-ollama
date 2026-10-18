package agent

// FASE 2 do adendo V3: as OPERACOES do ciclo de vida, ligadas a maquina de estados.
//
// A LACUNA QUE ESTE FICHEIRO FECHA
//
// A maquina de estados (E-1227) existia, mas o codigo de producao continuava a escrever
// `Status` a mao:
//
//   - `CreateGitWorktree` devolvia `Status: "active"` -- um valor que NEM ESTA na lista fechada;
//   - `removerWorktree` escrevia `session.Status = "removed"` directamente.
//
// A maquina que ninguem usa e' documentacao, nao e' controlo. Este ficheiro liga-a e acrescenta
// as operacoes que faltavam: suspender, retomar e arquivar.
//
// MIGRACAO COMPATIVEL
//
// Sessoes antigas podem ter `Status = "active"` (o valor historico) ou `"merged"`/`"rejected"`.
// `NormalizarEstado` le' os tres e traduz para a lista fechada, sem reescrever o passado:
//
//	*um estado antigo que nao se sabe ler e' um worktree que se trata como se nao existisse.*

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// estadosLegados traduz os valores historicos de `Status` para a lista fechada.
//
// `"active"` era o valor de todas as sessoes criadas antes da E-1227; sem esta traducao, a
// primeira operacao recusaria uma sessao perfeitamente valida.
var estadosLegados = map[string]EstadoDoWorktree{
	// "suspension" era o valor GRAVADO em registos anteriores. Fica como legado traduzido:
	//
	//	*Um valor antigo que nao se sabe ler e' um worktree que se trata como se nao existisse.*
	"suspension": WorktreeSuspenso,
	"active":     WorktreeEmUso,
	"merged":     WorktreeIntegrado,
	"rejected":   WorktreeArquivado,
	"removed":    WorktreeRemovido,
}

// NormalizarEstado traduz um estado legado para a lista fechada.
//
// Devolve o estado e se houve traducao. Um estado desconhecido devolve `WorktreeFalhado`:
// *nao saber em que estado esta' um worktree e' uma falha que precisa de diagnostico, nao um
// estado normal.*
func NormalizarEstado(session *GitWorktreeSession) (EstadoDoWorktree, bool) {
	if session == nil {
		return WorktreeFalhado, false
	}
	bruto := EstadoDoWorktree(strings.TrimSpace(session.Status))
	if bruto == "" {
		// Normalizar inclui o caso vazio: *uma funcao chamada Normalizar que nao normaliza o
		// caso vazio deixa o chamador com dois caminhos para a mesma pergunta.*
		session.Status = string(WorktreeCriado)
		return WorktreeCriado, false
	}
	if bruto.Valido() {
		return bruto, false
	}
	if traduzido, ok := estadosLegados[string(bruto)]; ok {
		session.Status = string(traduzido)
		return traduzido, true
	}
	session.Status = string(WorktreeFalhado)
	return WorktreeFalhado, false
}

// ---------------------------------------------------------------------------------------------
// OPERACOES
// ---------------------------------------------------------------------------------------------

// SuspenderWorktree para o trabalho sem apagar nada.
//
// O diretorio e o branch ficam; apenas o estado muda. Retomar e' a operacao inversa.
//
// *Suspender e' o que se faz quando ainda nao se sabe se o trabalho presta -- apagar seria
// decidir antes de saber.*
func SuspenderWorktree(session *GitWorktreeSession) error {
	if session == nil {
		return fmt.Errorf("%w: sessao nula", ErrTransicaoInvalida)
	}
	// Normalizar traduz o estado legado no proprio Status antes de validar a transicao.
	NormalizarEstado(session)
	_, _, err := Transitar(session, WorktreeSuspenso)
	if err != nil {
		return fmt.Errorf("suspender worktree %q: %w", session.MissionID, err)
	}
	return nil
}

// RetomarWorktree volta a por o worktree em uso.
func RetomarWorktree(session *GitWorktreeSession) error {
	if session == nil {
		return fmt.Errorf("%w: sessao nula", ErrTransicaoInvalida)
	}
	NormalizarEstado(session)
	_, _, err := Transitar(session, WorktreeEmUso)
	if err != nil {
		return fmt.Errorf("retomar worktree %q: %w", session.MissionID, err)
	}
	return nil
}

// ArquivarWorktree tira o worktree do caminho activo sem o remover.
//
// *Arquivar e' o oposto de apagar: o trabalho continua la' para quem o queira ler.*
func ArquivarWorktree(session *GitWorktreeSession) error {
	if session == nil {
		return fmt.Errorf("%w: sessao nula", ErrTransicaoInvalida)
	}
	NormalizarEstado(session)
	// Um worktree em uso tem de passar por suspension -- nao se arquiva debaixo de um agente.
	if EstadoDoWorktree(session.Status) == WorktreeEmUso {
		if err := SuspenderWorktree(session); err != nil {
			return err
		}
	}
	_, _, err := Transitar(session, WorktreeArquivado)
	if err != nil {
		return fmt.Errorf("arquivar worktree %q: %w", session.MissionID, err)
	}
	return nil
}

// MarcarIntegrado regista que os commits entraram no branch alvo.
//
// Nao faz o merge: *marcar como integrado e' registar um facto que ja' aconteceu, nao e' o facto.*
func MarcarIntegrado(session *GitWorktreeSession) error {
	if session == nil {
		return fmt.Errorf("%w: sessao nula", ErrTransicaoInvalida)
	}
	NormalizarEstado(session)
	_, _, err := Transitar(session, WorktreeIntegrado)
	if err != nil {
		return fmt.Errorf("marcar worktree %q como integrado: %w", session.MissionID, err)
	}
	return nil
}

// DiagnosticoDoCicloDeVida descreve onde o worktree esta' e o que pode acontecer a seguir.
//
// Usa `DiagnosticarWorktree` para o estado do disco e o `NormalizarEstado` para o estado logico,
// e diz explicitamente quando os dois discordam -- que e' exactamente o caso que produz
// "estado fantasma".
type DiagnosticoDoCicloDeVida struct {
	MissionID string                `json:"mission_id"`
	Estado    EstadoDoWorktree      `json:"estado"`
	Traduzido bool                  `json:"estado_traduzido"`
	Disco     DiagnosticoDoWorktree `json:"disco"`
	Proximos  []EstadoDoWorktree    `json:"proximos"`
	// Inconsistente explica quando o estado logico e o disco discordam.
	Inconsistente string `json:"inconsistente,omitempty"`
	// EstadoDoRegisto diz se o estado veio do registo PERSISTIDO (e nao apenas do disco).
	//
	// *Um estado que morre com o processo nao e' um estado: e' uma variavel.*
	EstadoDoRegisto bool `json:"estado_do_registo"`
	// EstadoObservado explica a origem do estado, para o chamador nao ter de adivinhar.
	EstadoObservado string `json:"estado_observado,omitempty"`
	// PodeRemover responde a' pergunta pratica, ja' com o estado final considerado.
	PodeRemover bool `json:"pode_remover"`
	// MotivoRemover explica a recusa, quando ha' uma.
	MotivoRemover string `json:"motivo_remover,omitempty"`
}

// DiagnosticarCicloDeVida le' o estado logico e o disco e compara-os.
func DiagnosticarCicloDeVida(ctx context.Context, session *GitWorktreeSession) DiagnosticoDoCicloDeVida {
	var d DiagnosticoDoCicloDeVida
	if session == nil {
		return d
	}
	d.MissionID = session.MissionID
	estado, traduziu := NormalizarEstado(session)
	d.Estado = estado
	d.Traduzido = traduziu
	d.Disco = DiagnosticarWorktree(ctx, session)
	d.Proximos = TransicoesDe(estado)

	// Comparar os dois: e' aqui que o "estado fantasma" aparece.
	existeNoDisco := d.Disco.CaminhoResolvido != ""
	switch {
	case estado == WorktreeRemovido && d.Disco.AlvoSeguro:
		d.Inconsistente = "estado diz removido mas o caminho ainda e' um alvo valido no disco"
	case estado != WorktreeRemovido && estado != WorktreeOrfao && d.Disco.CaminhoResolvido == "":
		d.Inconsistente = "estado nao e' removido mas nao ha' caminho para inspeccionar"
	case d.Disco.TemTrabalhoNaoIntegrado && estado == WorktreeIntegrado:
		d.Inconsistente = "estado diz integrado mas o worktree tem trabalho por integrar"
	}
	_ = existeNoDisco
	return d
}

// PodeRemoverAgora responde a' pergunta pratica: da' para remover sem perder trabalho?
//
// Exige estado logico que permita a transicao E disco sem trabalho por integrar.
//
//	*Duas condicoes, porque uma so' deixa passar metade dos casos maus.*
func PodeRemoverAgora(ctx context.Context, session *GitWorktreeSession) (bool, string) {
	if session == nil {
		return false, "sessao nula"
	}
	estado, _ := NormalizarEstado(session)
	permitido := false
	for _, destino := range transicoesDoWorktree[estado] {
		if destino == WorktreeRemovido {
			permitido = true
			break
		}
	}
	if !permitido {
		return false, fmt.Sprintf("o estado %q nao permite remover (proximos: %v)", estado, transicoesDoWorktree[estado])
	}
	diag := DiagnosticarWorktree(ctx, session)
	if !diag.AlvoSeguro {
		return false, "alvo nao resolvido: " + diag.MotivoDoAlvo
	}
	if diag.TemTrabalhoNaoIntegrado {
		return false, diag.Detalhe
	}
	return true, ""
}

// ErrEstadoInconsistente e' devolvido quando o estado logico e o disco discordam.
var ErrEstadoInconsistente = errors.New("estado do worktree inconsistente com o disco")

// DiagnosticarCicloDeVidaComRegisto combina o estado PERSISTIDO com o que o disco prova.
//
// A ordem de autoridade:
//
//  1. se ha' registo para a missao, o estado do REGISTO manda no logico (`em_uso`, `suspension`,
//     `arquivado`) -- foi o processo anterior que o decidiu, e essa decisao sobreviveu;
//  2. o DISCO manda no que existe: commits nao integrados, ficheiros por commitar, alvo seguro;
//  3. quando os dois discordam, `Inconsistente` di-lo.
//
// Com registo `nil`, o comportamento e' o da E-1232 (so' o disco fala) e `EstadoDoRegisto` fica
// falso -- *um diagnostico que nao recebeu registo nao pode dizer que o consultou.*
//
//	*Um registo que ninguem le' nao e' persistencia: e' um ficheiro.*
func DiagnosticarCicloDeVidaComRegisto(ctx context.Context, session *GitWorktreeSession, registo *RegistoDeWorktrees) DiagnosticoDoCicloDeVida {
	d := DiagnosticarCicloDeVida(ctx, session)
	if session == nil {
		return d
	}

	if registo != nil {
		if estado, existe := registo.EstadoDe(session.MissionID); existe {
			anterior := d.Estado
			d.Estado = estado
			d.EstadoDoRegisto = true
			d.Proximos = TransicoesDe(estado)
			d.EstadoObservado = "estado do registo persistido"
			if anterior != estado {
				// O disco derivava outra coisa; o registo prevalece e a diferenca e' registada.
				// *Duas fontes para a mesma pergunta sem dizer qual mandou e' como nao ter nenhuma.*
				d.Inconsistente = fmt.Sprintf(
					"o registo diz %q e o disco sugeria %q; o registo prevalece", estado, anterior)
			}
		} else {
			d.EstadoObservado = "sem entrada no registo; estado derivado do disco"
		}
	} else {
		d.EstadoObservado = "sem registo fornecido; estado derivado do disco"
	}

	// Reavaliar a remocao com o estado final: um worktree `em_uso` nao pode ser removido mesmo
	// que o disco esteja limpo.
	permitido := false
	for _, destino := range transicoesDoWorktree[d.Estado] {
		if destino == WorktreeRemovido {
			permitido = true
			break
		}
	}
	switch {
	case !permitido:
		d.PodeRemover = false
		d.MotivoRemover = fmt.Sprintf("o estado %q nao permite remover", d.Estado)
	case !d.Disco.AlvoSeguro:
		d.PodeRemover = false
		d.MotivoRemover = "alvo nao resolvido: " + d.Disco.MotivoDoAlvo
	case d.Disco.TemTrabalhoNaoIntegrado:
		d.PodeRemover = false
		d.MotivoRemover = d.Disco.Detalhe
	default:
		d.PodeRemover = true
		d.MotivoRemover = ""
	}
	return d
}
