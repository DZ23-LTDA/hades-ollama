package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/multillm"
)

// E-666 — O EXECUTOR DE INFERENCIA E A PROVA DE QUE A ROTA E' APLICADA (§4).
//
// **O QUE ESTAVA MEDIDO:** nada no runtime chamava modelo. A decisao por passo era tomada, registrada em
// `step.model.route`, entregue no `ToolContext.ModelID`... e **nenhuma ferramenta a consumia**.
//
// **O QUE ESTE TESTE PROVA, e nao e' pouco:** o modelo que EXECUTOU a inferencia e' o modelo que o MOTOR escolheu
// para AQUELE passo - (a) o mesmo do evento registrado; (b) o PRIMEIRO candidato de `RouteCandidates` para as
// capacidades daquele passo. *Se o executor usasse "o modelo da missao" ou um default, (b) falharia.*
func TestPassoDeInferenciaUsaOModeloRoteadoParaEle(t *testing.T) {
	cliente := &clienteDeInferenciaFalso{}
	passos := []Step{
		{
			ID: "codigo", Kind: KindDeInferencia, Title: "resolver em codigo", State: StepPending,
			RequiredCapabilities: []string{"coding"},
			Input:                map[string]any{"prompt": "escreva a funcao pedida"},
		},
		{
			ID: "imagem", Kind: KindDeInferencia, Title: "descrever a imagem", State: StepPending,
			RequiredCapabilities: []string{"vision"},
			Input:                map[string]any{"prompt": "descreva a imagem"},
		},
	}
	registro := registryDeTeste(t)
	runtime, missao := missaoDeInferencia(t, cliente, passos, registro)

	if err := runtime.Run(context.Background(), missao.ID); err != nil {
		t.Fatalf("executar a missao de inferencia: %v", err)
	}

	chamadas := cliente.Chamadas()
	if len(chamadas) != 2 {
		t.Fatalf("esperadas 2 chamadas de inferencia (uma por passo), vieram %d: %+v — *um passo de inferencia que "+
			"nao chama modelo e' um passo que mente no nome*", len(chamadas), chamadas)
	}

	// A trilha REGISTRADA, na ordem de execucao.
	roteados := modelosRoteados(t, runtime, missao.ID)
	if len(roteados) != len(chamadas) {
		t.Fatalf("a trilha registrou %d rotas e o executor fez %d chamadas — *registro e execucao tem de contar a "+
			"mesma historia*", len(roteados), len(chamadas))
	}
	for indice := range chamadas {
		if chamadas[indice].modelo != roteados[indice] {
			t.Fatalf("a chamada %d executou em %q, mas a rota registrada foi %q — *a decisao foi tomada, registrada "+
				"e NAO aplicada*", indice, chamadas[indice].modelo, roteados[indice])
		}
	}

	// (b) O modelo que executou e' o PRIMEIRO candidato do motor para AS CAPACIDADES DAQUELE PASSO.
	for indice, passo := range passos {
		candidatos, err := registro.RouteCandidates(RouteRequestForStep(missao, passo))
		if err != nil {
			t.Fatalf("candidatos do passo %q: %v", passo.Title, err)
		}
		if candidatos[0].Model.ID != chamadas[indice].modelo {
			t.Fatalf("o passo %q (capacidades %v) devia executar no primeiro candidato %q e executou em %q",
				passo.Title, passo.RequiredCapabilities, candidatos[0].Model.ID, chamadas[indice].modelo)
		}
	}

	// O prompt CHEGOU ao modelo (nao basta a rota chegar): o texto do ultimo `Message` e' o do passo.
	if chamadas[0].prompt != "escreva a funcao pedida" {
		t.Fatalf("o prompt do passo 1 nao chegou ao modelo (chegou %q)", chamadas[0].prompt)
	}
	t.Logf("rotas aplicadas: %v (capacidades dos passos: %v / %v)",
		roteados, passos[0].RequiredCapabilities, passos[1].RequiredCapabilities)
}

// CONTRAPOSTA 1: sem modelo roteado, a inferencia NAO acontece. *Zero rota nao e' licenca para escolher um modelo:
// e' ausencia de decisao, e ausencia de decisao nao se inventa.*
func TestPassoDeInferenciaRecusaSemModeloRoteado(t *testing.T) {
	cliente := &clienteDeInferenciaFalso{}
	ferramenta := inferenceTool{cliente: cliente}
	_, err := ferramenta.Execute(context.Background(), ToolContext{MissionID: "m", StepID: "s"}, map[string]any{"prompt": "ola"})
	if !errors.Is(err, ErrInferenciaSemModeloRoteado) {
		t.Fatalf("esperado ErrInferenciaSemModeloRoteado, veio %v", err)
	}
	if len(cliente.Chamadas()) != 0 {
		t.Fatal("NAO pode haver chamada de modelo quando nao ha modelo roteado")
	}
}

