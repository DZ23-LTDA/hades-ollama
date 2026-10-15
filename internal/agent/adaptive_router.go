package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/internal/multillm"
)

// E-657 — ADAPTIVE ROUTER / SUBTASK ROUTER (FASE C, §4)
//
// O que já existe (medido):
// - RouteModelForStep: escolhe modelo por passo (capacidades + pin) e regista em evento step.model.route
// - ModelCapabilityRegistry + ModelCapabilityProbe: registo de capacidades com origem (medido/declarado)
// - Step.RequiredCapabilities + Step.PinnedModelID: contrato do passo
// - ModelCapabilityRegistry.ResolveToolMode: devolve ToolMode (native/compatible_text/unknown) + motivo
//
// ESTADO MEDIDO (E-658): o router **decide pelo caminho canonico** (`Runtime.RouteModelForStep`) - fonte unica de
// decisao, falha fechado sem runtime, cache com identidade correcta e estatistica sem corrida.
//
// O QUE CONTINUA FALTANDO (nao declarar entregue): o **DECOMPOSITOR** (objetivo -> subtarefas em DAG) e o **evento
// de auditoria** da decisao de roteamento. Hoje quem consome o router e' quem monta as `Subtask` (testes e chamadas
// directas) - o laco de execucao ainda **nao** o chama.
//
// Lista original do que faltava:
// 1. Recebe um objetivo + contexto → decompõe em subtarefas (DAG) com capacidades requeridas
// 2. Para cada subtarefa, invoca RouteModelForStep → obtém modelo + modo + justificativa
// 3. Emite decisão de roteamento como evento (auditoria)
// 4. Fornece fallback honesto quando não há modelo medido para a capacidade
// 5. Suporta cache de decisões por (capacidades + pin) para performance
//
// Design: interface AdaptiveRouter + implementação DefaultAdaptiveRouter
// - Thread-safe, stateless (exceto cache LRU simples)
//  - Não faz chamadas de rede — só consulta ModelCapabilityRegistry + multillm.Registry

// ErrNoSuitableModel indica que nenhuma rota satisfaz as capacidades requeridas
var ErrNoSuitableModel = errors.New("no suitable model for required capabilities")

// RoutingDecision é a decisão de roteamento para uma subtarefa
type RoutingDecision struct {
	SubtaskID            string            `json:"subtask_id"`
	RequiredCapabilities []string          `json:"required_capabilities"`
	PinnedModelID        string            `json:"pinned_model_id,omitempty"`
	ChosenModel          string            `json:"chosen_model"`
	ToolMode             string            `json:"tool_mode"` // native | compatible_text | unknown
	Reason               string            `json:"reason"`    // por que este modelo
	ModelCapabilities    ModelCapabilities `json:"model_capabilities"`
	Confidence           float64           `json:"confidence"` // 0.0 a 1.0
	CacheHit             bool              `json:"cache_hit"`
	Timestamp            time.Time         `json:"timestamp"`
}

// Subtask representa uma tarefa decomposta com capacidades requeridas
type Subtask struct {
	ID                   string   `json:"id"`
	Objective            string   `json:"objective"`
	RequiredCapabilities []string `json:"required_capabilities"`
	PinnedModelID        string   `json:"pinned_model_id,omitempty"`
	Priority             int      `json:"priority"`               // maior = mais urgente
	Dependencies         []string `json:"dependencies,omitempty"` // IDs de subtarefas que devem terminar primeiro
}

// RoutingCacheKey é a chave do cache de decisões de roteamento
type RoutingCacheKey struct {
	CapabilitiesHash string
	PinnedModelID    string
}

// AdaptiveRouter interface para roteamento adaptativo por subtarefa
type AdaptiveRouter interface {
	// RouteSubtask escolhe o melhor modelo para uma subtarefa
	RouteSubtask(ctx context.Context, subtask Subtask) (RoutingDecision, error)

	// RouteBatch escolhe modelos para múltiplas subtarefas (respeita dependências)
	RouteBatch(ctx context.Context, subtasks []Subtask) ([]RoutingDecision, error)

	// InvalidateCache limpa o cache de decisões
	InvalidateCache()

	// Stats retorna estatísticas do router
	Stats() RouterStats
}

