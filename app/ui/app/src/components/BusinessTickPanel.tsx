// Ciclo de supervisão -- quatro desfechos, quatro números, um botão.
//
// NAO faz:
//   - NAO dispara um tick ao montar (e' uma ESCRITA: executa acções e gasta orçamento);
//   - NAO soma os quatro contadores num "N acções";
//   - NAO trata `errors` como ausência de resultado -- o resultado vem junto.
import { useCallback, useState } from "react";
import {
  descreverDesfechos,
  descreverErros,
  descreverOrcamento,
  descreverTick,
  dispararTick,
  precisaDeAlguem,
  resumirAcoes,
  SupervisaoAusenteError,
  type ResultadoDoTick,
} from "@/lib/businessTickClient";

type EstadoDoPainel = "pronto" | "a-correr" | "corrido" | "ausente" | "erro";

export function BusinessTickPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<ResultadoDoTick | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [confirmado, setConfirmado] = useState(false);
  const [anterior, setAnterior] = useState<number | null>(null);

  const correr = useCallback(async () => {
    if (!confirmado) {
      setErro("Confirme antes de correr um ciclo: isto executa ações e gasta orçamento.");
      setEstado("erro");
      return;
    }
    setErro(null);
    setEstado("a-correr");
    try {
      const resultado = await dispararTick();
      setAnterior(r?.tick_number ?? null);
      setR(resultado);
      setEstado("corrido");
    } catch (causa) {
      setR(null);
      if (causa instanceof SupervisaoAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível correr o ciclo.");
      setEstado("erro");
    }
  }, [confirmado, r]);

  const precisa = r ? precisaDeAlguem(r) : 0;
  const erros = r ? descreverErros(r) : null;

  return (
    <section
      data-testid="ciclo"
      aria-labelledby="ciclo-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="ciclo-titulo" className="text-sm font-semibold text-neutral-100">
          Ciclo de supervisão de negócio
        </h3>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-1 text-[11px] text-neutral-400">
            <input
              data-testid="ciclo-confirmar"
              type="checkbox"
              checked={confirmado}
              onChange={(e) => setConfirmado(e.target.checked)}
            />
            confirmo que isto executa ações
          </label>
          <button
            type="button"
            data-testid="ciclo-correr"
            onClick={() => void correr()}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Correr ciclo
          </button>
        </div>
      </header>

      {estado === "pronto" && (
        <p data-testid="ciclo-pronto" className="text-xs text-neutral-400">
          Nenhum ciclo correu nesta sessão. Isto é uma escrita: marque a confirmação para correr.
        </p>
      )}

      {estado === "a-correr" && (
        <p data-testid="ciclo-a-correr" className="text-xs text-neutral-400">
          A correr o ciclo...
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="ciclo-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="ciclo-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "corrido" && r && (
        <>
          <p data-testid="ciclo-numero" className="text-xs font-semibold text-neutral-200">
            {descreverTick(r, anterior)}
          </p>

          {/* Os quatro desfechos, separados. Nunca um "N acções". */}
          <p data-testid="ciclo-desfechos" className="mt-1 text-[11px] text-neutral-300">
            {descreverDesfechos(r)}
          </p>

          {precisa > 0 && (
            <p data-testid="ciclo-precisa" className="mt-1 text-[11px] text-amber-300">
              {precisa} ação(ões) precisam de alguém — {r.awaiting_approval ?? 0} de uma pessoa,{" "}
              {r.blocked_by_budget ?? 0} de dinheiro
            </p>
          )}

          {erros && (
            <p data-testid="ciclo-erros" className="mt-1 text-[11px] text-red-400">
              {erros}
            </p>
          )}

          <p data-testid="ciclo-orcamento" className="mt-1 text-[11px] text-neutral-500">
            {descreverOrcamento(r, null)}
          </p>

          <p data-testid="ciclo-acoes" className="mt-1 text-[11px] text-neutral-400">
            {resumirAcoes(r)}
          </p>

          {(r.errors ?? []).length > 0 && (
            <ul data-testid="ciclo-lista-erros" className="mt-1 space-y-0.5 text-[11px] text-red-400">
              {(r.errors ?? []).map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="ciclo-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        «Executada sem pessoa», «à espera de uma pessoa», «bloqueada por orçamento» e «incidente
        registado» são <strong>quatro desfechos, não um</strong>. Uma ação que precisa de dinheiro e
        uma que precisa de uma pessoa não são a mesma pendência.
      </p>
    </section>
  );
}
