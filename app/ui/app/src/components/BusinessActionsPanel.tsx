// Painel de acoes de negocio -- o que espera decisao humana, e PORQUE espera.
//
// O QUE ESTE PAINEL NAO FAZ:
//   - NAO trata "bloqueada por orcamento" como "a espera de aprovacao";
//   - NAO mostra uma acao sem o custo ao lado;
//   - NAO decide: aprovar EXECUTA e e' uma escrita que este painel nao faz;
//   - NAO trata supervisor ausente (503) como "nenhuma acao pendente".
import { useCallback, useEffect, useState } from "react";
import {
  aguardaHumano,
  bloqueadaPorOrcamento,
  descreverCusto,
  descreverEstadoDaAcao,
  descreverRisco,
  descreverTipo,
  listarAcoesDeNegocio,
  NegocioAusenteError,
  resumirPendentes,
  type AcaoDeNegocio,
} from "@/lib/businessClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-supervisor" | "erro";

export function BusinessActionsPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [acoes, setAcoes] = useState<AcaoDeNegocio[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const r = await listarAcoesDeNegocio();
      setAcoes(r.pending ?? []);
      setEstado("pronta");
    } catch (causa) {
      setAcoes([]);
      if (causa instanceof NegocioAusenteError) {
        setErro(causa.message);
        setEstado("sem-supervisor");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler as acoes.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  return (
    <section
      data-testid="negocio"
      aria-labelledby="negocio-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="negocio-titulo" className="text-sm font-semibold text-neutral-100">
          Ações de negócio
        </h3>
        <button
          type="button"
          data-testid="negocio-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "sem-supervisor" && (
        <div data-testid="negocio-sem-supervisor" className="text-xs text-amber-300">
          <p className="font-semibold">O supervisor de negócio não está ligado neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há ações pendentes»: é «não há supervisor». Uma lista vazia aqui
            mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="negocio-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="negocio-a-carregar" className="text-xs text-neutral-400">
          A ler as ações...
        </p>
      )}

      {estado === "pronta" && (
        <>
          {/* Orçamento e aprovação são pendências diferentes. */}
          <p data-testid="negocio-resumo" className="text-[11px] text-neutral-300">
            {resumirPendentes(acoes)}
          </p>

          {acoes.length === 0 ? (
            <p data-testid="negocio-sem-acoes" className="mt-2 text-xs text-neutral-400">
              O supervisor está ligado e não há ações à espera de decisão.
            </p>
          ) : (
            <ul data-testid="negocio-acoes" className="mt-2 space-y-1">
              {acoes.map((a) => (
                <li
                  key={a.id}
                  data-testid={`negocio-acao-${a.id}`}
                  className={
                    bloqueadaPorOrcamento(a)
                      ? "rounded border border-amber-800 px-2 py-1 text-[11px]"
                      : aguardaHumano(a)
                        ? "rounded border border-neutral-700 px-2 py-1 text-[11px]"
                        : "rounded border border-neutral-800 px-2 py-1 text-[11px]"
                  }
                >
                  <span className="font-semibold text-neutral-200">{descreverTipo(a)}</span>{" "}
                  <span className="text-neutral-300">{a.subject}</span>
                  <span
                    data-testid={`negocio-estado-${a.id}`}
                    className={
                      bloqueadaPorOrcamento(a)
                        ? "block text-amber-400"
                        : "block text-neutral-400"
                    }
                  >
                    {descreverEstadoDaAcao(a.status)}
                  </span>
                  {/* O custo ao lado, sempre. */}
                  <span data-testid={`negocio-custo-${a.id}`} className="block text-neutral-400">
                    {descreverCusto(a.cost_cents)} · {descreverRisco(a)}
                  </span>
                  <span className="block text-neutral-500">{a.reason}</span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="negocio-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê. Aprovar e recusar continuam a ser ações separadas — e aprovar{" "}
        <em>executa</em>. Uma ação bloqueada por orçamento não se resolve com uma aprovação.
      </p>
    </section>
  );
}