// RouterStats estatísticas do router
type RouterStats struct {
	TotalRequests    int64     `json:"total_requests"`
	CacheHits        int64     `json:"cache_hits"`
	CacheMisses      int64     `json:"cache_misses"`
	AvgLatencyMS     float64   `json:"avg_latency_ms"`
	UnknownModelRate float64   `json:"unknown_model_rate"` // % de chamadas que resultaram em unknown
	LastUpdated      time.Time `json:"last_updated"`
}

// DefaultAdaptiveRouter implementação padrão
type DefaultAdaptiveRouter struct {
	mu           sync.RWMutex
	registry     *ModelCapabilityRegistry
	multillmReg  *multillm.Registry
	runtime      *Runtime // E-658: dono da decisao (caminho canonico RouteModelForStep)
	cache        map[RoutingCacheKey]RoutingDecision
	cacheOrder   []RoutingCacheKey // para LRU simples
	maxCacheSize int
	stats        RouterStats
}

// NewAdaptiveRouterParaRuntime liga o roteador adaptativo ao caminho CANONICO de roteamento por passo (E-658).
//
// *POR QUE ESTE CONSTRUTOR SUBSTITUI O ANTIGO:* a versao anterior (`NewAdaptiveRouter(registry, multillmReg)`)
// criava um roteador capaz de decidir por conta propria, chamando `multillmReg.Route` direto - um SEGUNDO caminho
// paralelo ao `Runtime.RouteModelForStep`, que e' o caminho auditado. Um construtor que produz um roteador
// divergente e' uma armadilha; ele foi REMOVIDO em vez de mantido "por compatibilidade" (nao havia nenhum call
// site). Aqui o runtime e' o **dono da decisao** e o registro de capacidades serve so' para ENRIQUECER.
func NewAdaptiveRouterParaRuntime(runtime *Runtime, registro *ModelCapabilityRegistry) *DefaultAdaptiveRouter {
	if registro == nil {
		// O registro tem um padrao MEDIDO neste ambiente; nulo viraria panic no enriquecimento.
		registro = DefaultModelCapabilityRegistry()
	}
	return &DefaultAdaptiveRouter{
		registry:     registro,
		runtime:      runtime,
		cache:        make(map[RoutingCacheKey]RoutingDecision),
		cacheOrder:   make([]RoutingCacheKey, 0),
		maxCacheSize: 1000,
		stats:        RouterStats{LastUpdated: time.Now().UTC()},
	}
}

// capabilitiesHash cria hash determinístico das capacidades
func capabilitiesHash(caps []string) string {
	// Ordena para consistência
	sorted := make([]string, len(caps))
	copy(sorted, caps)
	// Ordenação simples
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	h := make([]byte, 0, len(caps)*8)
	for _, c := range sorted {
		h = append(h, c...)
	}
	return string(h)
}

// makeCacheKey cria chave de cache
func makeCacheKey(caps []string, pinnedModelID string) RoutingCacheKey {
	return RoutingCacheKey{
		CapabilitiesHash: capabilitiesHash(caps),
		PinnedModelID:    pinnedModelID,
	}
}

