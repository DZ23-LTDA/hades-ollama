// Gráfico da série temporal do BI (estágio 32), SEM dependência nova.
//
// Decisões que guiam este componente:
//
//  1. **SEM BIBLIOTECA**: a curva é um `<polyline>` em SVG. Isso evita decidir licença e peso de pacote
//     para desenhar onze pontos, e mantém o gráfico testável sem DOM;
//  2. **A TABELA EQUIVALENTE NÃO É OPCIONAL**: um gráfico sozinho exclui quem usa leitor de tela e quem
//     precisa do número exato. A tabela é a mesma informação em forma acessível — e continua visível,
//     não escondida atrás de um botão;
//  3. **ESCALA HONESTA**: amplitude zero (todos os valores iguais) não pode virar divisão por zero e
//     `NaN` no `polyline` — um gráfico com coordenadas `NaN` simplesmente não desenha e não avisa;
//  4. **DADO INSUFICIENTE É DITO**: com menos de duas amostras não existe curva, e o componente diz
//     isso em vez de desenhar um ponto fingindo tendência.
import { useState } from "react";
import type { BusinessSnapshot } from "@/lib/agenticClient";

// Métricas plotáveis, com rótulo em pt-BR e unidade. `dinheiro` divide por 100 na exibição.
export type TrendMetric = {
  id: string;
  label: string;
  money?: boolean;
  pick: (snapshot: BusinessSnapshot) => number;
};

export const TREND_METRICS: TrendMetric[] = [
  { id: "receita_cents", label: "Receita (pedidos)", money: true, pick: (s) => s.report.commerce.revenue_cents },
  { id: "pedidos", label: "Pedidos", pick: (s) => s.report.commerce.orders },
  { id: "liquido_cents", label: "Financeiro líquido", money: true, pick: (s) => s.report.finance.net_cents },
  { id: "receita_financeira_cents", label: "Receita financeira", money: true, pick: (s) => s.report.finance.income_cents },
  { id: "despesa_financeira_cents", label: "Despesa financeira", money: true, pick: (s) => s.report.finance.expense_cents },
  { id: "lancamentos_em_aberto", label: "Lançamentos em aberto", pick: (s) => s.report.finance.unreconciled_entries },
  { id: "campanhas_ativas", label: "Campanhas ativas", pick: (s) => s.report.campaigns.active },
  { id: "impressoes", label: "Impressões", pick: (s) => s.report.campaigns.impressions },
  { id: "propostas", label: "Propostas", pick: (s) => s.report.sales.proposals },
  { id: "follow_ups_vencidos", label: "Follow-ups vencidos", pick: (s) => s.report.sales.follow_ups_overdue },
  { id: "violacoes_sla", label: "Violações de SLA", pick: (s) => s.report.inbox.sla_violations },
];

export const TREND_CHART_WIDTH = 320;
export const TREND_CHART_HEIGHT = 96;

export type TrendPoint = { window: string; value: number };

// Série ordenada por janela. Ordena de forma defensiva: desenhar fora de ordem produziria uma curva
// com "zigue-zague" que parece oscilação real do negócio.
export function trendPoints(history: BusinessSnapshot[], metric: TrendMetric): TrendPoint[] {
  return [...(history ?? [])]
    .map((snapshot) => ({ window: snapshot.window, value: metric.pick(snapshot) }))
    .sort((left, right) => left.window.localeCompare(right.window));
}

// Coordenadas do gráfico. Amplitude ZERO (todos os valores iguais) devolve uma linha no meio da área:
// é uma linha plana — verdadeira — e nunca `NaN`, que faria o SVG sumir sem avisar.
export function trendCoordinates(points: TrendPoint[]): Array<{ x: number; y: number }> {
  if (points.length === 0) {
    return [];
  }
  const values = points.map((point) => point.value);
  const minimum = Math.min(...values);
  const maximum = Math.max(...values);
  const amplitude = maximum - minimum;
  const stepX = points.length > 1 ? TREND_CHART_WIDTH / (points.length - 1) : 0;
  return points.map((point, index) => {
    const ratio = amplitude === 0 ? 0.5 : (point.value - minimum) / amplitude;
    return {
      x: Number((index * stepX).toFixed(2)),
      // O eixo Y do SVG cresce para baixo: o maior valor fica no topo.
      y: Number((TREND_CHART_HEIGHT - ratio * TREND_CHART_HEIGHT).toFixed(2)),
    };
  });
}

// Rótulos do eixo Y (ordem de grandeza). Os VALORES EXATOS seguem na tabela: o eixo orienta, a tabela prova.
export function trendAxisLabels(points: TrendPoint[], metric: TrendMetric): { max: string; mid: string; min: string } {
  if (points.length === 0) {
    return { max: "", mid: "", min: "" };
  }
  const valores = points.map((point) => point.value);
  const maximo = Math.max(...valores);
  const minimo = Math.min(...valores);
  // Meio ARREDONDADO para inteiro: meio de 3 centavos nao existe e exibir 1.5 centavo seria precisao falsa.
  const meio = Math.round(minimo + (maximo - minimo) / 2);
  return {
    max: formatTrendValue(maximo, metric),
    mid: formatTrendValue(meio, metric),
    min: formatTrendValue(minimo, metric),
  };
}

// Seleção de intervalo: 0 significa "tudo". Fatia os ÚLTIMOS N pontos, que é o recorte que interessa na operação.
export function filterTrendPoints(history: BusinessSnapshot[], metric: TrendMetric, limite: number): TrendPoint[] {
  const pontos = trendPoints(history, metric);
  if (limite <= 0 || limite >= pontos.length) {
    return pontos;
  }
  return pontos.slice(pontos.length - limite);
}

