package agent

// FASE 21 do adendo V3 — TELEMETRIA DE QUALIDADE, LATENCIA, CUSTO E RECURSOS.
//
// O QUE JA' EXISTIA, e o que NAO existia: `telemetry.go` e' OpenTelemetry -- tracing
// distribuido, spans, exportacao OTLP. Isso responde "onde o tempo foi gasto numa chamada";
// NAO responde "este modelo e' melhor por tarefa", "qual o p99", "quanto custou de facto" nem
// "quanto consumiu de energia". *Um trace nao e' um agregado: e' um registo pontual.*
//
// Estas funcoes sao PURAS e nao tem estado. Agregam amostras que o runtime ja' produz.
//
// REGRA CENTRAL DO ADENDO, aplicada em cada campo:
//
//   *Nunca invente energia ou custo quando a plataforma nao fornecer medida confiavel;
//   marque NAO_MEDIDO e declare o metodo.*
//
// Por isso os campos que podem nao ter sido medidos sao PONTEIROS: `nil` e' a unica forma, no
// tipo, de dizer "nao medido" sem o confundir com "mediu zero". Um `int64` zero e' uma medida
// legitima (consumiu 0 mJ porque nao correu); `nil` e' a ausencia de medida.

import (
	"encoding/json"
	"sort"
	"strings"
)

// MetodoDosPercentis declara COMO o percentil foi calculado. *Um numero sem metodo e' um numero
// que nao se pode comparar com outro.*
const MetodoDosPercentis = "nearest-rank sobre amostras bem-sucedidas (sem interpolacao)"

// AmostraDeInferencia e' uma medicao. Zero nos campos opcionais significa NAO MEDIDO -- e' por
// isso que o resumo os converte em ponteiros.
type AmostraDeInferencia struct {
	ModelID  string `json:"model_id"`
	Provider string `json:"provider"`

	// LatenciaMS e' o tempo total da chamada. SEMPRE medido.
	LatenciaMS int64 `json:"latency_ms"`

	// TTFTMS e' o time to first token. ZERO = NAO MEDIDO (nem toda a chamada e' streaming).
	TTFTMS int64 `json:"ttft_ms,omitempty"`

	TokensEntrada int64 `json:"tokens_input"`
	TokensSaida   int64 `json:"tokens_output"`

	// Custo em centavos. PREVISTO antes de chamar, REALIZADO depois. *Separa-los e' o que
	// permite detectar que o previsto mentiu.*
	CustoPrevistoCentavos  int64 `json:"cost_forecast_cents"`
	CustoRealizadoCentavos int64 `json:"cost_actual_cents"`

	// EnergiaMilijoules e VRAMMiB: ZERO = NAO MEDIDO.
	EnergiaMilijoules int64 `json:"energy_millijoules,omitempty"`
	VRAMMiB           int64 `json:"vram_mib,omitempty"`

	Sucesso     bool  `json:"success"`
	Cancelada   bool  `json:"cancelled"`
	Retries     int   `json:"retries"`
	DuracaoFila int64 `json:"queue_ms,omitempty"`
}

// FaixaDeLatencia agrupa o que se mede sempre.
type FaixaDeLatencia struct {
	Media int64 `json:"mean_ms"`
	P50   int64 `json:"p50_ms"`
	P95   int64 `json:"p95_ms"`
	P99   int64 `json:"p99_ms"`
	Max   int64 `json:"max_ms"`
}

// ResumoDeMetricas e' o agregado. Todo campo que pode nao ter sido medido e' ponteiro ou esta'
// listado em EixosNaoMedidos.
type ResumoDeMetricas struct {
	Amostras int `json:"samples"`

	Latencia FaixaDeLatencia `json:"latency"`

	// TTFT nil = nenhuma amostra mediu time to first token.
	TTFT *FaixaDeLatencia `json:"ttft,omitempty"`

	// TokensPorSegundo usa a soma de tokens de saida sobre a soma do tempo -- nao a media de
	// medias. *A media de medias da' peso igual a uma chamada de 10 ms e a uma de 10 s.*
	TokensPorSegundo float64 `json:"tokens_per_second"`
	TokensEntrada    int64   `json:"tokens_input"`
	TokensSaida      int64   `json:"tokens_output"`

	Sucessos      int `json:"successes"`
	Falhas        int `json:"failures"`
	Cancelamentos int `json:"cancellations"`
	Retries       int `json:"retries"`

	// TaxaDeSucesso e' sobre as NAO CANCELADAS: *contar um cancelamento pedido pelo utilizador
	// como falha do modelo seria culpar o modelo por uma decisao humana.*
	TaxaDeSucesso float64 `json:"success_rate"`

	CustoPrevistoCentavos  int64 `json:"cost_forecast_cents"`
	CustoRealizadoCentavos int64 `json:"cost_actual_cents"`
	// DesvioDeCustoCentavos = realizado - previsto. Positivo = gastou mais do que previu.
	DesvioDeCustoCentavos int64 `json:"cost_deviation_cents"`

	// EnergiaMilijoules nil = NAO MEDIDO. VRAMMiB nil = NAO MEDIDO.
	EnergiaMilijoules *int64 `json:"energy_millijoules,omitempty"`
	VRAMMiB           *int64 `json:"vram_mib,omitempty"`

	// EixosNaoMedidos DECLARA o que falta. *Uma omissao silenciosa le-se como zero.*
	EixosNaoMedidos []string `json:"axes_not_measured"`
	MetodoPercentis string   `json:"percentiles_method"`
}

