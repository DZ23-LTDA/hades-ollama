// Painel de traces auditáveis -- a superfície que faltava para `/traces`.
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - não inventa spans: mostra o que o servidor devolveu, ou diz que está vazio;
//   - não trata lista vazia como erro: uma org sem traces é um estado legítimo;
//   - não esconde um span falhado atrás de um total optimista;
//   - não mostra `duration_ns` cru: converte para a unidade que se lê;
//   - não apaga nada. Este painel é de leitura.
//
//	*Um painel que confia cegamente no servidor não é um painel: é um espelho.*
import { useCallback, useEffect, useState } from "react";
import {
  contarPorEstado,
  descreverEstadoDoSpan,
  estadoDoSpan,
  formatarDuracao,
  listarTraces,
  type SpanDeTrace,
} from "@/lib/tracesClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "vazia" | "erro";

const COR_DO_ESTADO: Record<ReturnType<typeof estadoDoSpan>, string> = {
  falhou: "text-red-400",
  aberto: "text-amber-300",
  concluido: "text-emerald-400",
};

export function TracesPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [spans, setSpans] = useState<SpanDeTrace[]>([]);
  const [erro, setErro] = useState<string | null>(null);
  const [filtro, setFiltro] = useState("");
  const [filtroAplicado, setFiltroAplicado] = useState<string>("");

  const carregar = useCallback(async (traceId: string) => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await listarTraces(traceId || undefined);
      const lista = resposta.spans ?? [];
      setSpans(lista);
      setEstado(lista.length === 0 ? "vazia" : "pronta");
    } catch (causa) {
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler os traces.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar("");
  }, [carregar]);

  const aplicarFiltro = useCallback(() => {
    const limpo = filtro.trim();
    setFiltroAplicado(limpo);
    void carregar(limpo);
  }, [carregar, filtro]);

  const contagem = contarPorEstado(spans);

  return (
    <section
      data-testid="traces"
      aria-labelledby="traces-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="traces-titulo" className="text-sm font-semibold text-neutral-100">
          Traces auditáveis
        </h3>
        <button
          type="button"
          data-testid="traces-actualizar"
          onClick={() => void carregar(filtroAplicado)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1 text-xs text-neutral-300">
          Filtrar por trace_id
          <input
            type="text"
            data-testid="traces-filtro"
            value={filtro}
            onChange={(evento) => setFiltro(evento.target.value)}
            placeholder="deixe vazio para todos"
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>
        <button
          type="button"
          data-testid="traces-filtrar"
          onClick={aplicarFiltro}
          className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800"
        >
          Filtrar
        </button>
      </div>

      {estado === "a-carregar" && (
        <p data-testid="traces-a-carregar" className="text-xs text-neutral-400">
          A ler os traces...
        </p>
      )}

      {estado === "erro" && erro && (
        <p data-testid="traces-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "vazia" && (
        <p data-testid="traces-vazio" className="text-xs text-neutral-400">
          {filtroAplicado
            ? `Nenhum span para o trace ${filtroAplicado}.`
            : "Esta organização ainda não tem traces registados."}
        </p>
      )}

      {estado === "pronta" && (
        <>
          <p data-testid="traces-contagem" className="mb-2 text-xs text-neutral-300">
            {contagem.total} span{contagem.total === 1 ? "" : "s"} ·{" "}
            <span className={COR_DO_ESTADO.concluido}>{contagem.concluidos} concluído(s)</span> ·{" "}
            <span className={COR_DO_ESTADO.aberto}>{contagem.abertos} aberto(s)</span> ·{" "}
            <span className={COR_DO_ESTADO.falhou}>{contagem.falhados} falhado(s)</span>
          </p>

          <table data-testid="traces-tabela" className="w-full text-left text-xs text-neutral-200">
            <thead className="text-neutral-400">
              <tr>
                <th scope="col" className="py-1 pr-2">span</th>
                <th scope="col" className="py-1 pr-2">nome</th>
                <th scope="col" className="py-1 pr-2">estado</th>
                <th scope="col" className="py-1 pr-2">duração</th>
              </tr>
            </thead>
            <tbody>
              {spans.map((span) => {
                const estadoDoSpanActual = estadoDoSpan(span);
                return (
                  <tr
                    key={span.span_id}
                    data-testid={`traces-span-${span.span_id}`}
                    className="border-t border-neutral-800"
                  >
                    <td className="py-1 pr-2 font-mono text-[11px] text-neutral-500">
                      {span.span_id.slice(0, 8)}
                    </td>
                    <td className="py-1 pr-2">{span.name}</td>
                    <td
                      data-testid={`traces-estado-${span.span_id}`}
                      className={`py-1 pr-2 ${COR_DO_ESTADO[estadoDoSpanActual]}`}
                    >
                      {descreverEstadoDoSpan(span)}
                    </td>
                    <td className="py-1 pr-2 font-mono text-[11px] text-neutral-400">
                      {estadoDoSpanActual === "aberto" ? "—" : formatarDuracao(span.duration_ns)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </>
      )}

      <p data-testid="traces-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Os traces são locais por omissão e limitados à organização autenticada. Este painel só lê: não
        apaga nem exporta. Um span sem data de fim é mostrado como <strong>aberto</strong>, não como
        concluído.
      </p>
    </section>
  );
}