export function formatTrendValue(value: number, metric: TrendMetric): string {
  return metric.money ? (value / 100).toFixed(2) : String(value);
}

export function BusinessTrendChart({
  history,
  metric,
  rangeLimits = [5, 10, 0],
}: {
  history: BusinessSnapshot[];
  metric: TrendMetric;
  rangeLimits?: number[];
}) {
  const [limite, setLimite] = useState(0);
  const points = filterTrendPoints(history, metric, limite);
  const coordinates = trendCoordinates(points);
  const label = `${metric.label}: ${points.length} amostra(s)`;
  const [amostraVisivel, setAmostraVisivel] = useState<number | null>(null);

  if (points.length < 2) {
    return (
      <div className="mt-2 rounded-lg border border-neutral-200 px-3 py-2 text-xs text-neutral-600 dark:border-neutral-800 dark:text-neutral-400">
        <p>
          Sem curva: são necessárias ao menos 2 amostras para desenhar tendência — há {points.length} neste recorte.
        </p>
        {points.length === 1 && (
          <p className="mt-1">
            Amostra atual: {points[0].window} = {formatTrendValue(points[0].value, metric)}
          </p>
        )}
      </div>
    );
  }

  const values = points.map((point) => point.value);
  const flat = Math.min(...values) === Math.max(...values);
  const polyline = coordinates.map((coordinate) => `${coordinate.x},${coordinate.y}`).join(" ");

  const eixo = trendAxisLabels(points, metric);

  return (
    <div className="mt-2">
      {rangeLimits.length > 1 && (
        <div className="flex flex-wrap items-center gap-1 text-[11px] text-neutral-500 dark:text-neutral-400">
          <span>Intervalo:</span>
          {rangeLimits.map((opcao) => (
            <button
              key={opcao}
              type="button"
              aria-pressed={limite === opcao}
              className="rounded border border-neutral-300 px-2 py-0.5 disabled:opacity-40 dark:border-neutral-700"
              // Opcao maior que a serie nao filtra nada: oferecer "ultimas 90" com 6 amostras seria enganoso.
              disabled={opcao > 0 && opcao > history.length}
              onClick={() => setLimite(opcao)}
            >
              {opcao === 0 ? "Tudo" : `Últimas ${opcao}`}
            </button>
          ))}
        </div>
      )}
      {/* Marcações do eixo em HTML: texto dentro de SVG com preserveAspectRatio="none" sai distorcido. */}
      <div className="mt-1 flex justify-between text-[10px] tabular-nums text-neutral-500 dark:text-neutral-400">
        <span>máx {eixo.max}</span>
        <span>méd {eixo.mid}</span>
        <span>mín {eixo.min}</span>
      </div>
      <svg
        role="img"
        aria-label={label}
        viewBox={`0 0 ${TREND_CHART_WIDTH} ${TREND_CHART_HEIGHT}`}
        className="h-24 w-full"
        preserveAspectRatio="none"
      >
        <polyline points={polyline} fill="none" stroke="currentColor" strokeWidth="2" className="text-sky-600 dark:text-sky-400" />
        {coordinates.map((coordinate, index) => (
          // FOCO, não só hover: dica que só existe no mouse exclui quem usa teclado ou leitor de tela.
          <circle
            key={points[index].window}
            cx={coordinate.x}
            cy={coordinate.y}
            r={3}
            tabIndex={0}
            className="fill-sky-700 outline-none dark:fill-sky-300"
            aria-label={`${points[index].window}: ${formatTrendValue(points[index].value, metric)}`}
            onMouseEnter={() => setAmostraVisivel(index)}
            onMouseLeave={() => setAmostraVisivel(null)}
            onFocus={() => setAmostraVisivel(index)}
            onBlur={() => setAmostraVisivel(null)}
          />
        ))}
      </svg>
      {amostraVisivel !== null && points[amostraVisivel] && (
        <p role="status" aria-live="polite" className="mt-1 text-[11px] text-neutral-600 dark:text-neutral-300">
          {points[amostraVisivel].window}: {formatTrendValue(points[amostraVisivel].value, metric)}
        </p>
      )}
      {flat && (
        // "Todos iguais" é uma informação, não um gráfico bonito: o eixo não tem amplitude.
        <p role="status" aria-live="polite" className="mt-1 text-[11px] text-neutral-500 dark:text-neutral-400">
          Todos os valores são iguais neste recorte: a linha é plana e o eixo não tem amplitude.
        </p>
      )}
      {/* A MESMA informação em forma acessível, sempre visível. */}
      <table className="mt-2 w-full text-xs">
        <caption className="sr-only">{label}</caption>
        <thead>
          <tr className="text-left text-neutral-500 dark:text-neutral-400">
            <th scope="col">Janela</th>
            <th scope="col">Valor</th>
            <th scope="col">Variação</th>
          </tr>
        </thead>
        <tbody>
          {points.map((point, index) => {
            const previous = index > 0 ? points[index - 1].value : null;
            const delta = previous === null ? null : point.value - previous;
            return (
              <tr key={point.window}>
                <td>{point.window}</td>
                <td>{formatTrendValue(point.value, metric)}</td>
                <td>{delta === null ? "—" : delta > 0 ? `+${formatTrendValue(delta, metric)}` : formatTrendValue(delta, metric)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
