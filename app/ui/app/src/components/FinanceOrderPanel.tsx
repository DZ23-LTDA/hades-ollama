// Lançamentos financeiros de uma ordem -- três escritas, três regras.
//
// NAO faz:
//   - NAO lança ao montar (são três ESCRITAS);
//   - NAO mostra a receita como "lançada" quando o servidor a recusou (receita fictícia);
//   - NAO assume 201 para as três: `reconcile` devolve 200 e não cria nada;
//   - NAO esconde o factor 100: o montante aparece formatado em reais ao lado dos centavos.
import { useCallback, useState } from "react";
import {
  descreverFronteiraDaReceita,
  descreverLancamento,
  descreverOperacao,
  FinanceiroAusenteError,
  formatarCentavos,
  lancar,
  LancamentoRecusadoError,
  validarCusto,
  validarMontante,
  type LancamentoFinanceiro,
  type OperacaoFinanceira,
} from "@/lib/financeOrderClient";

type EstadoDoPainel = "pronto" | "a-lancar" | "lancado" | "recusado" | "ausente" | "erro";

export function FinanceOrderPanel() {
  const [ordem, setOrdem] = useState("");
  const [centavos, setCentavos] = useState("");
  const [categoria, setCategoria] = useState("");
  const [referencia, setReferencia] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<LancamentoFinanceiro | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [op, setOp] = useState<OperacaoFinanceira>("revenue");

  const executar = useCallback(
    async (operacao: OperacaoFinanceira) => {
      setOp(operacao);

      if (!confirmado) {
        setErro("Confirme antes de lançar: isto escreve no financeiro da empresa.");
        setEstado("erro");
        return;
      }

      if (operacao === "cost") {
        const n = Number(centavos);
        const erroM = validarMontante(Number.isFinite(n) ? n : undefined);
        if (erroM) {
          setErro(erroM);
          setEstado("recusado");
          return;
        }
        const erroC = validarCusto(categoria, referencia);
        if (erroC) {
          setErro(erroC);
          setEstado("recusado");
          return;
        }
      }

      setErro(null);
      setEstado("a-lancar");
      try {
        const resposta = await lancar({
          operacao,
          orderId: ordem,
          amountCents: operacao === "cost" ? Number(centavos) : undefined,
          category: categoria,
          reference: referencia,
        });
        setR(resposta);
        setEstado("lancado");
      } catch (causa) {
        setR(null);
        if (causa instanceof FinanceiroAusenteError) {
          setErro(causa.message);
          setEstado("ausente");
          return;
        }
        if (causa instanceof LancamentoRecusadoError) {
          setErro(causa.message);
          setEstado("recusado");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível lançar.");
        setEstado("erro");
      }
    },
    [ordem, centavos, categoria, referencia, confirmado],
  );

  const centavosNum = Number(centavos);

  return (
    <section
      data-testid="financeiro"
      aria-labelledby="financeiro-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="financeiro-titulo" className="text-sm font-semibold text-neutral-100">
          Lançamentos financeiros
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="financeiro-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto escreve no financeiro
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="financeiro-ordem"
          value={ordem}
          onChange={(e) => setOrdem(e.target.value)}
          placeholder="id da ordem"
          aria-label="Identificador da ordem"
          className="w-36 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="financeiro-centavos"
          value={centavos}
          onChange={(e) => setCentavos(e.target.value)}
          placeholder="amount_cents"
          aria-label="Montante em centavos"
          inputMode="numeric"
          className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="financeiro-categoria"
          value={categoria}
          onChange={(e) => setCategoria(e.target.value)}
          placeholder="categoria"
          aria-label="Categoria"
          className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="financeiro-referencia"
          value={referencia}
          onChange={(e) => setReferencia(e.target.value)}
          placeholder="referência"
          aria-label="Referência"
          className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="financeiro-receita"
          onClick={() => void executar("revenue")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Lançar receita
        </button>
        <button
          type="button"
          data-testid="financeiro-custo"
          onClick={() => void executar("cost")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Lançar custo
        </button>
        <button
          type="button"
          data-testid="financeiro-reconciliar"
          onClick={() => void executar("reconcile")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Reconciliar extrato
        </button>
      </div>

      {/* O factor 100, visivel ANTES do erro: centavos ao lado de reais. */}
      <p data-testid="financeiro-montante" className="mb-2 text-[11px] text-neutral-500">
        {Number.isFinite(centavosNum) && centavos.trim() !== ""
          ? `${centavos} centavos = ${formatarCentavos(centavosNum)}`
          : "montante em CENTAVOS, não em reais"}
      </p>

      <p data-testid="financeiro-operacao" className="mb-2 text-[11px] text-neutral-500">
        {descreverOperacao(op)}
      </p>

      {estado === "pronto" && (
        <p data-testid="financeiro-pronto" className="text-xs text-neutral-400">
          Nada lançado nesta sessão. São três escritas com regras diferentes — confirme para lançar.
        </p>
      )}

      {estado === "a-lancar" && (
        <p data-testid="financeiro-a-lancar" className="text-xs text-neutral-400">
          A lançar...
        </p>
      )}

      {/* A recusa vem do SERVIDOR: o painel nao sabe se o pedido foi pago. */}
      {estado === "recusado" && (
        <p data-testid="financeiro-recusado" className="text-xs text-amber-300">
          O servidor recusou o lançamento: {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="financeiro-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="financeiro-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "lancado" && (
        <>
          <p data-testid="financeiro-resultado" className="text-xs font-semibold text-emerald-300">
            lançamento registado · {descreverOperacao(op)}
          </p>
          <p data-testid="financeiro-lancamento" className="mt-1 text-[11px] text-neutral-300">
            {descreverLancamento(r)}
          </p>
        </>
      )}

      <p data-testid="financeiro-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverFronteiraDaReceita()} E as três não devolvem o mesmo status: <code>cost</code> é 201,{" "}
        <code>reconcile</code> é 200 e não cria nada.
      </p>
    </section>
  );
}
