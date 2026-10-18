package agent

// FASE 2 do adendo V3: a MAQUINA DE ESTADOS do worktree e a DESCOBERTA de worktrees externos.
//
// A LACUNA QUE ESTE FICHEIRO FECHA
//
// O `GitWorktreeSession.Status` era uma `string` com quatro valores escritos em comentario:
// "active", "merged", "rejected", "removed". Isso permitia, e o codigo fazia-o:
//
//   - escrever qualquer string em `Status` sem ninguem validar;
//   - marcar "removed" sem remover (o defeito corrigido em E-1226);
//   - saltar de "active" para "removed" sem passar por integracao ou arquivo.
//
// O adendo exige, textualmente:
//   - "criacao, preparacao, diagnostico, suspensao, retomada, arquivamento e remocao possuem
//     estados explicitos";
//   - "worktree criado externamente pode ser descoberto ou anexado sem reset";
//   - "falha parcial nao pode deixar branch orfa ou estado fantasma sem diagnostico".
//
// Duas pecas, entao:
//
//  1. Uma lista FECHADA de estados com transicoes declaradas. *Um estado que ninguem pode nomear
//     invalido nao e' um estado: e' um campo de texto.*
//  2. Um inventor que le o `git worktree list` do repositorio e diz o que esta' la' -- incluindo
//     worktrees que o HADES NAO criou. *Um worktree externo que nao se ve' e' um worktree que se
//     apaga por engano.*

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// EstadoDoWorktree e' o ciclo de vida de um worktree. Lista fechada.
type EstadoDoWorktree string

const (
	// WorktreeCriado: o diretorio e o branch existem, mas o ambiente ainda nao foi preparado.
	WorktreeCriado EstadoDoWorktree = "criado"
	// WorktreePreparado: dependencias instaladas, ambiente pronto para receber trabalho.
	WorktreePreparado EstadoDoWorktree = "preparado"
	// WorktreeEmUso: um agente esta' a escrever.
	WorktreeEmUso EstadoDoWorktree = "em_uso"
	// WorktreeSuspenso: trabalho parado, preservado no disco, sem agente activo.
	WorktreeSuspenso EstadoDoWorktree = "suspension"
	// WorktreeIntegrado: os commits entraram no branch alvo.
	WorktreeIntegrado EstadoDoWorktree = "integrado"
	// WorktreeArquivado: preservado para consulta, fora do caminho activo. NAO e' removido.
	WorktreeArquivado EstadoDoWorktree = "arquivado"
	// WorktreeRemovido: o diretorio foi apagado; o branch pode continuar.
	WorktreeRemovido EstadoDoWorktree = "removido"
	// WorktreeFalhado: algo correu mal e o estado precisa de diagnostico humano.
	WorktreeFalhado EstadoDoWorktree = "falhado"
	// WorktreeOrfao: o diretorio desapareceu sem passar pela remocao.
	WorktreeOrfao EstadoDoWorktree = "orfao"
)

// EstadosDoWorktree devolve a lista fechada, para a UI e para os testes.
func EstadosDoWorktree() []EstadoDoWorktree {
	return []EstadoDoWorktree{
		WorktreeCriado, WorktreePreparado, WorktreeEmUso, WorktreeSuspenso,
		WorktreeIntegrado, WorktreeArquivado, WorktreeRemovido, WorktreeFalhado, WorktreeOrfao,
	}
}

// Valido diz se o estado pertence a lista fechada.
func (e EstadoDoWorktree) Valido() bool {
	for _, c := range EstadosDoWorktree() {
		if c == e {
			return true
		}
	}
	return false
}

