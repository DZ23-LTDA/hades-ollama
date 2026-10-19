package agent

// FASE 28.7: o store dos builds.
//
// O DEFEITO QUE ESTE FICHEIRO FECHA
//
// O E-1306 entregou `build.start` e o doc-comment do `build.verify` dizia, com honestidade:
//
//	"nao existe registo de builds"
//
// O adendo pede `build.status`, `build.logs`, `build.cancel` e `build.artifacts`. Nenhum deles pode
// existir sem um sitio onde o build VIVA depois de a rota responder. Uma rota `build.status` que
// reconstroi o estado a partir do pedido nao consulta estado nenhum: consulta o cliente.
//
//	*Uma rota de estado sem store nao e' uma rota de estado: e' um eco.*
//
// O QUE ESTE STORE E' E O QUE NAO E'
//
// E' um registo em memoria, com cancelamento por contexto e escrita opcional em disco. NAO e' um
// event log imutavel, nao e' um broker e nao e' uma fila. Um build vive enquanto o processo vive; a
// persistencia em disco e' um extra para diagnostico, nao uma promessa de durabilidade.
//
// O QUE ELE GARANTE
//
//  1. um build tem ID estavel e estado explicito, nunca inferido do pedido;
//  2. o estado transita para CANCELADO por pedido E por contexto, e a transicao e' visivel;
//  3. `cancelar` devolve falso quando o build ja' terminou -- nao finge cancelar o que acabou;
//  4. a saida do comando fica guardada e e' limitada, para nao transformar um build ruidoso em
//     consumo de memoria sem tecto;
//  5. dois pedidos de cancelamento do mesmo build nao disparam dois cancelamentos.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// EstadoDeBuild e' o estado observavel de um build. Os valores sao texto porque atravessam a API e
// um cliente tem de os poder comparar sem conhecer um enum numerico.
type EstadoDeBuild string

const (
	// BuildNaFila ainda nao comecou a correr.
	BuildNaFila EstadoDeBuild = "NA_FILA"
	// BuildAExecutar esta' a correr.
	BuildAExecutar EstadoDeBuild = "A_EXECUTAR"
	// BuildConcluido terminou sem erro.
	BuildConcluido EstadoDeBuild = "CONCLUIDO"
	// BuildFalhou terminou com erro.
	BuildFalhou EstadoDeBuild = "FALHOU"
	// BuildCancelado foi cancelado antes de terminar.
	BuildCancelado EstadoDeBuild = "CANCELADO"
)

// maxSaidaGuardada limita a saida guardada por build.
//
//	*Uma saida sem limite e' uma fuga de memoria com autorizacao do compilador.*
const maxSaidaGuardada = 256 * 1024

// maxBuildsGuardados limita quantos builds o store mantem. Ao ultrapassar, os mais antigos ja'
// terminados sao descartados -- um build a correr nunca e' descartado.
const maxBuildsGuardados = 200

// Erros do store.
var (
	// ErrBuildDesconhecido e' devolvido quando o ID nao existe.
	ErrBuildDesconhecido = errors.New("build desconhecido")
	// ErrBuildJaTerminou e' devolvido ao cancelar um build que ja' acabou.
	ErrBuildJaTerminou = errors.New("build ja terminou")
)

// RegistoDeBuild e' o que o store guarda por build. E' uma copia segura: o store devolve copias e
// nunca o ponteiro interno.
type RegistoDeBuild struct {
	ID             string            `json:"id"`
	Alvo           string            `json:"alvo"`
	Raiz           string            `json:"raiz"`
	Destino        string            `json:"destino"`
	Estado         EstadoDeBuild     `json:"estado"`
	IniciadoEm     time.Time         `json:"iniciado_em"`
	TerminadoEm    *time.Time        `json:"terminado_em,omitempty"`
	Resultado      *ResultadoDeBuild `json:"resultado,omitempty"`
	Erro           string            `json:"erro,omitempty"`
	SaidaDoComando string            `json:"saida_do_comando,omitempty"`
	// Cancelavel diz se ainda faz sentido pedir cancelamento. E' derivado do estado, mas vai no
	// JSON porque um cliente que tenha de o calcular esta' a reimplementar a maquina de estados.
	Cancelavel bool `json:"cancelavel"`
	// NadaAssinado e NadaPublicado sao invariantes de fronteira: um build nunca assina nem publica.
	NadaAssinado  bool `json:"nada_assinado"`
	NadaPublicado bool `json:"nada_publicado"`
}