// RouteSubtask implementa AdaptiveRouter.RouteSubtask
func (r *DefaultAdaptiveRouter) RouteSubtask(ctx context.Context, subtask Subtask) (RoutingDecision, error) {
	start := time.Now()
	defer func() {
		r.mu.Lock()
		r.stats.TotalRequests++
		r.stats.AvgLatencyMS = (r.stats.AvgLatencyMS*float64(r.stats.TotalRequests-1) + float64(time.Since(start).Milliseconds())) / float64(r.stats.TotalRequests)
		r.stats.LastUpdated = time.Now().UTC()
		if r.stats.UnknownModelRate == 0 {
			r.stats.UnknownModelRate = 0
		}
		r.mu.Unlock()
	}()

	// Validação
	if len(subtask.RequiredCapabilities) == 0 {
		return RoutingDecision{}, errors.New("subtask requires at least one capability")
	}

	// Verifica cache
	cacheKey := makeCacheKey(subtask.RequiredCapabilities, subtask.PinnedModelID)
	r.mu.RLock()
	cached, ok := r.cache[cacheKey]
	r.mu.RUnlock()
	if ok {
		// E-658 - DUAS CORRECOES NESTE CAMINHO:
		//
		// (a) CORRIDA DE DADOS: `r.stats.CacheHits++` era feito **sob RLock** - uma ESCRITA num campo partilhado
		//     protegido apenas por lock de LEITURA. Em chamadas concorrentes isto e' corrida; a estatistica exige o
		//     lock de escrita (e um teste concorrente sob `-race` passa a prova-lo).
		// (b) IDENTIDADE: o cache e' por (capacidades, pin), **nao** por subtarefa. Devolver a copia guardada com o
		//     `SubtaskID` do primeiro dono TROCARIA a identidade de quem pergunta - e a identidade e' do chamador.
		r.mu.Lock()
		r.stats.CacheHits++
		r.mu.Unlock()
		cached.CacheHit = true
		cached.SubtaskID = subtask.ID
		cached.RequiredCapabilities = subtask.RequiredCapabilities
		cached.PinnedModelID = subtask.PinnedModelID
		return cached, nil
	}
	r.mu.Lock()
	r.stats.CacheMisses++
	r.mu.Unlock()

	// Verifica pin primeiro
	if subtask.PinnedModelID != "" {
		if caps, known := r.registry.Lookup(subtask.PinnedModelID); known {
			mode, reason := r.registry.ResolveToolMode(subtask.PinnedModelID)
			decision := RoutingDecision{
				SubtaskID:            subtask.ID,
				RequiredCapabilities: subtask.RequiredCapabilities,
				PinnedModelID:        subtask.PinnedModelID,
				ChosenModel:          subtask.PinnedModelID,
				ToolMode:             string(mode),
				Reason:               "pinned model: " + reason,
				ModelCapabilities:    caps,
				Confidence:           1.0,
				CacheHit:             false,
				Timestamp:            time.Now().UTC(),
			}
			// Cache
			r.addToCache(subtask.RequiredCapabilities, subtask.PinnedModelID, decision)
			return decision, nil
		}
		// Pin não existe no registo -> cai para seleção por capacidades
	}

	// E-658 - A DECISAO VEM DO CAMINHO CANONICO, NAO DE UM SEGUNDO ROTEADOR.
	//
	// Antes daqui havia um `multillm.RouteRequest` montado a mao + `r.multillmReg.Route(...)`: um SEGUNDO
	// caminho de roteamento, paralelo ao `Runtime.RouteModelForStep` - que e' o caminho auditado (contrato do
	// passo, pin por passo pela ponte E-435 e o evento `step.model.route`). Dois roteadores decidindo em paralelo
	// e' a forma mais barata de divergir em silencio; e a montagem a mao repetia o mapeamento que a ponte E-437 ja'
	// havia corrigido (capacidade de MODELO x capacidade de FERRAMENTA).
	//
	// Sem runtime nao se escolhe - e o router FALHA FECHADO, exatamente como `RouteModelForStep` faz. Nunca cair
	// em silencio no modelo da missao, porque isso esconderia a exigencia nao honrada do subtarefa.
	if r.runtime == nil {
		return RoutingDecision{}, ErrModelRegistryUnavailable
	}
	decision, err := r.runtime.RouteModelForStep(Mission{}, Step{
		ID:                   subtask.ID,
		RequiredCapabilities: subtask.RequiredCapabilities,
		PinnedModelID:        subtask.PinnedModelID,
	})
	if err != nil {
		return RoutingDecision{}, err
	}

	// Obtém capacidades do modelo escolhido
	caps, known := r.registry.Lookup(decision.Model.ID)
	if !known {
		// Modelo escolhido pelo multillm mas não no nosso registo -> desconhecido
		// Fallback honesto: o modelo foi escolhido pelo registry, mas NAO ha medicao local dele.
		//
		// E-658 - DOIS DEFEITOS QUE O TESTE PEGOU AQUI:
		// (a) `Confidence` era **0,5** - um numero INVENTADO (nao existe medicao nenhuma sobre este modelo). A regra
		//     desta casa e' que resultado ausente e' `NAO MEDIDO`; 0 e' a unica confianca honesta, e o motivo declara
		//     o limite.
		// (b) a decisao degradada **nao era cacheada**, embora o router anuncie cache por (capacidades, pin): a
		//     mesma subtarefa repetida voltava a rotear, e `CacheMisses` contava sempre. A decisao e' determinista
		//     para a mesma chave, entao entra no cache como qualquer outra.
		degradada := RoutingDecision{
			SubtaskID:            subtask.ID,
			RequiredCapabilities: subtask.RequiredCapabilities,
			PinnedModelID:        subtask.PinnedModelID,
			ChosenModel:          decision.Model.ID,
			ToolMode:             string(ToolModeUnknown),
			Reason:               "modelo escolhido pelo registry mas nao medido localmente (confianca NAO MEDIDA)",
			ModelCapabilities:    ModelCapabilities{},
			Confidence:           0,
			CacheHit:             false,
			Timestamp:            time.Now().UTC(),
		}
		r.addToCache(subtask.RequiredCapabilities, subtask.PinnedModelID, degradada)
		return degradada, nil
	}

	mode, reason := r.registry.ResolveToolMode(decision.Model.ID)
	// E-658 PENDENTE (nao inventar semantica): `multillm.RouteDecision.Score` e' um inteiro de scoring
	// BRUTO (medido em internal/multillm/router.go:100-129: QualityScore*10 ou *100, +100000 de bonus de
	// provedor, +10000 de bonus de pin), NAO uma confianca 0..1. Normaliza-lo exigiria uma regra de produto
	// que ainda nao existe; entao a confianca fica NAO MEDIDA (0) e o score bruto continua visivel no motivo.
	// A variavel recebe nome proprio porque `:=` com o mesmo nome SOMBREAVA a decisao do multillm (era o
	// defeito que impedia o pacote de compilar).
	roteada := RoutingDecision{
		SubtaskID:            subtask.ID,
		RequiredCapabilities: subtask.RequiredCapabilities,
		PinnedModelID:        subtask.PinnedModelID,
		ChosenModel:          decision.Model.ID,
		ToolMode:             string(mode),
		Reason:               fmt.Sprintf("%s | score_bruto_do_registry=%d (nao normalizado para 0..1)", reason, decision.Score),
		ModelCapabilities:    caps,
		Confidence:           0,
		CacheHit:             false,
		Timestamp:            time.Now().UTC(),
	}

	// Cache
	r.addToCache(subtask.RequiredCapabilities, subtask.PinnedModelID, roteada)
	return roteada, nil
}