// CONTRAPOSTA 2: sem cliente de inferencia o kind NAO se registra - *fail-closed no PLANEJAMENTO, que e' onde a
// recusa e' barata, em vez de um kind que aceita o passo e falha no meio da execucao.*
func TestPassoDeInferenciaNaoSeRegistraSemCliente(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws", DataRoot: t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("construir runtime sem cliente de inferencia: %v", err)
	}
	if _, existe := runtime.tools.Get(KindDeInferencia); existe {
		t.Fatalf("sem cliente injetado o kind %q NAO pode existir", KindDeInferencia)
	}
}

// CONTRAPOSTA 3: prompt ausente e' recusado (e o descritor DECLARA `prompt` como input obrigatorio).
func TestPassoDeInferenciaRecusaSemPrompt(t *testing.T) {
	ferramenta := inferenceTool{cliente: &clienteDeInferenciaFalso{}}
	_, err := ferramenta.Execute(context.Background(), ToolContext{ModelID: "local/forte"}, map[string]any{})
	if !errors.Is(err, ErrInferenciaSemPrompt) {
		t.Fatalf("esperado ErrInferenciaSemPrompt, veio %v", err)
	}
	if !listaContem(ferramenta.Descriptor().RequiredInputs, "prompt") {
		t.Fatal("o descritor tem de DECLARAR `prompt` como input obrigatorio")
	}
}

// CONTRAPOSTA 4: resposta vazia e' DECLARADA (`empty: true`) - *nao se confunde resposta vazia com entrega*.
func TestRespostaVaziaDaInferenciaEDeclarada(t *testing.T) {
	ferramenta := inferenceTool{cliente: &clienteDeInferenciaFalso{semConteudo: true}}
	resultado, err := ferramenta.Execute(context.Background(), ToolContext{ModelID: "local/forte"}, map[string]any{"prompt": "ola"})
	if err != nil {
		t.Fatalf("inferencia: %v", err)
	}
	valor, ok := resultado.Value.(map[string]any)
	if !ok {
		t.Fatalf("o valor do resultado veio como %T", resultado.Value)
	}
	if vazio, _ := valor["empty"].(bool); !vazio {
		t.Fatalf("resposta vazia tem de vir com `empty: true`, veio %v", valor)
	}
	if modelo, _ := valor["model"].(string); modelo != "local/forte" {
		t.Fatalf("o resultado tem de declarar o modelo USADO, veio %q", modelo)
	}
}

// E-666 — A PROVA DO §4: **CAPACIDADES DIFERENTES EXECUTAM EM MODELOS DIFERENTES.**
//
// *Por que este teste existe separado do anterior:* com o registro padrao de teste, `local/forte` cobre `coding` **e**
// `vision` — entao o teste anterior prova que **a rota e' aplicada**, mas nao prova que o roteamento **distingue**
// capacidades (as duas rotas poderiam ser o mesmo modelo sem que nada falhasse). Aqui o registro tem especialistas
// **disjuntos**, e a asserção passa a ser sobre a DIFERENCA.
func TestCapacidadesDiferentesExecutamEmModelosDiferentes(t *testing.T) {
	cliente := &clienteDeInferenciaFalso{}
	passos := []Step{
		{
			ID: "codigo", Kind: KindDeInferencia, Title: "resolver em codigo", State: StepPending,
			RequiredCapabilities: []string{"coding"},
			Input:                map[string]any{"prompt": "corrija a funcao"},
		},
		{
			ID: "imagem", Kind: KindDeInferencia, Title: "descrever a imagem", State: StepPending,
			RequiredCapabilities: []string{"vision"},
			Input:                map[string]any{"prompt": "descreva a imagem"},
		},
	}
	runtime, missao := missaoDeInferencia(t, cliente, passos, registroDeEspecialistasDisjuntos(t))
	if err := runtime.Run(context.Background(), missao.ID); err != nil {
		t.Fatalf("executar a missao de inferencia: %v", err)
	}

	chamadas := cliente.Chamadas()
	if len(chamadas) != 2 {
		t.Fatalf("esperadas 2 chamadas, vieram %d", len(chamadas))
	}
	if chamadas[0].modelo != "local/especialista-codigo" {
		t.Fatalf("o passo de `coding` tinha de executar no especialista de codigo e executou em %q", chamadas[0].modelo)
	}
	if chamadas[1].modelo != "local/especialista-visao" {
		t.Fatalf("o passo de `vision` tinha de executar no especialista de visao e executou em %q", chamadas[1].modelo)
	}
	if chamadas[0].modelo == chamadas[1].modelo {
		t.Fatal("capacidades diferentes executaram no MESMO modelo — *o roteamento nao distinguiu, e a orquestracao " +
			"multimodelo nao aconteceu*")
	}
	roteados := modelosRoteados(t, runtime, missao.ID)
	if len(roteados) != 2 || roteados[0] != chamadas[0].modelo || roteados[1] != chamadas[1].modelo {
		t.Fatalf("a trilha registrada %v nao corresponde ao que executou (%q, %q)", roteados, chamadas[0].modelo, chamadas[1].modelo)
	}
	t.Logf("§4 MEDIDO: passo `coding` -> %s · passo `vision` -> %s", chamadas[0].modelo, chamadas[1].modelo)
}

