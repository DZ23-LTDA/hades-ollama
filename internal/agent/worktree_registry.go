package agent

// FASE 2 do adendo V3: PERSISTENCIA do estado de ciclo de vida dos worktrees.
//
// A LACUNA QUE ESTE FICHEIRO FECHA
//
// O estado logico (`em_uso`, `suspension`, `arquivado`) vivia **apenas na sessao em memoria**. Isso
// tinha duas consequencias que a E-1232 tornou visiveis:
//
//  1. reiniciar o processo **apagava o estado** -- a suspensao e o arquivo nao sobreviviam;
//  2. o diagnostico so' conseguia afirmar o que o DISCO prova ("existe"), e devolvia `preparado`
//     para tudo -- mesmo para um worktree que o utilizador tinha suspendido.
//
// O `estado_observado` da E-1232 declarava essa limitacao honestamente, mas declarar uma lacuna
// nao a fecha.
//
//	*Um estado que morre com o processo nao e' um estado: e' uma variavel.*
//
// O QUE ESTE REGISTO E'
//
// Um ficheiro JSON versionado, escrito de forma ATOMICA, que mapeia missao -> estado. Nao guarda
// conteudo do worktree, nem credenciais, nem caminhos de utilizador alem do proprio worktree.
//
// AS TRES REGRAS QUE O TORNAM SEGURO
//
//  1. **Escrita atomica.** Escreve num temporario e renomeia. *Um registo meio escrito e' pior do
//     que nenhum registo: mente sobre metade do passado.*
//
//  2. **Ficheiro ausente nao e' erro.** Primeira execucao nao tem registo; devolve um registo
//     vazio. *Confundir "ainda nao ha'" com "falhou" faria o chamador tratar o caso normal como
//     avaria.*
//
//  3. **Ficheiro corrompido e' erro explicito, nunca estado inventado.** Se o JSON nao se le', o
//     chamador tem de saber. *Um registo corrompido lido como vazio apagaria o estado de todos os
//     worktrees com um encolher de ombros.*
//
// O QUE ELE NAO FAZ
//
// Nao substitui o disco: o registo diz o estado LOGICO, o git diz o que existe. O diagnostico
// combina os dois, e quando discordam di-lo.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// NomeDoRegistoDeWorktrees e' o ficheiro, dentro do data root, onde o estado persiste.
const NomeDoRegistoDeWorktrees = "worktree-state.json"

// esquemaDoRegistoDeWorktrees e' a versao do formato. Um esquema desconhecido e' recusado.
const esquemaDoRegistoDeWorktrees = 1

// maxRegistoDeWorktreesBytes limita o ficheiro lido.
//
//	*Um registo que pode crescer sem limite e' um vector de esgotamento de memoria disfarcado de
//	ficheiro de estado.*
const maxRegistoDeWorktreesBytes = 1 << 20

// EntradaDoRegisto e' o estado persistido de UM worktree.
type EntradaDoRegisto struct {
	MissionID string           `json:"mission_id"`
	Estado    EstadoDoWorktree `json:"estado"`
	Caminho   string           `json:"caminho,omitempty"`
	Branch    string           `json:"branch,omitempty"`
	// AtualizadoEm permite diagnosticar estado antigo sem o confundir com atual.
	AtualizadoEm time.Time `json:"atualizado_em"`
}

// RegistoDeWorktrees e' o estado logico persistido de todos os worktrees conhecidos.
type RegistoDeWorktrees struct {
	Esquema  int                         `json:"esquema"`
	Entradas map[string]EntradaDoRegisto `json:"entradas"`

	// caminho e' onde este registo vive. Nao e' serializado.
	caminho string
}

// CaminhoDoRegisto devolve o caminho canonico do registo dentro de um data root.
func CaminhoDoRegisto(dataRoot string) string {
	return filepath.Join(dataRoot, NomeDoRegistoDeWorktrees)
}

// NovoRegistoDeWorktrees cria um registo vazio ligado a um caminho.
func NovoRegistoDeWorktrees(caminho string) *RegistoDeWorktrees {
	return &RegistoDeWorktrees{
		Esquema:  esquemaDoRegistoDeWorktrees,
		Entradas: map[string]EntradaDoRegisto{},
		caminho:  caminho,
	}
}

// ErrRegistoCorrompido e' devolvido quando o ficheiro existe mas nao se le'.
var ErrRegistoCorrompido = errors.New("registo de worktrees corrompido")