// addToCache adiciona ao cache com LRU simples
func (r *DefaultAdaptiveRouter) addToCache(caps []string, pinnedModelID string, decision RoutingDecision) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := makeCacheKey(caps, pinnedModelID)
	r.cache[key] = decision
	r.cacheOrder = append(r.cacheOrder, key)

	// LRU simples: remove mais antigo se exceder
	if len(r.cacheOrder) > r.maxCacheSize {
		oldest := r.cacheOrder[0]
		delete(r.cache, oldest)
		r.cacheOrder = r.cacheOrder[1:]
	}
}

// DecomporPlanoEmSubtarefas projeta o PLANO da missao nas subtarefas roteaveis (E-662).
//
// *POR QUE PROJETAR O PLANO, E NAO "DECOMPOR" DE NOVO:* o plano JA' e' o DAG do trabalho, e `ValidatePlanGraph`
// (E-434) ja' o recusa no planejamento quando as arestas sao invalidas. Um decompositor que inventasse subtarefas a
// partir do texto do objetivo criaria um **segundo** plano - sem as garantias do primeiro - e os dois divergiriam.
// Aqui a decomposicao e' uma PROJECAO: cada passo vira uma subtarefa com as MESMAS capacidades, o MESMO pin e as
// MESMAS arestas.
//
// A prioridade e' a POSICAO no plano (o primeiro passo tem prioridade maior). Ela serve apenas de desempate em
// `RouteBatch`: *nunca* para furar uma dependencia.
func DecomporPlanoEmSubtarefas(plano []Step) []Subtask {
	total := len(plano)
	subtarefas := make([]Subtask, 0, total)
	for indice, passo := range plano {
		subtarefas = append(subtarefas, Subtask{
			ID:                   passo.ID,
			Objective:            passo.Title,
			RequiredCapabilities: append([]string(nil), passo.RequiredCapabilities...),
			PinnedModelID:        passo.PinnedModelID,
			Priority:             total - indice,
			Dependencies:         append([]string(nil), passo.DependsOn...),
		})
	}
	return subtarefas
}