// transicoesDoWorktree declara o que pode seguir o que.
//
// A regra que decide cada linha: **nao se salta para um estado final sem passar pelo trabalho**.
// `criado -> removido` nao existe porque remover o que nunca correu esconde um erro de orquestracao.
// `em_uso -> removido` nao existe porque apagar com agente a escrever e' a receita da perda que a
// E-1226 corrigiu.
//
// *Uma transicao que nao existe obriga a quem a quer a perguntar por que a quer.*
var transicoesDoWorktree = map[EstadoDoWorktree][]EstadoDoWorktree{
	WorktreeCriado:    {WorktreePreparado, WorktreeFalhado, WorktreeOrfao},
	WorktreePreparado: {WorktreeEmUso, WorktreeSuspenso, WorktreeFalhado, WorktreeOrfao},
	WorktreeEmUso:     {WorktreeSuspenso, WorktreeIntegrado, WorktreeFalhado, WorktreeOrfao},
	WorktreeSuspenso:  {WorktreeEmUso, WorktreeArquivado, WorktreeIntegrado, WorktreeFalhado, WorktreeOrfao},
	WorktreeIntegrado: {WorktreeArquivado, WorktreeRemovido},
	WorktreeArquivado: {WorktreeRemovido, WorktreeEmUso},
	WorktreeFalhado:   {WorktreeSuspenso, WorktreeArquivado, WorktreeRemovido},
	WorktreeOrfao:     {WorktreeRemovido},
	WorktreeRemovido:  {}, // terminal
}

// TransicoesDe devolve o que pode seguir um estado, para diagnostico e para a UI.
func TransicoesDe(de EstadoDoWorktree) []EstadoDoWorktree {
	destinos := transicoesDoWorktree[de]
	saida := make([]EstadoDoWorktree, len(destinos))
	copy(saida, destinos)
	return saida
}

// ErrTransicaoInvalida e' devolvido quando se tenta saltar para fora do ciclo de vida.
var ErrTransicaoInvalida = fmt.Errorf("transicao de estado de worktree invalida")

// Transitar valida e aplica uma transicao de estado.
//
// Recusa estado de origem desconhecido (senao qualquer string entraria no ciclo) e recusa destino
// que nao esteja declarado.
//
// Devolve o estado anterior e o novo em caso de sucesso, para o chamador registar a mudanca.
func Transitar(session *GitWorktreeSession, para EstadoDoWorktree) (de EstadoDoWorktree, paraFinal EstadoDoWorktree, err error) {
	if session == nil {
		return "", "", fmt.Errorf("%w: sessao nula", ErrTransicaoInvalida)
	}
	atual := EstadoDoWorktree(session.Status)
	// Um estado vazio e' o `criado` implicito de uma sessao recem-construida em memoria.
	// *Nao tratar o vazio como erro obrigaria todos os chamadores a inicializar o campo.*
	if atual == "" {
		atual = WorktreeCriado
		session.Status = string(WorktreeCriado)
	}
	if !para.Valido() {
		return atual, "", fmt.Errorf("%w: estado de destino desconhecido %q", ErrTransicaoInvalida, para)
	}
	if !atual.Valido() {
		return atual, "", fmt.Errorf("%w: estado de origem desconhecido %q", ErrTransicaoInvalida, atual)
	}
	if atual == para {
		// Idempotente: repetir a mesma transicao nao e' erro, mas tambem nao e' mudanca.
		return atual, para, nil
	}
	for _, destino := range transicoesDoWorktree[atual] {
		if destino == para {
			session.Status = string(para)
			return atual, para, nil
		}
	}
	return atual, "", fmt.Errorf("%w: %q -> %q; a partir de %q so' se pode ir para %v",
		ErrTransicaoInvalida, atual, para, atual, transicoesDoWorktree[atual])
}

// ---------------------------------------------------------------------------------------------
// DESCOBERTA DE WORKTREES
// ---------------------------------------------------------------------------------------------

// WorktreeDescoberto e' um worktree que existe no repositorio, tenha o HADES criado ou nao.
type WorktreeDescoberto struct {
	// CaminhoResolvido e' o caminho absoluto e limpo.
	CaminhoResolvido string `json:"caminho_resolvido"`
	// Branch e' o branch de referencia, quando destacado.
	Branch string `json:"branch,omitempty"`
	// Detached diz que o worktree esta' num commit sem branch.
	Detached bool `json:"detached"`
	// Commit e' o HEAD do worktree, quando legivel.
	Commit string `json:"commit,omitempty"`
	// CriadoPeloHades diz se o caminho segue a convencao `.agent-worktrees`.
	CriadoPeloHades bool `json:"criado_pelo_hades"`
	// Principal marca o worktree principal do repositorio (nunca deve ser removido por engano).
	Principal bool `json:"principal"`
	// Bloqueado indica que o git o reporta como bloqueado/prunable.
	Bloqueado bool `json:"bloqueado"`
}