// Percentil nearest-rank: o menor valor cuja posicao cobre p% da amostra. Sem interpolacao,
// porque interpolar inventaria um valor que nenhuma medicao produziu.
//
// `valores` tem de estar ORDENADO. p em 1..100.
func Percentil(valores []int64, p int) int64 {
	if len(valores) == 0 {
		return 0
	}
	if p <= 0 {
		return valores[0]
	}
	if p >= 100 {
		return valores[len(valores)-1]
	}
	// nearest-rank: ceil(p/100 * n) - 1
	indice := (p*len(valores)+99)/100 - 1
	if indice < 0 {
		indice = 0
	}
	if indice >= len(valores) {
		indice = len(valores) - 1
	}
	return valores[indice]
}

// faixaDe calcula a faixa para um conjunto nao vazio (a lista e' copiada e ordenada).
func faixaDe(valores []int64) FaixaDeLatencia {
	if len(valores) == 0 {
		return FaixaDeLatencia{}
	}
	ordenado := make([]int64, len(valores))
	copy(ordenado, valores)
	sort.Slice(ordenado, func(i, j int) bool { return ordenado[i] < ordenado[j] })
	soma := int64(0)
	for _, v := range ordenado {
		soma += v
	}
	return FaixaDeLatencia{
		Media: soma / int64(len(ordenado)),
		P50:   Percentil(ordenado, 50),
		P95:   Percentil(ordenado, 95),
		P99:   Percentil(ordenado, 99),
		Max:   ordenado[len(ordenado)-1],
	}
}