// BuildStore guarda builds em memoria e, opcionalmente, em disco.
type BuildStore struct {
	mu              sync.Mutex
	registos        map[string]*RegistoDeBuild
	cancelamentos   map[string]context.CancelFunc
	contadorDeOrdem int
	dir             string
	agora           func() time.Time
}

// NovoBuildStore cria um store sem persistencia em disco.
func NovoBuildStore() *BuildStore {
	return &BuildStore{
		registos:      map[string]*RegistoDeBuild{},
		cancelamentos: map[string]context.CancelFunc{},
		agora:         time.Now,
	}
}

// NovoBuildStoreComDirectorio cria um store que escreve cada registo em `dir`. O directorio e'
// criado se faltar. Se nao for possivel criar, o store funciona so' em memoria -- a persistencia e'
// um extra de diagnostico, e recusar construir por causa dela seria pior do que nao persistir.
func NovoBuildStoreComDirectorio(dir string) *BuildStore {
	s := NovoBuildStore()
	if dir == "" {
		return s
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return s
	}
	s.dir = dir
	return s
}

// novoID gera um identificador estavel e opaco. Nao usa o alvo nem o tempo legivel: um ID que
// revela o conteudo e' um ID que convida a ser adivinhado.
func novoID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Um ID nao pode falhar. Se o gerador falhar, o build e' recusado em vez de receber um ID
		// previsivel -- um ID previsivel e' uma colisao a espera.
		return ""
	}
	return hex.EncodeToString(b[:])
}