// InventariarWorktrees le `git worktree list --porcelain` e descreve o que existe.
//
// Nao altera nada. Devolve lista ordenada por caminho para ser comparavel entre chamadas.
//
// *Um worktree externo que nao se ve' e' um worktree que se apaga por engano -- e o erro nao e'
// de quem apagou, e' de quem nao o mostrou.*
func InventariarWorktrees(ctx context.Context, repoRoot string) ([]WorktreeDescoberto, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("repo root e' obrigatorio para inventariar worktrees")
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolver repo root: %w", err)
	}
	// `git worktree list` SOBE a arvore de diretorios: chamado num subdiretorio de um
	// repositorio, devolve os worktrees do repositorio PAI. Sem esta verificacao, um "inventor"
	// de um diretorio que nao e' repositorio devolveria os worktrees de outro projeto -- e o
	// chamador apagaria o que nao e' dele.
	//
	// *Um inventor que devolve worktrees de OUTRO repositorio e' pior do que um erro: parece
	// resposta.*
	toplevel, errTop := runRawGit(ctx, abs, "rev-parse", "--show-toplevel")
	if errTop != nil {
		return nil, fmt.Errorf("%q nao e' a raiz de um repositorio git: %w", abs, errTop)
	}
	if filepath.Clean(strings.TrimSpace(toplevel)) != filepath.Clean(abs) {
		return nil, fmt.Errorf("%q nao e' a raiz de um repositorio git (a raiz e' %q)",
			abs, filepath.Clean(strings.TrimSpace(toplevel)))
	}

	saida, err := runRawGit(ctx, abs, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("listar worktrees: %w", err)
	}

	var (
		lista    []WorktreeDescoberto
		atual    WorktreeDescoberto
		temAtual bool
	)
	// O formato --porcelain emite blocos separados por linha vazia.
	fechar := func() {
		if !temAtual {
			return
		}
		atual.CaminhoResolvido = filepath.Clean(atual.CaminhoResolvido)
		atual.CriadoPeloHades = caminhoSegueConvencao(atual.CaminhoResolvido)
		lista = append(lista, atual)
		atual = WorktreeDescoberto{}
		temAtual = false
	}
	for _, linha := range strings.Split(saida, "\n") {
		linha = strings.TrimRight(linha, "\r")
		if strings.TrimSpace(linha) == "" {
			fechar()
			continue
		}
		campo, valor, temValor := strings.Cut(linha, " ")
		switch campo {
		case "worktree":
			fechar()
			atual.CaminhoResolvido = valor
			temAtual = true
		case "HEAD":
			if temValor {
				atual.Commit = valor
			}
		case "branch":
			if temValor {
				atual.Branch = strings.TrimPrefix(valor, "refs/heads/")
			}
		case "detached":
			atual.Detached = true
		case "prunable", "locked":
			atual.Bloqueado = true
		}
	}
	fechar()

	// O primeiro da lista e' o worktree principal do repositorio.
	if len(lista) > 0 {
		lista[0].Principal = true
	}
	sort.Slice(lista, func(a, b int) bool {
		return lista[a].CaminhoResolvido < lista[b].CaminhoResolvido
	})
	return lista, nil
}

// caminhoSegueConvencao diz se o caminho tem o segmento da convencao do HADES.
func caminhoSegueConvencao(caminho string) bool {
	for _, parte := range strings.Split(filepath.ToSlash(caminho), "/") {
		if parte == SegmentoDeWorktrees {
			return true
		}
	}
	return false
}

// ResumoDoInventario e' uma linha legivel para o Mission Control.
func ResumoDoInventario(lista []WorktreeDescoberto) string {
	externos, hades := 0, 0
	for _, w := range lista {
		if w.CriadoPeloHades {
			hades++
		} else {
			externos++
		}
	}
	return fmt.Sprintf("%d worktree(s): %d do HADES, %d externo(s)", len(lista), hades, externos)
}
