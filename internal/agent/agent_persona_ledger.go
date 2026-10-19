// Ledger transacional das instalacoes de persona geridas pelo HADES.
//
// O QUE ESTE FICHEIRO ACRESCENTA AO `agent_persona.go`
//
// O `agent_persona.go` sabe calcular o hash e classificar o drift. Falta-lhe a MEMORIA: sem um registo do que
// o HADES escreveu, "drift" nao tem contra o que ser comparado -- so' contra a expectativa do momento, que e'
// exactamente o que muda.
//
//	*Um drift que se mede contra o presente nao e' drift: e' uma diferenca. O drift precisa do que ficou
//	gravado no momento em que se escreveu.*
//
// PORQUE O LEDGER E' TRANSACIONAL, E NAO UM LOG
//
// Uma instalacao escreve um ficheiro E regista-o. Entre as duas coisas ha' uma janela. Se o processo morrer
// dentro dela, o ficheiro existe e o ledger nao sabe -- e o proximo reconcile chama-lhe FOREIGN_UNMANAGED,
// que manda NAO remover. E' o comportamento certo: um orfao desconhecido nao se apaga.
//
//	*O estado que o HADES nao registou nao e' dele. Um ficheiro escrito por engano nao se torna nosso por
//	estar no sitio certo.*
//
// # IDEMPOTENCIA
//
// Um retry do mesmo pedido (rede caiu, agente repetiu) tem de NAO duplicar a entrada. A chave e'
// (persona, target, destino) -- o que identifica a escrita, nao o seu conteudo. Mesma chave e mesmos hashes:
// devolve a entrada existente e `criada=false`. Mesma chave e hashes DIFERENTES: `CONFLICTED` -- duas
// escritas do mesmo sitio com conteudos diferentes nao sao um retry, sao um conflito.
//
//	*Um ledger que aceita duas verdades para a mesma chave nao reconcilia nada: escolhe a ultima.*
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// ErrPersonaLedgerCorrupt cobre registos ilegiveis ou adulterados.
var ErrPersonaLedgerCorrupt = errors.New("agent persona ledger is corrupt")

// PersonaLedger guarda as instalacoes por escopo.
type PersonaLedger struct {
	mu   sync.Mutex
	root string
	// byScope: escopo -> chave -> entrada
	byScope map[string]map[string]PersonaLedgerEntry
}

// NewPersonaLedger abre (ou cria) o ledger. Raiz vazia = so' em memoria, como no PluginRegistry.
func NewPersonaLedger(root string) (*PersonaLedger, error) {
	ledger := &PersonaLedger{root: strings.TrimSpace(root), byScope: map[string]map[string]PersonaLedgerEntry{}}
	if ledger.root != "" {
		if err := os.MkdirAll(filepath.Join(ledger.root, "personas"), 0o755); err != nil {
			return nil, err
		}
	}
	return ledger, nil
}

// PersonaLedgerKey e' a identidade de uma escrita. Nao inclui hashes: o hash e' o CONTEUDO da escrita, nao a
// sua identidade.
func PersonaLedgerKey(entry PersonaLedgerEntry) string {
	return entry.PersonaID + "|" + string(entry.Target) + "|" + entry.Destination
}

func (l *PersonaLedger) scopePath(scope string) string {
	return filepath.Join(l.root, "personas", sanitizeIDComponent(scope)+".json")
}

// loadLocked garante que o escopo esta' carregado do disco. Chamador segura o mutex.
//
// Sem isto, um ledger REABERTO (processo novo) respondia «zero instalacoes» com o ficheiro cheio -- e um
// `Record` novo persistia SO a sua entrada, apagando as antigas.
//
//	*Uma lista vazia e um erro nao sao a mesma coisa -- e uma lista vazia que devia ter conteudo e' pior.*
func (l *PersonaLedger) loadLocked(scope string) error {
	if _, ok := l.byScope[scope]; ok {
		return nil
	}
	l.byScope[scope] = map[string]PersonaLedgerEntry{}
	if l.root == "" {
		return nil
	}
	raw, err := os.ReadFile(l.scopePath(scope))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var lista []PersonaLedgerEntry
	if err := json.Unmarshal(raw, &lista); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrPersonaLedgerCorrupt, err)
	}
	for _, e := range lista {
		l.byScope[scope][PersonaLedgerKey(e)] = e
	}
	return nil
}

// Record grava uma instalacao. Devolve a entrada efectiva e se foi CRIADA agora.
//
// Idempotente pela chave: repetir o mesmo pedido nao duplica.
func (l *PersonaLedger) Record(scope string, entry PersonaLedgerEntry) (PersonaLedgerEntry, bool, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return PersonaLedgerEntry{}, false, fmt.Errorf("%w: scope is required", ErrPersonaInvalid)
	}
	if strings.TrimSpace(entry.PersonaID) == "" || strings.TrimSpace(string(entry.Target)) == "" ||
		strings.TrimSpace(entry.Destination) == "" {
		return PersonaLedgerEntry{}, false, fmt.Errorf("%w: persona, target and destination are required", ErrPersonaInvalid)
	}
	if entry.SourceHash == "" || entry.RenderedHash == "" {
		// *Um registo sem hashes nao reconcilia: aceita-se, mas nao se finge que mede.*
		return PersonaLedgerEntry{}, false, fmt.Errorf("%w: source and rendered hashes are required", ErrPersonaInvalid)
	}
	// O caminho e' indexado pelo escopo: um escopo que nao e' da organizacao autenticada escreveria no
	// ficheiro de outra. A guarda tem de vir ANTES de o caminho ser montado.
	if err := requireOrganization(scope); err != nil {
		return PersonaLedgerEntry{}, false, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.loadLocked(scope); err != nil {
		return PersonaLedgerEntry{}, false, err
	}
	chave := PersonaLedgerKey(entry)
	if existente, jaExiste := l.byScope[scope][chave]; jaExiste {
		if existente.SourceHash == entry.SourceHash && existente.RenderedHash == entry.RenderedHash {
			// Retry do mesmo pedido: nao duplica.
			return existente, false, nil
		}
		// Mesma chave, conteudo diferente: nao e' retry, e' conflito.
		return existente, false, fmt.Errorf("%w: %s already recorded with different hashes",
			ErrPluginVersionConflict, entry.PersonaID)
	}
	l.byScope[scope][chave] = entry
	if err := l.persistLocked(scope); err != nil {
		// A gravacao em memoria nao sobrevive a falha de persistencia: desfazer para nao mentir sobre o
		// estado.
		delete(l.byScope[scope], chave)
		return PersonaLedgerEntry{}, false, err
	}
	return entry, true, nil
}