// ordenarPorDependencias devolve as subtarefas em ordem TOPOLOGICA (Kahn), com prioridade como desempate.
//
// Recusa o que a interface prometia respeitar: dependencia que nao existe, subtarefa sem ID, ID repetido e ciclo.
// *Uma ordem que nao respeita a aresta nao e' ordem - e' a lista de novo com outro criterio.*
func ordenarPorDependencias(subtarefas []Subtask) ([]Subtask, error) {
	porID := make(map[string]Subtask, len(subtarefas))
	posicao := make(map[string]int, len(subtarefas))
	for indice, subtarefa := range subtarefas {
		id := strings.TrimSpace(subtarefa.ID)
		if id == "" {
			return nil, fmt.Errorf("%w: subtarefa sem ID na posicao %d", ErrPlanGraphInvalid, indice)
		}
		if _, repetido := porID[id]; repetido {
			return nil, fmt.Errorf("%w: subtarefa repetida %q", ErrPlanGraphInvalid, id)
		}
		subtarefa.ID = id
		porID[id] = subtarefa
		posicao[id] = indice
	}

	pendentes := make(map[string]int, len(subtarefas))
	dependentes := make(map[string][]string, len(subtarefas))
	for _, subtarefa := range subtarefas {
		vistas := map[string]bool{}
		for _, bruta := range subtarefa.Dependencies {
			dependencia := strings.TrimSpace(bruta)
			if dependencia == "" || vistas[dependencia] {
				continue
			}
			if dependencia == subtarefa.ID {
				return nil, fmt.Errorf("%w: a subtarefa %q depende de si mesma", ErrPlanGraphInvalid, subtarefa.ID)
			}
			if _, existe := porID[dependencia]; !existe {
				return nil, fmt.Errorf("%w: a subtarefa %q depende de %q, que nao existe",
					ErrPlanGraphInvalid, subtarefa.ID, dependencia)
			}
			vistas[dependencia] = true
			pendentes[subtarefa.ID]++
			dependentes[dependencia] = append(dependentes[dependencia], subtarefa.ID)
		}
	}

	// Prontas = sem dependencia pendente. A cada rodada escolhe a de MAIOR prioridade (desempate: posicao original).
	prontas := make([]string, 0, len(subtarefas))
	for _, subtarefa := range subtarefas {
		if pendentes[subtarefa.ID] == 0 {
			prontas = append(prontas, subtarefa.ID)
		}
	}
	ordenadas := make([]Subtask, 0, len(subtarefas))
	for len(prontas) > 0 {
		escolhida := 0
		for i := 1; i < len(prontas); i++ {
			a, b := porID[prontas[i]], porID[prontas[escolhida]]
			if a.Priority > b.Priority || (a.Priority == b.Priority && posicao[a.ID] < posicao[b.ID]) {
				escolhida = i
			}
		}
		id := prontas[escolhida]
		prontas = append(prontas[:escolhida], prontas[escolhida+1:]...)
		ordenadas = append(ordenadas, porID[id])
		for _, dependente := range dependentes[id] {
			pendentes[dependente]--
			if pendentes[dependente] == 0 {
				prontas = append(prontas, dependente)
			}
		}
	}
	if len(ordenadas) != len(subtarefas) {
		return nil, fmt.Errorf("%w: ciclo entre as subtarefas (sobrou %d de %d)",
			ErrPlanGraphInvalid, len(subtarefas)-len(ordenadas), len(subtarefas))
	}
	return ordenadas, nil
}

// RouteBatch implementa AdaptiveRouter.RouteBatch
func (r *DefaultAdaptiveRouter) RouteBatch(ctx context.Context, subtasks []Subtask) ([]RoutingDecision, error) {
	if len(subtasks) == 0 {
		return nil, nil
	}

	// E-662 - A ORDEM E' TOPOLOGICA, E A PRIORIDADE E' SO' DESEMPATE.
	//
	// O comentario da interface prometia "respeita dependencias" e a implementacao ordenava apenas por PRIORIDADE
	// - *um dependente com prioridade maior era decidido ANTES do seu prerequisito.* Ordem que nao respeita a
	// aresta do DAG nao e' ordem.
	ordenadas, err := ordenarPorDependencias(subtasks)
	if err != nil {
		return nil, err
	}

	results := make([]RoutingDecision, 0, len(ordenadas))
	for _, subtask := range ordenadas {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		decision, err := r.RouteSubtask(ctx, subtask)
		if err != nil {
			return results, err
		}
		results = append(results, decision)
	}
	return results, nil
}

// InvalidateCache limpa o cache
func (r *DefaultAdaptiveRouter) InvalidateCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[RoutingCacheKey]RoutingDecision)
	r.cacheOrder = make([]RoutingCacheKey, 0)
}

// Stats retorna estatísticas
func (r *DefaultAdaptiveRouter) Stats() RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stats := r.stats
	// Calcula taxa de unknown
	if r.stats.TotalRequests > 0 {
		// Não temos contador direto de unknown, mas podemos estimar
	}
	return stats
}