// ResumirMetricas agrega as amostras.
//
// A lista vazia produz um resumo com TODOS os eixos declarados NAO MEDIDOS. *Devolver zeros
// sem os declarar seria pior do que devolver nada: pareceria que se mediu e deu zero.*
func ResumirMetricas(amostras []AmostraDeInferencia) ResumoDeMetricas {
	resumo := ResumoDeMetricas{
		Amostras:        len(amostras),
		MetodoPercentis: MetodoDosPercentis,
	}
	if len(amostras) == 0 {
		resumo.EixosNaoMedidos = []string{"latencia", "ttft", "tokens", "custo", "energia", "vram"}
		return resumo
	}

	// As faixas de latencia e TTFT usam so' as amostras BEM-SUCEDIDAS: *misturar a latencia de
	// um timeout com a de uma chamada boa descreveria uma chamada que nao existe.*
	latencias := make([]int64, 0, len(amostras))
	ttfts := make([]int64, 0, len(amostras))
	energiaMedida := false
	vramMedida := false
	energia := int64(0)
	vram := int64(0)
	comTTFT := false

	for _, amostra := range amostras {
		resumo.TokensEntrada += amostra.TokensEntrada
		resumo.TokensSaida += amostra.TokensSaida
		resumo.Retries += amostra.Retries
		resumo.CustoPrevistoCentavos += amostra.CustoPrevistoCentavos
		resumo.CustoRealizadoCentavos += amostra.CustoRealizadoCentavos
		switch {
		case amostra.Cancelada:
			resumo.Cancelamentos++
		case amostra.Sucesso:
			resumo.Sucessos++
		default:
			resumo.Falhas++
		}
		if amostra.EnergiaMilijoules > 0 {
			energiaMedida = true
			energia += amostra.EnergiaMilijoules
		}
		if amostra.VRAMMiB > 0 {
			vramMedida = true
			if amostra.VRAMMiB > vram {
				vram = amostra.VRAMMiB
			}
		}
		if !amostra.Sucesso || amostra.Cancelada {
			continue
		}
		latencias = append(latencias, amostra.LatenciaMS)
		if amostra.TTFTMS > 0 {
			comTTFT = true
			ttfts = append(ttfts, amostra.TTFTMS)
		}
	}

	resumo.Latencia = faixaDe(latencias)
	if comTTFT {
		faixa := faixaDe(ttfts)
		resumo.TTFT = &faixa
	}
	resumo.DesvioDeCustoCentavos = resumo.CustoRealizadoCentavos - resumo.CustoPrevistoCentavos

	// Taxa de sucesso sobre as NAO canceladas.
	avaliaveis := resumo.Sucessos + resumo.Falhas
	if avaliaveis > 0 {
		resumo.TaxaDeSucesso = float64(resumo.Sucessos) / float64(avaliaveis)
	}

	// Tokens/segundo: soma de tokens sobre soma de tempo (nao media de medias).
	segundosTotais := int64(0)
	for _, amostra := range latencias {
		segundosTotais += amostra
	}
	if segundosTotais > 0 {
		resumo.TokensPorSegundo = float64(resumo.TokensSaida) / (float64(segundosTotais) / 1000.0)
	}

	if energiaMedida {
		total := energia
		resumo.EnergiaMilijoules = &total
	}
	if vramMedida {
		pico := vram
		resumo.VRAMMiB = &pico
	}

	// DECLARAR o que falta -- e' isto que impede uma omissao de ser lida como zero.
	naoMedidos := make([]string, 0, 6)
	if len(latencias) == 0 {
		naoMedidos = append(naoMedidos, "latencia")
	}
	if resumo.TTFT == nil {
		naoMedidos = append(naoMedidos, "ttft")
	}
	if resumo.TokensSaida == 0 {
		naoMedidos = append(naoMedidos, "tokens")
	}
	if resumo.CustoRealizadoCentavos == 0 && resumo.CustoPrevistoCentavos == 0 {
		naoMedidos = append(naoMedidos, "custo")
	}
	if resumo.EnergiaMilijoules == nil {
		naoMedidos = append(naoMedidos, "energia")
	}
	if resumo.VRAMMiB == nil {
		naoMedidos = append(naoMedidos, "vram")
	}
	resumo.EixosNaoMedidos = naoMedidos

	return resumo
}

// ComparacaoPorModelo agrega por modelo e devolve a lista ordenada por latencia p95 crescente.
// *Ordenar por media esconderia a cauda, e a cauda e' o que o utilizador sente.*
func ComparacaoPorModelo(amostras []AmostraDeInferencia) []ResumoPorModelo {
	porModelo := map[string][]AmostraDeInferencia{}
	for _, amostra := range amostras {
		chave := amostra.Provider + "/" + amostra.ModelID
		porModelo[chave] = append(porModelo[chave], amostra)
	}
	saida := make([]ResumoPorModelo, 0, len(porModelo))
	for chave, conjunto := range porModelo {
		saida = append(saida, ResumoPorModelo{Modelo: chave, Resumo: ResumirMetricas(conjunto)})
	}
	// Ordenacao determinista: p95, depois nome. *Sem o segundo criterio, dois modelos com o
	// mesmo p95 trocariam de lugar entre execucoes e o relatorio nao seria reproduzivel.*
	sort.Slice(saida, func(i, j int) bool {
		if saida[i].Resumo.Latencia.P95 != saida[j].Resumo.Latencia.P95 {
			return saida[i].Resumo.Latencia.P95 < saida[j].Resumo.Latencia.P95
		}
		return saida[i].Modelo < saida[j].Modelo
	})
	return saida
}

// ResumoPorModelo e' uma linha da comparacao.
type ResumoPorModelo struct {
	Modelo string           `json:"model"`
	Resumo ResumoDeMetricas `json:"summary"`
}

// ── FASE 21: colheita a partir dos eventos REAIS ────────────────────────────────
//
// `ResumirMetricas` agrega amostras; esta parte COLHE as amostras dos eventos que o runtime
// ja' produz. Sem esta ponte, a FASE 21 seria uma funcao pura bonita a agregar zeros.
//
// A DURACAO VEM DO EVENTO, nao de um relogio refeito aqui: `step.succeeded` e `step.failed`
// passam a trazer `duration_ms` medido em volta do `tool.Execute` (runtime.go). *Refazer a
// conta aqui seria a segunda verdade sobre o mesmo tempo.*
//
// O QUE OS EVENTOS **NAO** TRAZEM, e que por isso continua declarado NAO MEDIDO:
//   tokens de entrada/saida, custo realizado, energia, VRAM.
// O `step.model.route` traz o custo ESTIMADO por 1k tokens -- que nao e' custo realizado e
// nao se pode somar como se fosse. *Somar uma estimativa por 1k sem os tokens medidos daria um
// numero com cara de factura e sem factura por tras.*