// registroDeEspecialistasDisjuntos devolve um registro em que CADA capacidade tem um dono diferente - e' o que torna
// observavel a diferenca entre os passos.
func registroDeEspecialistasDisjuntos(t *testing.T) *multillm.Registry {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "providers.json")
	dados := `{"providers":[{"name":"local","type":"openai-compatible","base_url":"https://local.example",` +
		`"api_key_env":"LOCAL_KEY","priority":1,"models":[` +
		`{"id":"especialista-codigo","capabilities":["coding"],"cost_per_1k_input_cents":1,"cost_per_1k_output_cents":1,"quality_score":30},` +
		`{"id":"especialista-visao","capabilities":["vision"],"cost_per_1k_input_cents":9,"cost_per_1k_output_cents":9,"quality_score":40}` +
		`]}]}`
	if err := os.WriteFile(caminho, []byte(dados), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCAL_KEY", "local")
	registro, err := multillm.Load(caminho)
	if err != nil {
		t.Fatalf("carregar registro disjunto: %v", err)
	}
	return registro
}

// modeloDaChamadaDeInferencia registra o que o modelo recebeu de fato.
type modeloDaChamadaDeInferencia struct {
	modelo string
	prompt string
}

// clienteDeInferenciaFalso e' um `InferenceChatClient` que ANOTA o modelo pedido em cada chamada - *e' o unico
// jeito de medir qual modelo EXECUTOU.*
type clienteDeInferenciaFalso struct {
	mu          sync.Mutex
	chamadas    []modeloDaChamadaDeInferencia
	semConteudo bool
}

func (c *clienteDeInferenciaFalso) Chat(_ context.Context, request *api.ChatRequest, callback api.ChatResponseFunc) error {
	c.mu.Lock()
	prompt := ""
	if len(request.Messages) > 0 {
		prompt = request.Messages[len(request.Messages)-1].Content
	}
	c.chamadas = append(c.chamadas, modeloDaChamadaDeInferencia{modelo: request.Model, prompt: prompt})
	semConteudo := c.semConteudo
	c.mu.Unlock()
	conteudo := "resposta de " + request.Model
	if semConteudo {
		conteudo = ""
	}
	return callback(api.ChatResponse{Message: api.Message{Content: conteudo}})
}

func (c *clienteDeInferenciaFalso) Chamadas() []modeloDaChamadaDeInferencia {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]modeloDaChamadaDeInferencia(nil), c.chamadas...)
}

// missaoDeInferencia monta a missao com o executor de inferencia registrado e APROVA os passos que exigem
// aprovacao. *`model.infer` e' efeito externo: sem aprovacao o runtime recusa executar - e a recusa esta' certa.*
func missaoDeInferencia(t *testing.T, cliente InferenceChatClient, passos []Step, registro *multillm.Registry) (*Runtime, Mission) {
	t.Helper()
	raiz := t.TempDir()
	runtime, contexto := runtimeSobre(t, raiz)
	projeto, err := contexto.CreateProject("projeto da inferencia", filepath.Join(raiz, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	// ⚠️ A ORDEM IMPORTA (E-438): o planejador tem de ser trocado ANTES do `CreateMission`, porque e' o
	// `CreateMission` que planeja e PERSISTE o plano.
	runtime.planner = planejadorFixo{passos: passos}
	runtime.models = registro
	runtime.tools.Register(inferenceTool{cliente: cliente})

	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "provar que a inferencia usa o modelo roteado para o passo", OrganizationID: "org-a", ProjectID: projeto.ID,
		// A CONCESSÃO é explícita: *sem grant a política recusa o passo, e isso é o comportamento certo* — o
		// escopo `model:infer` não se herda por estar registado na ferramenta.
		Capabilities: []string{"model:infer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(missao.Approvals) == 0 {
		t.Fatalf("passos de inferencia (`%s`, efeito externo) TEM de gerar pedido de aprovacao", KindDeInferencia)
	}
	decidida := missao
	for _, aprovacao := range missao.Approvals {
		decidida, err = runtime.DecideApproval(missao.ID, aprovacao.ID, true, "aprovado pelo teste de inferencia por passo")
		if err != nil {
			t.Fatalf("aprovar %s: %v", aprovacao.ID, err)
		}
	}
	return runtime, decidida
}

// modelosRoteados devolve, NA ORDEM, os modelos registrados nos eventos `step.model.route`.
func modelosRoteados(t *testing.T, runtime *Runtime, missaoID string) []string {
	t.Helper()
	eventos, err := runtime.Events(missaoID)
	if err != nil {
		t.Fatalf("eventos: %v", err)
	}
	var modelos []string
	for _, evento := range eventos {
		if evento.Type != "step.model.route" {
			continue
		}
		pacote, ok := evento.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload de `step.model.route` veio como %T", evento.Payload)
		}
		if modelo, ok := pacote["model"].(string); ok {
			modelos = append(modelos, modelo)
		}
	}
	return modelos
}

func listaContem(valores []string, procurado string) bool {
	for _, valor := range valores {
		if valor == procurado {
			return true
		}
	}
	return false
}
