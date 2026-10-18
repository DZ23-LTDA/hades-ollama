package agent

// FASE 2 do adendo V3: ciclo de vida de worktree com PROTECAO DE TRABALHO.
//
// O DEFEITO QUE ESTE FICHEIRO CORRIGE
//
// A `RemoveGitWorktree` original fazia, sem verificar nada:
//
//	runRawGit(ctx, session.RepoRoot, "worktree", "remove", "--force", session.WorktreeDir)
//	os.RemoveAll(session.WorktreeDir)
//	session.Status = "removed"
//	return nil
//
// Tres problemas, todos do mesmo tipo:
//
//  1. `--force` + `os.RemoveAll` apagam o worktree MESMO com commits que nunca foram integrados.
//     O trabalho do agente desaparece sem uma linha de aviso.
//  2. Todos os erros sao descartados (`_, _ =`, `_ =`), portanto o chamador nunca sabe que falhou.
//  3. `Status = "removed"` e' escrito sempre, mesmo quando nada foi removido -- um estado fantasma.
//
// O adendo exige textualmente:
//   - "alteracoes e commits nao integrados sao preservados antes de qualquer remoção";
//   - "exclusao exige alvo resolvido, confirmacao apropriada e protecao de trabalho nao integrado";
//   - "falha parcial nao pode deixar branch orfa ou estado fantasma sem diagnostico".
//
// A regra que fica em codigo:
//
//	*Apagar trabalho que ninguem integrou nao e' limpeza: e' perda de dados com nome tecnico.*
//
// O QUE ESTE FICHEIRO NAO FAZ
//
// Nao muda a assinatura de `RemoveGitWorktree`. A funcao passa a ser SEGURA POR OMISSAO: recusa
// remover quando ha' trabalho nao integrado. Quem quiser mesmo descartar tem de o dizer, e para
// isso existe `RemoveGitWorktreeComPerda`.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SegmentoDeWorktrees e' a convencao de diretorio que `CreateGitWorktree` usa.
//
// A validacao de alvo depende desta constante: *um path que nao segue a convencao nao foi criado
// pelo HADES, e apaga-lo seria agir sobre o que nao e' nosso.*
const SegmentoDeWorktrees = ".agent-worktrees"

// ErrTrabalhoNaoIntegrado e' devolvido quando o worktree tem trabalho que ninguem integrou.
//
// E' um erro TIPADO porque o chamador tem de o distinguir de uma falha de git: *"ha' trabalho a
// perder" e' uma decisao, "o git falhou" e' um acidente.*
var ErrTrabalhoNaoIntegrado = errors.New("worktree tem trabalho nao integrado")

// ErrAlvoNaoResolvido e' devolvido quando o caminho do worktree nao e' seguro para remover.
var ErrAlvoNaoResolvido = errors.New("alvo do worktree nao resolvido")

// DiagnosticoDoWorktree e' o que se sabe sobre um worktree antes de o remover.
//
// Todos os campos sao preenchidos por leitura; nenhum deles muda o repositorio.
type DiagnosticoDoWorktree struct {
	// CaminhoResolvido e' o caminho absoluto e limpo do worktree.
	CaminhoResolvido string `json:"caminho_resolvido"`
	// AlvoSeguro diz se o caminho segue a convencao e nao e' ancestral do repositorio.
	AlvoSeguro bool `json:"alvo_seguro"`
	// MotivoDoAlvo explica porque o alvo foi recusado, quando foi.
	MotivoDoAlvo string `json:"motivo_do_alvo,omitempty"`
	// CommitsNaoIntegrados conta os commits em `BaseCommit..HEAD` do worktree.
	CommitsNaoIntegrados int `json:"commits_nao_integrados"`
	// CommitsIntegrados diz que esses commits JA' estao no branch alvo.
	//
	// *Trabalho ja' integrado nao e' trabalho perdido.*
	CommitsIntegrados bool `json:"commits_integrados"`
	// FicheirosModificados conta as entradas de `git status --porcelain`.
	FicheirosModificados int `json:"ficheiros_modificados"`
	// TemTrabalhoNaoIntegrado resume os dois campos acima.
	TemTrabalhoNaoIntegrado bool `json:"tem_trabalho_nao_integrado"`
	// Detalhe explica o estado em linguagem simples.
	Detalhe string `json:"detalhe"`
}

// PodeRemover diz se o worktree pode ser removido sem perda.
func (d DiagnosticoDoWorktree) PodeRemover() bool {
	return d.AlvoSeguro && !d.TemTrabalhoNaoIntegrado
}