// LeitorDeEventos e' injectavel: em producao e' o store; num teste e' uma funcao.
type LeitorDeEventos func(missionID string) ([]Event, error)

// MetricasDasMissoes colhe `step.succeeded` e `step.failed` das missoes e agrega.
//
// Erros de leitura NAO abortam a colheita: uma missao ilegivel nao pode apagar as metricas das
// outras. O numero de missoes ilegiveis e' DECLARADO no resumo -- *uma falha silenciosa de
// leitura faria o relatorio parecer mais completo do que e'.*
func MetricasDasMissoes(missoes []Mission, leitor LeitorDeEventos) (ResumoDeMetricas, int) {
	amostras := make([]AmostraDeInferencia, 0, len(missoes)*4)
	ilegiveis := 0
	for _, missao := range missoes {
		eventos, err := leitor(missao.ID)
		if err != nil {
			ilegiveis++
			continue
		}
		amostras = append(amostras, amostrasDosEventos(eventos)...)
	}
	return ResumirMetricas(amostras), ilegiveis
}

// amostrasDosEventos converte os eventos de UMA missao em amostras. Extraida para que a
// colheita e a comparacao por modelo usem a MESMA conversao.
func amostrasDosEventos(eventos []Event) []AmostraDeInferencia {
	modelo := map[string]string{} // step_id -> "provider/model"
	for _, evento := range eventos {
		if evento.Type != "step.model.route" {
			continue
		}
		pacote, ok := evento.Payload.(map[string]any)
		if !ok {
			continue
		}
		if nome, ok := pacote["model"].(string); ok && nome != "" {
			modelo[evento.StepID] = nome
		}
	}
	amostras := make([]AmostraDeInferencia, 0, 4)
	for _, evento := range eventos {
		if evento.Type != "step.succeeded" && evento.Type != "step.failed" {
			continue
		}
		pacote, _ := evento.Payload.(map[string]any)
		duracao := int64(0)
		if pacote != nil {
			if valor, ok := numeroComoInt64(pacote["duration_ms"]); ok {
				duracao = valor
			}
		}
		nome := modelo[evento.StepID]
		if nome == "" {
			// Sem decisao de modelo registada o passo ainda e' uma amostra: a duracao foi
			// medida. Marcar o modelo como vazio e' declarar a lacuna.
			nome = "nao-roteado"
		}
		amostra := AmostraDeInferencia{
			ModelID:    nome,
			Provider:   nome,
			LatenciaMS: duracao,
			Sucesso:    evento.Type == "step.succeeded",
		}
		if evento.Type == "step.failed" && pacote != nil {
			if texto, ok := pacote["error"].(string); ok {
				amostra.Cancelada = contemCancelamento(texto)
			}
		}
		amostras = append(amostras, amostra)
	}
	return amostras
}

// numeroComoInt64 aceita os tipos que o JSON/persistencia podem devolver, sem inventar valor.
func numeroComoInt64(valor any) (int64, bool) {
	switch numero := valor.(type) {
	case float64:
		return int64(numero), true
	case int:
		return int64(numero), true
	case int64:
		return numero, true
	case json.Number:
		if convertido, err := numero.Int64(); err == nil {
			return convertido, true
		}
	}
	return 0, false
}

// contemCancelamento reconhece o cancelamento como CAUSA, para nao o contar como falha do
// modelo. *Culpar o modelo por uma decisao humana e' o erro que o resumo evita.*
func contemCancelamento(texto string) bool {
	minusculo := strings.ToLower(texto)
	return strings.Contains(minusculo, "cancel")
}

// ComparacaoPorModeloAmostras junta a colheita e a comparacao por modelo num unico passo, para
// o chamador nao ter de colher duas vezes. *Duas colheitas sobre os mesmos eventos dariam dois
// agregados que podem divergir se um evento novo chegar no meio.*
func ComparacaoPorModeloAmostras(missoes []Mission, leitor LeitorDeEventos) []ResumoPorModelo {
	amostras := make([]AmostraDeInferencia, 0, len(missoes)*4)
	for _, missao := range missoes {
		eventos, err := leitor(missao.ID)
		if err != nil {
			continue
		}
		amostras = append(amostras, amostrasDosEventos(eventos)...)
	}
	return ComparacaoPorModelo(amostras)
}