// Registar cria um registo NA_FILA e devolve o ID. Devolve erro se o ID nao puder ser gerado.
func (s *BuildStore) Registar(alvo, raiz, destino string) (string, error) {
	id := novoID()
	if id == "" {
		return "", errors.New("nao foi possivel gerar identificador de build")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registos[id] = &RegistoDeBuild{
		ID:            id,
		Alvo:          alvo,
		Raiz:          raiz,
		Destino:       destino,
		Estado:        BuildNaFila,
		IniciadoEm:    s.agora().UTC(),
		Cancelavel:    true,
		NadaAssinado:  true,
		NadaPublicado: true,
	}
	s.contadorDeOrdem++
	s.podar()
	return id, nil
}

// podar descarta os builds terminados mais antigos quando o store passa do tecto. Deve ser chamado
// com o lock tomado.
func (s *BuildStore) podar() {
	if len(s.registos) <= maxBuildsGuardados {
		return
	}
	type par struct {
		id string
		em time.Time
	}
	var terminados []par
	for id, r := range s.registos {
		if r.Estado == BuildConcluido || r.Estado == BuildFalhou || r.Estado == BuildCancelado {
			terminados = append(terminados, par{id, r.IniciadoEm})
		}
	}
	sort.Slice(terminados, func(i, j int) bool { return terminados[i].em.Before(terminados[j].em) })
	excesso := len(s.registos) - maxBuildsGuardados
	for i := 0; i < len(terminados) && i < excesso; i++ {
		delete(s.registos, terminados[i].id)
		delete(s.cancelamentos, terminados[i].id)
	}
}

// Iniciar marca o build como A_EXECUTAR e devolve um contexto cancelavel. O `cancelar` devolvido
// esta' ligado ao store: chama-lo duas vezes e' inofensivo.
func (s *BuildStore) Iniciar(id string) (context.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registos[id]
	if !ok {
		return nil, ErrBuildDesconhecido
	}
	if r.Estado != BuildNaFila {
		return nil, errors.New("build ja' arrancou ou terminou: estado " + string(r.Estado))
	}
	ctx, cancelar := context.WithCancel(context.Background())
	s.cancelamentos[id] = cancelar
	r.Estado = BuildAExecutar
	r.Cancelavel = true
	return ctx, nil
}

// Terminar grava o resultado final. E' idempotente por estado: se o build ja' foi cancelado, o
// resultado nao o "des-cancela".
//
//	*Um build cancelado que depois se declara CONCLUIDO e' um cancelamento que mentiu.*
func (s *BuildStore) Terminar(id string, resultado *ResultadoDeBuild, erroDoBuild error, saida string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registos[id]
	if !ok {
		return ErrBuildDesconhecido
	}
	agora := s.agora().UTC()
	r.TerminadoEm = &agora
	r.Cancelavel = false
	if len(saida) > maxSaidaGuardada {
		saida = saida[:maxSaidaGuardada] + "\n[saida truncada em " + itoaDoStore(maxSaidaGuardada) + " bytes]"
	}
	r.SaidaDoComando = saida
	if r.Estado == BuildCancelado {
		// O cancelamento ganha. O resultado fica guardado para diagnostico, mas o estado nao muda.
		r.Resultado = resultado
		s.persistir(r)
		return nil
	}
	r.Resultado = resultado
	if erroDoBuild != nil {
		r.Estado = BuildFalhou
		r.Erro = erroDoBuild.Error()
	} else {
		r.Estado = BuildConcluido
	}
	if cancel, ok := s.cancelamentos[id]; ok {
		cancel()
		delete(s.cancelamentos, id)
	}
	s.persistir(r)
	return nil
}

// Cancelar pede o cancelamento. Devolve ErrBuildJaTerminou se o build ja' acabou -- um pedido de
// cancelamento que "funciona" sobre um build terminado e' um botao que mente.
func (s *BuildStore) Cancelar(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registos[id]
	if !ok {
		return ErrBuildDesconhecido
	}
	if r.Estado == BuildConcluido || r.Estado == BuildFalhou || r.Estado == BuildCancelado {
		return ErrBuildJaTerminou
	}
	r.Estado = BuildCancelado
	r.Cancelavel = false
	agora := s.agora().UTC()
	r.TerminadoEm = &agora
	if cancel, ok := s.cancelamentos[id]; ok {
		cancel()
		delete(s.cancelamentos, id)
	}
	s.persistir(r)
	return nil
}

// Obter devolve uma copia do registo.
func (s *BuildStore) Obter(id string) (RegistoDeBuild, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.registos[id]
	if !ok {
		return RegistoDeBuild{}, ErrBuildDesconhecido
	}
	return *r, nil
}

// Listar devolve todos os registos, mais recentes primeiro.
func (s *BuildStore) Listar() []RegistoDeBuild {
	s.mu.Lock()
	defer s.mu.Unlock()
	saida := make([]RegistoDeBuild, 0, len(s.registos))
	for _, r := range s.registos {
		saida = append(saida, *r)
	}
	sort.Slice(saida, func(i, j int) bool {
		if saida[i].IniciadoEm.Equal(saida[j].IniciadoEm) {
			return saida[i].ID < saida[j].ID
		}
		return saida[i].IniciadoEm.After(saida[j].IniciadoEm)
	})
	return saida
}

// persistir escreve o registo em disco, se houver directorio. Deve ser chamado com o lock tomado.
// Falha de escrita nao propaga: o estado em memoria e' a verdade operacional.
func (s *BuildStore) persistir(r *RegistoDeBuild) {
	if s.dir == "" {
		return
	}
	corpo, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(s.dir, r.ID+".json"), corpo, 0o644)
}

// itoaDoStore evita importar strconv so' para um caso.
func itoaDoStore(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