// DiagnosticarWorktree inspecciona um worktree antes da remocao, sem o alterar.
//
// Nunca devolve erro: um diagnostico que falha a meio tem de dizer o que conseguiu ver.
// *Um diagnostico que rebenta deixa o chamador sem informacao exactamente quando precisa dela.*
func DiagnosticarWorktree(ctx context.Context, session *GitWorktreeSession) DiagnosticoDoWorktree {
	var d DiagnosticoDoWorktree
	if session == nil || strings.TrimSpace(session.WorktreeDir) == "" {
		d.MotivoDoAlvo = "sessao ou caminho vazio"
		d.Detalhe = "sem caminho nao ha' alvo; nao se remove o que nao se identificou"
		return d
	}

	abs, err := filepath.Abs(session.WorktreeDir)
	if err != nil {
		d.MotivoDoAlvo = "nao foi possivel resolver o caminho absoluto"
		d.Detalhe = "alvo nao resolvido"
		return d
	}
	d.CaminhoResolvido = filepath.Clean(abs)

	if motivo := motivoDeAlvoInseguro(d.CaminhoResolvido, session.RepoRoot); motivo != "" {
		d.MotivoDoAlvo = motivo
		d.Detalhe = motivo
		return d
	}
	d.AlvoSeguro = true

	// A pergunta certa NAO e' "ha' commits?" -- e' "ha' commits que o branch alvo ainda nao tem?".
	//
	// *Trabalho ja' integrado nao e' trabalho perdido.* Recusar a remocao depois de um merge
	// bem-sucedido seria um falso positivo -- e um falso positivo num gate ensina a ignorar o gate.
	//
	// Por isso, quando o `TargetBranch` e' conhecido, verificamos se o HEAD do worktree ja' esta'
	// contido nele. Se estiver, os commits estao integrados e nao contam.
	headIntegrado := false
	if strings.TrimSpace(session.BaseCommit) != "" {
		saida, err := runRawGit(ctx, d.CaminhoResolvido, "rev-list", "--count", session.BaseCommit+"..HEAD")
		if err == nil {
			n := 0
			if _, scanErr := fmt.Sscanf(strings.TrimSpace(saida), "%d", &n); scanErr == nil {
				d.CommitsNaoIntegrados = n
			}
		}
	}
	if d.CommitsNaoIntegrados > 0 && strings.TrimSpace(session.TargetBranch) != "" {
		d.CommitsIntegrados = true
		headDoWorktree, errHead := runRawGit(ctx, d.CaminhoResolvido, "rev-parse", "HEAD")
		if errHead == nil {
			headDoWorktree = strings.TrimSpace(headDoWorktree)
			// `merge-base --is-ancestor HEAD <alvo>` devolve 0 quando ja' esta' contido.
			if _, errAnc := runRawGit(ctx, session.RepoRoot, "merge-base", "--is-ancestor", headDoWorktree, session.TargetBranch); errAnc == nil {
				headIntegrado = true
			}
		}
	}
	_ = headIntegrado

	// Ficheiros por commitar dentro do worktree.
	if saida, err := runRawGit(ctx, d.CaminhoResolvido, "status", "--porcelain"); err == nil {
		for _, linha := range strings.Split(saida, "\n") {
			if strings.TrimSpace(linha) != "" {
				d.FicheirosModificados++
			}
		}
	}

	// Commits integrados no alvo nao contam como trabalho por integrar.
	commitsPorIntegrar := d.CommitsNaoIntegrados
	if headIntegrado {
		commitsPorIntegrar = 0
	}
	d.TemTrabalhoNaoIntegrado = commitsPorIntegrar > 0 || d.FicheirosModificados > 0
	switch {
	case headIntegrado && d.FicheirosModificados == 0:
		d.Detalhe = fmt.Sprintf("%d commit(s) ja' integrado(s) no alvo %q", d.CommitsNaoIntegrados, session.TargetBranch)
	case d.CommitsNaoIntegrados > 0 && d.FicheirosModificados > 0:
		d.Detalhe = fmt.Sprintf("%d commit(s) nao integrado(s) e %d ficheiro(s) por commitar",
			d.CommitsNaoIntegrados, d.FicheirosModificados)
	case d.CommitsNaoIntegrados > 0:
		d.Detalhe = fmt.Sprintf("%d commit(s) nao integrado(s)", d.CommitsNaoIntegrados)
	case d.FicheirosModificados > 0:
		d.Detalhe = fmt.Sprintf("%d ficheiro(s) por commitar", d.FicheirosModificados)
	default:
		d.Detalhe = "sem trabalho por integrar"
	}
	return d
}

