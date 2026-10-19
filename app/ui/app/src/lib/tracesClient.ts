// FASE 18: traces auditáveis -- a superfície que faltava para `/traces`.
//
// Segunda das 93 rotas do E-1319 a chegar ao produto. Como o `/retention`, era
// uma rota com backend pronto, testes no servidor e nenhuma forma de a ver.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// O CONTRATO, lido do servidor (`internal/agent/traces.go`):
//
//	type TraceSpan struct {
//	    TraceID, OrganizationID, SpanID, ParentID, Name, Status
//	    StartAt time.Time; EndAt *time.Time
//	    Duration time.Duration   // JSON: duration_ns, em nanossegundos
//	    Attributes map[string]any
//	    Error string
//	}
//
// DURAÇÃO EM NANOSSEGUNDOS NÃO É DURAÇÃO PARA HUMANO. O `time.Duration` do Go
// serializa como `duration_ns` inteiro; mostrar "4820000000" ao utilizador seria
// mostrar o número certo na unidade errada.
import { agentFetch } from "@/lib/agenticClient";

/** Uma duração em ns, dita na unidade que se lê. */
export function formatarDuracao(ns: number | undefined): string {
  if (ns === undefined || ns === null || Number.isNaN(ns)) return "—";
  if (ns < 1_000) return `${ns} ns`;
  if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)} µs`;
  if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)} ms`;
  return `${(ns / 1_000_000_000).toFixed(2)} s`;
}

export interface SpanDeTrace {
  trace_id: string;
  organization_id?: string;
  span_id: string;
  parent_id?: string;
  name: string;
  status: string;
  start_at: string;
  end_at?: string;
  duration_ns?: number;
  attributes?: Record<string, unknown>;
  error?: string;
}

export interface RespostaDeTraces {
  spans: SpanDeTrace[] | null;
}

/**
 * Um span falhado diz-se falhado; um span sem `end_at` ainda está aberto.
 *
 *	*Um span aberto mostrado como concluído é um trace que mente sobre o tempo
 *	que levou.*
 */
export type EstadoDoSpan = "falhou" | "aberto" | "concluido";

export function estadoDoSpan(span: SpanDeTrace): EstadoDoSpan {
  const status = (span.status ?? "").toLowerCase();
  if (span.error || status === "error" || status === "failed" || status === "failure") {
    return "falhou";
  }
  if (!span.end_at) return "aberto";
  return "concluido";
}

export function descreverEstadoDoSpan(span: SpanDeTrace): string {
  switch (estadoDoSpan(span)) {
    case "falhou":
      return span.error ? `falhou: ${span.error}` : "falhou";
    case "aberto":
      return "aberto";
    case "concluido":
      return "concluído";
  }
}

/** Conta por estado. Devolve números reais, não uma soma que esconde. */
export function contarPorEstado(spans: SpanDeTrace[]): {
  total: number;
  falhados: number;
  abertos: number;
  concluidos: number;
} {
  let falhados = 0;
  let abertos = 0;
  let concluidos = 0;
  for (const span of spans) {
    const estado = estadoDoSpan(span);
    if (estado === "falhou") falhados += 1;
    else if (estado === "aberto") abertos += 1;
    else concluidos += 1;
  }
  return { total: spans.length, falhados, abertos, concluidos };
}

/**
 * O prefixo `/api/agent/v1` é EXPLÍCITO.
 *
 * O `scripts/verify-contracts.mjs` compara o path CRU com o path do backend; o
 * `agentFetch` acrescenta o prefixo em runtime, mas o gate não sabe disso. O
 * E-1321 custou um CI vermelho por causa disto — a armadilha está documentada no
 * `retentionClient.ts` e aplica-se a este ficheiro também.
 */
export async function listarTraces(
  traceId?: string,
  init?: RequestInit,
): Promise<RespostaDeTraces> {
  const consulta =
    traceId && traceId.trim() !== ""
      ? `?trace_id=${encodeURIComponent(traceId.trim())}`
      : "";
  return agentFetch<RespostaDeTraces>(`/api/agent/v1/traces${consulta}`, init);
}