// CarregarRegistoDeWorktrees le' o registo do disco.
//
// Ficheiro ausente devolve um registo vazio e `nil` -- primeira execucao e' o caso normal.
// Ficheiro ilegivel devolve `ErrRegistoCorrompido` e **nunca** um registo vazio.
func CarregarRegistoDeWorktrees(caminho string) (*RegistoDeWorktrees, error) {
	if caminho == "" {
		return nil, fmt.Errorf("%w: caminho vazio", ErrRegistoCorrompido)
	}
	info, err := os.Stat(caminho)
	if err != nil {
		if os.IsNotExist(err) {
			//	*Confundir "ainda nao ha'" com "falhou" faria o chamador tratar o caso normal
			//	como avaria.*
			return NovoRegistoDeWorktrees(caminho), nil
		}
		return nil, fmt.Errorf("inspeccionar registo: %w", err)
	}
	if info.Size() > maxRegistoDeWorktreesBytes {
		return nil, fmt.Errorf("%w: ficheiro com %d bytes excede o limite", ErrRegistoCorrompido, info.Size())
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("ler registo: %w", err)
	}

	var lido RegistoDeWorktrees
	// `DisallowUnknownFields` de proposito: um campo que este binario nao conhece significa que
	// outro escreveu o ficheiro; ler na mesma esconderia uma divergencia de esquema.
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&lido); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegistoCorrompido, err)
	}
	if lido.Esquema != esquemaDoRegistoDeWorktrees {
		return nil, fmt.Errorf("%w: esquema %d nao suportado (esperado %d)",
			ErrRegistoCorrompido, lido.Esquema, esquemaDoRegistroWorktreesEsperado())
	}
	if lido.Entradas == nil {
		lido.Entradas = map[string]EntradaDoRegisto{}
	}
	// Validar cada entrada: um estado fora da lista fechada nao entra no registo em memoria.
	for id, e := range lido.Entradas {
		if id == "" {
			return nil, fmt.Errorf("%w: entrada sem mission_id", ErrRegistoCorrompido)
		}
		if !e.Estado.Valido() {
			return nil, fmt.Errorf("%w: missao %q tem estado desconhecido %q",
				ErrRegistoCorrompido, id, e.Estado)
		}
	}
	lido.caminho = caminho
	return &lido, nil
}

// Guardar escreve o registo de forma ATOMICA.
//
// Escreve num temporario no MESMO diretorio e renomeia. Um rename no mesmo volume e' atomico nos
// sistemas que o HADES suporta.
//
//	*Um registo meio escrito e' pior do que nenhum registo: mente sobre metade do passado.*
func (r *RegistoDeWorktrees) Guardar() error {
	if r == nil {
		return fmt.Errorf("%w: registo nulo", ErrRegistoCorrompido)
	}
	if r.caminho == "" {
		return fmt.Errorf("%w: registo sem caminho", ErrRegistoCorrompido)
	}
	r.Esquema = esquemaDoRegistoDeWorktrees
	if r.Entradas == nil {
		r.Entradas = map[string]EntradaDoRegisto{}
	}
	bruto, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar registo: %w", err)
	}
	bruto = append(bruto, '\n')

	dir := filepath.Dir(r.caminho)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("criar diretorio do registo: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".worktree-state-*.tmp")
	if err != nil {
		return fmt.Errorf("criar temporario do registo: %w", err)
	}
	tmpNome := tmp.Name()
	// Se algo falhar, o temporario nao fica para tras.
	defer func() { _ = os.Remove(tmpNome) }()

	if _, err := tmp.Write(bruto); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("escrever temporario do registo: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sincronizar temporario do registo: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechar temporario do registo: %w", err)
	}
	if err := os.Rename(tmpNome, r.caminho); err != nil {
		return fmt.Errorf("publicar registo: %w", err)
	}
	return nil
}

// Registar grava o estado de uma missao.
func (r *RegistoDeWorktrees) Registar(missionID string, estado EstadoDoWorktree, caminho, branch string) error {
	if r == nil {
		return fmt.Errorf("%w: registo nulo", ErrRegistoCorrompido)
	}
	if missionID == "" {
		return errors.New("mission_id e' obrigatorio para registar estado")
	}
	if !estado.Valido() {
		//	*Um registo nao pode guardar um estado que o proprio sistema recusa.*
		return fmt.Errorf("%w: estado desconhecido %q", ErrTransicaoInvalida, estado)
	}
	if r.Entradas == nil {
		r.Entradas = map[string]EntradaDoRegisto{}
	}
	r.Entradas[missionID] = EntradaDoRegisto{
		MissionID:    missionID,
		Estado:       estado,
		Caminho:      caminho,
		Branch:       branch,
		AtualizadoEm: time.Now().UTC(),
	}
	return nil
}

// EstadoDe devolve o estado persistido de uma missao, se existir.
//
// O segundo valor distingue "nao esta' no registo" de "esta' como `criado`": *nao saber e' um
// facto diferente de saber que nao comecou.*
func (r *RegistoDeWorktrees) EstadoDe(missionID string) (EstadoDoWorktree, bool) {
	if r == nil || r.Entradas == nil {
		return "", false
	}
	e, ok := r.Entradas[missionID]
	if !ok {
		return "", false
	}
	return e.Estado, true
}

// Remover tira uma missao do registo.
func (r *RegistoDeWorktrees) Remover(missionID string) {
	if r == nil || r.Entradas == nil {
		return
	}
	delete(r.Entradas, missionID)
}

// Missoes devolve as missoes registadas, ordenadas -- para a resposta ser comparavel entre
// chamadas.
func (r *RegistoDeWorktrees) Missoes() []string {
	if r == nil || r.Entradas == nil {
		return nil
	}
	ids := make([]string, 0, len(r.Entradas))
	for id := range r.Entradas {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Contagem devolve quantas missoes estao registadas.
func (r *RegistoDeWorktrees) Contagem() int {
	if r == nil {
		return 0
	}
	return len(r.Entradas)
}

// esquemaDoRegistroWorktreesEsperado isola a constante para a mensagem de erro.
func esquemaDoRegistroWorktreesEsperado() int { return esquemaDoRegistoDeWorktrees }
