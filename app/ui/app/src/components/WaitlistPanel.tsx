// Lista de espera -- a janela PEDIDA nao e' o slot OFERECIDO, e uma oferta expira.
//
// NAO faz:
//   - NAO mostra `from`/`to` como se fosse o horario marcado;
//   - NAO aceita `offered_slot` sem `offer_expires_at` como oferta valida;
//   - NAO mostra uma oferta expirada como se ainda pudesse ser aceita.
import { useCallback, useEffect, useState } from "react";
import {
  descreverAcao,
  descreverEstado,
  descreverJanelaPedida,
  FilaAusenteError,
  listarFilaDeEspera,
  ofertaValida,
  type EntradaNaFila,
  type RespostaDeFila,
} from "@/lib/waitlistClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "ausente" | "erro";

export function WaitlistPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [r, setR] = useState<RespostaDeFila | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await listarFilaDeEspera();
      setR(resposta);
      setEstado("pronta");
    } catch (causa) {
      setR(null);
      if (causa instanceof FilaAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler a fila de espera.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const fila = r?.waitlist ?? [];
  const validas = fila.filter((e) => ofertaValida(e)).length;

  return (
    <section
      data-testid="fila"
      aria-labelledby="fila-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="fila-titulo" className="text-sm font-semibold text-neutral-100">
          Lista de espera
        </h3>
        <button
          type="button"
          data-testid="fila-atualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="fila-a-carregar" className="text-xs text-neutral-400">
          A ler a fila de espera...
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="fila-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="fila-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && (
        <>
          <p data-testid="fila-total" className="text-xs font-semibold text-neutral-200">
            {fila.length} entrada(s) na fila · {validas} com oferta válida
          </p>

          {fila.length === 0 ? (
            <p data-testid="fila-vazia" className="mt-1 text-[11px] text-neutral-400">
              A fila está vazia para esta organização.
            </p>
          ) : (
            <ul data-testid="fila-lista" className="mt-2 space-y-1 text-[11px]">
              {fila.map((e: EntradaNaFila) => {
                const valida = ofertaValida(e);
                return (
                  <li
                    key={e.id}
                    data-testid={`fila-${e.id}`}
                    className="rounded border border-neutral-800 px-2 py-1"
                  >
                    <div className="text-neutral-200">
                      <span className="font-semibold">{descreverJanelaPedida(e)}</span>
                    </div>
                    <div
                      data-testid={`fila-estado-${e.id}`}
                      className={
                        valida
                          ? "text-emerald-400"
                          : e.offered_slot
                            ? "text-amber-300"
                            : "text-neutral-400"
                      }
                    >
                      {descreverEstado(e)}
                    </div>
                    <div data-testid={`fila-acao-${e.id}`} className="text-neutral-500">
                      {descreverAcao(e)}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </>
      )}

      <p data-testid="fila-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        «De … a …» é a janela <strong>pedida</strong>, não o horário marcado. Uma oferta sem prazo
        não é uma oferta: esta lista diz quando cada uma expira.
      </p>
    </section>
  );
}
