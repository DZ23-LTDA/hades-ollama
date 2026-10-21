// Painel financeiro -- o livro, os recebiveis, o que falta contabilizar e as divergencias.
//
// O QUE ESTE PAINEL NAO FAZ:
//   - NAO soma `receivable` + `unposted` + `refund` e chama-lhe receita.
//   - NAO apresenta o liquido sem a cobertura de reconciliacao ao lado.
//   - NAO esconde divergencias.
//   - NAO trata financeiro ausente (503) como empresa sem numeros.
//   - NAO registra receita/custo nem reconcilia: so' le. As 3 escritas ficam de fora.
import { useCallback, useEffect, useState } from "react";
import { centavosParaReais } from "@/lib/commerceClient";
import {
  descreverDivergencia,
  descreverNaoContabilizado,
  descreverQualidadeDoLiquido,
  descreverRecebivel,
  descreverReembolso,
  FinanceiroAusenteError,
  formatarPercentagem,
  obterRelatorioFinanceiro,
  type RelatorioFinanceiro,
} from "@/lib/financeClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-financeiro" | "erro";

export function FinancePanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [relatorio, setRelatorio] = useState<RelatorioFinanceiro | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const r = await obterRelatorioFinanceiro();
      setRelatorio(r);
      setEstado("pronta");
    } catch (causa) {
      setRelatorio(null);
      if (causa instanceof FinanceiroAusenteError) {
        setErro(causa.message);
        setEstado("sem-financeiro");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler o financeiro.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const livro = relatorio?.ledger;
  const recebiveis = relatorio?.receivables ?? [];
  const divergencias = relatorio?.divergences ?? [];

  return (
    <section
      data-testid="financeiro"
      aria-labelledby="financeiro-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="financeiro-titulo" className="text-sm font-semibold text-neutral-100">
          Financeiro
        </h3>
        <button
          type="button"
          data-testid="financeiro-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "sem-financeiro" && (
        <div data-testid="financeiro-sem-financeiro" className="text-xs text-amber-300">
          <p className="font-semibold">O financeiro não está configurado para esta empresa.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «a empresa não tem números»: é «não há empresa financeira». Um relatório vazio
            aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="financeiro-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="financeiro-a-carregar" className="text-xs text-neutral-400">
          A ler o relatório financeiro...
        </p>
      )}

      {estado === "pronta" && relatorio && livro && (
        <>
          <h4 className="mb-2 text-xs font-semibold text-neutral-300">Livro</h4>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-[11px]">
            <dt className="text-neutral-500">Entradas</dt>
            <dd data-testid="financeiro-entradas" className="text-emerald-300">
              {centavosParaReais(livro.income_cents)}
            </dd>
            <dt className="text-neutral-500">Saídas</dt>
            <dd data-testid="financeiro-saidas" className="text-red-300">
              {centavosParaReais(livro.expense_cents)}
            </dd>
            <dt className="text-neutral-500">Líquido</dt>
            <dd data-testid="financeiro-liquido" className="font-semibold text-neutral-100">
              {centavosParaReais(livro.net_cents)}
            </dd>
            <dt className="text-neutral-500">Reconciliação</dt>
            <dd data-testid="financeiro-cobertura" className="text-neutral-300">
              {formatarPercentagem(livro.reconciliation_coverage_pct)}
            </dd>
          </dl>

          {/* O liquido sem a cobertura ao lado e' meia verdade. */}
          <p
            data-testid="financeiro-qualidade"
            className={
              livro.reconciliation_coverage_pct >= 100
                ? "mt-1 text-[11px] text-emerald-400"
                : "mt-1 text-[11px] text-amber-400"
            }
          >
            {descreverQualidadeDoLiquido(livro)}
          </p>

          <p data-testid="financeiro-lancamentos" className="mt-1 text-[11px] text-neutral-500">
            {livro.entry_count} lançamento(s): {livro.reconciled_count} conferido(s),{" "}
            {livro.unreconciled_count} por conferir.
          </p>

          <h4 className="mb-1 mt-4 text-xs font-semibold text-neutral-300">A receber</h4>
          <p data-testid="financeiro-recebivel" className="text-[11px] text-amber-300">
            {descreverRecebivel(relatorio.receivable_cents, recebiveis.length)}
          </p>

          {recebiveis.length > 0 && (
            <ul data-testid="financeiro-recebiveis" className="mt-1 space-y-0.5 text-[11px] text-neutral-300">
              {recebiveis.map((r) => (
                <li key={r.order_id}>
                  {r.order_number} · {centavosParaReais(r.total_cents)}
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-1 mt-4 text-xs font-semibold text-neutral-300">Por contabilizar e a devolver</h4>
          <p data-testid="financeiro-nao-contabilizado" className="text-[11px] text-neutral-300">
            {descreverNaoContabilizado(relatorio.unposted_revenue_cents)}
          </p>
          <p data-testid="financeiro-reembolso" className="text-[11px] text-neutral-300">
            {descreverReembolso(relatorio.refund_due_cents)}
          </p>

          <h4 className="mb-1 mt-4 text-xs font-semibold text-neutral-300">
            Divergências ({divergencias.length})
          </h4>

          {divergencias.length === 0 ? (
            <p data-testid="financeiro-sem-divergencias" className="text-xs text-neutral-400">
              {livro.entry_count === 0
                ? "Não há lançamentos, portanto não há divergências para mostrar."
                : "Há lançamentos e nenhuma divergência registada."}
            </p>
          ) : (
            <ul data-testid="financeiro-divergencias" className="space-y-0.5 text-[11px] text-amber-300">
              {divergencias.map((d) => (
                <li key={d.order_id + d.kind}>{descreverDivergencia(d)}</li>
              ))}
            </ul>
          )}

          <p data-testid="financeiro-pedidos" className="mt-4 text-[11px] text-neutral-500">
            {relatorio.orders_paid} pago(s), {relatorio.orders_fulfilled} cumprido(s),{" "}
            {relatorio.orders_pending} pendente(s), {relatorio.orders_cancelled} cancelado(s).
          </p>
        </>
      )}

      <p data-testid="financeiro-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê o relatório. Registar receita, registar custo e reconciliar um pedido
        continuam a ser ações separadas — e um número «a receber» nunca é somado a uma receita
        realizada.
      </p>
    </section>
  );
}
