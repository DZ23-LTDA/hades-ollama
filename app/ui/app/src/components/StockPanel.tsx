// Stock por SKU -- os tres numeros, o sinal do movimento, e a soma conferida.
//
// NAO faz:
//   - NAO mostra `on_hand` como se fosse `available`;
//   - NAO esconde o que esta' reservado;
//   - NAO aceita um saldo que nao bata com os movimentos que vieram no mesmo payload.
import { useCallback, useEffect, useState } from "react";
import {
  conferirDerivacao,
  conferirDisponibilidade,
  descreverIndisponivel,
  descreverMovimento,
  descreverSaldo,
  obterStock,
  StockAusenteError,
  type RespostaDeStock,
} from "@/lib/stockClient";

type EstadoDoPainel = "pronto" | "a-carregar" | "pronta" | "ausente" | "erro";

export function StockPanel() {
  const [sku, setSku] = useState("");
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeStock | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const consultar = useCallback(async (alvo: string) => {
    const limpo = alvo.trim();
    if (limpo === "") {
      setR(null);
      setErro("Indique um SKU para consultar o stock.");
      setEstado("ausente");
      return;
    }
    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await obterStock(limpo);
      setR(resposta);
      setEstado("pronta");
    } catch (causa) {
      setR(null);
      if (causa instanceof StockAusenteError) {
        // Mensagem propria: a do servidor pode ser um codigo interno. O painel diz o que
        // aconteceu em portugues, e nao repete o texto da maquina.
        setErro(
          causa.message.includes("sem SKU")
            ? "Indique um SKU para consultar o stock."
            : "O armazém de comércio não está ligado neste runtime.",
        );
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel consultar o stock.");
      setEstado("erro");
    }
  }, []);

  // Arranca a pedir o SKU: este painel nao tem nada a mostrar sem input.
  useEffect(() => {
    void consultar("");
  }, [consultar]);

  const saldo = r?.balance ?? null;
  const movimentos = r?.movements ?? [];
  const indisponivel = descreverIndisponivel(saldo);
  const derivacao = r ? conferirDerivacao(r) : null;
  const disponibilidade = saldo ? conferirDisponibilidade(saldo) : null;

  return (
    <section
      data-testid="stock"
      aria-labelledby="stock-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="stock-titulo" className="text-sm font-semibold text-neutral-100">
          Stock por SKU
        </h3>
        <div className="flex items-center gap-2">
          <input
            data-testid="stock-sku"
            value={sku}
            onChange={(e) => setSku(e.target.value)}
            placeholder="SKU"
            aria-label="SKU a consultar"
            className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <button
            type="button"
            data-testid="stock-consultar"
            onClick={() => void consultar(sku)}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Consultar
          </button>
        </div>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="stock-a-carregar" className="text-xs text-neutral-400">
          A consultar o stock...
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="stock-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="stock-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && (
        <>
          {/* Os tres numeros, nunca um. */}
          <p data-testid="stock-saldo" className="text-xs font-semibold text-neutral-200">
            {descreverSaldo(saldo)}
          </p>

          {/* O que existe mas ja' tem dono. */}
          {indisponivel && (
            <p data-testid="stock-indisponivel" className="mt-1 text-[11px] text-amber-300">
              {indisponivel}
            </p>
          )}

          {/* O saldo e' derivado: a soma e' conferida. */}
          {derivacao && (
            <p
              data-testid="stock-derivacao"
              className={
                derivacao.startsWith("saldo conferido")
                  ? "mt-1 text-[11px] text-emerald-400"
                  : "mt-1 text-[11px] text-red-400"
              }
            >
              {derivacao}
            </p>
          )}

          {disponibilidade && (
            <p data-testid="stock-disponibilidade" className="mt-1 text-[11px] text-red-400">
              {disponibilidade}
            </p>
          )}

          <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
            Movimentos ({movimentos.length})
          </h4>

          {movimentos.length === 0 ? (
            <p data-testid="stock-sem-movimentos" className="text-[11px] text-neutral-500">
              Sem movimentos registados para este SKU.
            </p>
          ) : (
            <ul data-testid="stock-movimentos" className="space-y-0.5 text-[11px]">
              {movimentos.map((m) => (
                <li
                  key={m.id}
                  data-testid={`stock-mov-${m.id}`}
                  className={(m.delta ?? 0) >= 0 ? "text-emerald-400" : "text-neutral-400"}
                >
                  <span className="font-semibold">{descreverMovimento(m)}</span> · {m.kind}
                  {m.reference ? ` · ${m.reference}` : ""}
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="stock-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        O saldo é <strong>derivado</strong> dos movimentos, não guardado. «Em armazém» não é
        «disponível»: o que já tem dono aparece separado.
      </p>
    </section>
  );
}