// persistLocked grava o escopo com as chaves ORDENADAS.
//
// *Duas gravacoes do mesmo ledger tem de produzir os mesmos bytes, senao o ficheiro muda sem que nada tenha
// mudado -- e um diff que aparece do nada e' pior do que nenhum.*
func (l *PersonaLedger) persistLocked(scope string) error {
	if l.root == "" {
		return nil
	}
	escopo := l.byScope[scope]
	chaves := make([]string, 0, len(escopo))
	for chave := range escopo {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	lista := make([]PersonaLedgerEntry, 0, len(chaves))
	for _, chave := range chaves {
		lista = append(lista, escopo[chave])
	}
	return writeJSONAtomic(l.scopePath(scope), lista)
}

// Load le o escopo do disco. Uma gravacao adulterada e' RECUSADA, nao carregada.
func (l *PersonaLedger) Load(scope string) error {
	scope = strings.TrimSpace(scope)
	if l.root == "" || scope == "" {
		return nil
	}
	if err := requireOrganization(scope); err != nil {
		return err
	}
	raw, err := os.ReadFile(l.scopePath(scope))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var lista []PersonaLedgerEntry
	if err := json.Unmarshal(raw, &lista); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrPersonaLedgerCorrupt, err)
	}
	escopo := make(map[string]PersonaLedgerEntry, len(lista))
	for _, entrada := range lista {
		if strings.TrimSpace(entrada.PersonaID) == "" {
			continue
		}
		if entrada.SourceHash == "" || entrada.RenderedHash == "" {
			// *Um registo sem hashes nao se carrega como se medisse alguma coisa.*
			return fmt.Errorf("%w: entry %s has no hashes", ErrPersonaLedgerCorrupt, entrada.PersonaID)
		}
		escopo[PersonaLedgerKey(entrada)] = entrada
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byScope[scope] = escopo
	return nil
}

// Entries devolve as entradas do escopo, ordenadas por chave.
func (l *PersonaLedger) Entries(scope string) []PersonaLedgerEntry {
	// Leitura tambem e' scoped: ver o ledger de outra organizacao e' tao grave como escrever nele.
	_ = requireOrganization(scope)
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.loadLocked(strings.TrimSpace(scope)); err != nil {
		// Leitura nao devolve erro hoje; devolver nil e' o mesmo que hoje, mas o ficheiro corrompido deixa
		// de ser lido como «vazio».
		return nil
	}
	escopo := l.byScope[strings.TrimSpace(scope)]
	chaves := make([]string, 0, len(escopo))
	for chave := range escopo {
		chaves = append(chaves, chave)
	}
	sort.Strings(chaves)
	saida := make([]PersonaLedgerEntry, 0, len(chaves))
	for _, chave := range chaves {
		saida = append(saida, escopo[chave])
	}
	return saida
}

// Reconciliation e' o veredicto para UMA entrada.
type Reconciliation struct {
	Entry  PersonaLedgerEntry `json:"entry"`
	State  DriftState         `json:"state"`
	Action string             `json:"action"`
}

// Reconcile compara o ledger com o disco.
//
// `noDisco` mapeia destino -> bytes. Destino ausente do mapa significa ficheiro inexistente.
// `esperadoAgora` pode ser nil quando a persona de origem ja' nao existe.
func (l *PersonaLedger) Reconcile(scope string, noDisco map[string][]byte, esperadoAgora func(PersonaLedgerEntry) []byte) []Reconciliation {
	entradas := l.Entries(scope)
	saida := make([]Reconciliation, 0, len(entradas))
	for _, entrada := range entradas {
		atual, existe := noDisco[entrada.Destination]
		if !existe {
			atual = nil
		}
		var esperado []byte
		if esperadoAgora != nil {
			esperado = esperadoAgora(entrada)
		}
		estado := ClassifyDrift(entrada, atual, esperado)
		saida = append(saida, Reconciliation{Entry: entrada, State: estado, Action: DescribeDrift(estado)})
	}
	return saida
}

// DriftSummary conta os estados. Devolve SEMPRE todas as chaves, incluindo as de valor zero.
//
// *Um resumo que omite o estado com zero faz o ecra dizer "tudo actual" quando nao ha' nada registado.*
func DriftSummary(itens []Reconciliation) map[DriftState]int {
	todos := []DriftState{
		DriftCurrent, DriftOutdated, DriftLocallyModified, DriftRemoved,
		DriftForeign, DriftConflicted, DriftBroken,
	}
	resumo := make(map[DriftState]int, len(todos))
	for _, estado := range todos {
		resumo[estado] = 0
	}
	for _, item := range itens {
		resumo[item.State]++
	}
	return resumo
}