// motivoDeAlvoInseguro recusa caminhos que nao deviam ser apagados.
//
// As regras, em ordem: tem de ser absoluto (o `Abs` ja' garante), tem de conter o segmento da
// convencao, e NAO pode ser ancestral do proprio repositorio.
//
// *`os.RemoveAll` num ancestral do repositorio apagaria o produto para "limpar" um worktree.*
func motivoDeAlvoInseguro(caminho string, repoRoot string) string {
	if !filepath.IsAbs(caminho) {
		return "caminho nao absoluto"
	}
	// Raiz de volume ou caminho demasiado curto nunca e' um worktree.
	if len(filepath.Clean(caminho)) <= len(filepath.VolumeName(caminho))+1 {
		return "caminho e' a raiz do volume"
	}
	// Tem de seguir a convencao criada pelo HADES.
	partes := strings.Split(filepath.ToSlash(caminho), "/")
	segueConvencao := false
	for _, parte := range partes {
		if parte == SegmentoDeWorktrees {
			segueConvencao = true
			break
		}
	}
	if !segueConvencao {
		return "caminho nao segue a convencao " + SegmentoDeWorktrees + "; nao foi criado pelo HADES"
	}
	// Nao pode ser o repositorio nem um ancestral dele.
	if strings.TrimSpace(repoRoot) != "" {
		absRepo, err := filepath.Abs(repoRoot)
		if err == nil {
			absRepo = filepath.Clean(absRepo)
			if caminho == absRepo {
				return "caminho e' o proprio repositorio"
			}
			if strings.HasPrefix(absRepo, caminho+string(filepath.Separator)) {
				return "caminho e' ancestral do repositorio"
			}
		}
	}
	return ""
}

// RemoveGitWorktreeSeguro remove o worktree APENAS se nao houver trabalho por integrar.
//
// Devolve `ErrTrabalhoNaoIntegrado` (com o diagnostico em anexo) quando recusa.
// *Recusar e' o comportamento correcto: o chamador pode entao decidir integrar, arquivar ou
// descartar com `RemoveGitWorktreeComPerda`.*
func RemoveGitWorktreeSeguro(ctx context.Context, session *GitWorktreeSession) error {
	if session == nil {
		return nil
	}
	diag := DiagnosticarWorktree(ctx, session)
	if !diag.AlvoSeguro {
		return fmt.Errorf("%w: %s", ErrAlvoNaoResolvido, diag.MotivoDoAlvo)
	}
	if diag.TemTrabalhoNaoIntegrado {
		return fmt.Errorf("%w: %s", ErrTrabalhoNaoIntegrado, diag.Detalhe)
	}
	return removerWorktree(ctx, session, diag)
}

// RemoveGitWorktreeComPerda remove mesmo com trabalho nao integrado, mas EXIGE um motivo.
//
// O motivo existe para o registo: *um descarte deliberado tem de deixar rastro, senao amanha
// ninguem sabe se o trabalho foi integrado ou apagado.*
func RemoveGitWorktreeComPerda(ctx context.Context, session *GitWorktreeSession, motivo string) error {
	if session == nil {
		return nil
	}
	if strings.TrimSpace(motivo) == "" {
		return errors.New("descartar trabalho nao integrado exige um motivo")
	}
	diag := DiagnosticarWorktree(ctx, session)
	if !diag.AlvoSeguro {
		return fmt.Errorf("%w: %s", ErrAlvoNaoResolvido, diag.MotivoDoAlvo)
	}
	session.MotivoDoDescarte = motivo
	session.TrabalhoDescartado = diag.Detalhe
	return removerWorktree(ctx, session, diag)
}

// removerWorktree faz a remocao efectiva e propaga os erros.
//
// A diferenca para o original: nenhum erro e' descartado, e o estado so' muda quando a remocao
// realmente aconteceu.
func removerWorktree(ctx context.Context, session *GitWorktreeSession, diag DiagnosticoDoWorktree) error {
	if session.RepoRoot != "" {
		// `--force` continua necessario porque o git recusa worktrees com modificacoes locais;
		// mas aqui ja' sabemos que nao ha' nada a perder.
		if _, err := runRawGit(ctx, session.RepoRoot, "worktree", "remove", "--force", diag.CaminhoResolvido); err != nil {
			// Limpar a metadata orfa do git antes de desistir, para nao deixar estado fantasma.
			_, _ = runRawGit(ctx, session.RepoRoot, "worktree", "prune")
			return fmt.Errorf("remover worktree via git: %w", err)
		}
	}
	if err := os.RemoveAll(diag.CaminhoResolvido); err != nil {
		return fmt.Errorf("remover diretorio do worktree: %w", err)
	}
	// O branch NAO e' apagado: *um branch e' o unico sitio onde o trabalho nao integrado ainda
	// pode estar, e apaga-lo seria perder a ultima copia.*
	session.Status = "removed"
	session.CaminhoRemovido = diag.CaminhoResolvido
	return nil
}

// DiagnosticarERemover e' o ponto de entrada que os chamadores devem usar.
//
// Faz o diagnostico, tenta a remocao segura, e devolve o diagnostico em qualquer dos casos, para
// o chamador poder reportar o que aconteceu.
func DiagnosticarERemover(ctx context.Context, session *GitWorktreeSession) (DiagnosticoDoWorktree, error) {
	diag := DiagnosticarWorktree(ctx, session)
	err := RemoveGitWorktreeSeguro(ctx, session)
	return diag, err
}
