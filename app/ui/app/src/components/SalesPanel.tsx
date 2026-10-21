// Painel de vendas -- propostas com a conta ABERTA, itens e pos-venda atrasado.
//
// O QUE ESTE PAINEL NAO FAZ, DE PROPOSITO:
//   - NAO mostra so' o total: mostra subtotal, desconto, imposto, frete e total.
//     *Uma proposta que mostra so' o total nao mostra a proposta: mostra o preco.*
//   - NAO escreve R$ 0,00 para frete nao declarado: escreve «frete nao declarado».
//   - NAO mostra centavos como reais.
//   - NAO trata vendas ausentes (503) como lista vazia.
//   - NAO cria proposta, envia, decide nem completa acompanhamento: so' le.
import { useCallback, useEffect, useState } from "react";
import {
  abrirAConta,
  contarUnidades,
  descreverEstadoDaProposta,
  descreverValorDaProposta,
  listarAcompanhamentos,
  listarPropostas,
  obterProposta,
  VendasAusenteError,
  type AcompanhamentoDeVenda,
  type PropostaDeVenda,
} from "@/lib/salesClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-vendas" | "erro";

export function SalesPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [propostas, setPropostas] = useState<PropostaDeVenda[]>([]);
  const [acompanhamentos, setAcompanhamentos] = useState<AcompanhamentoDeVenda[]>([]);
  const [filtro, setFiltro] = useState("");
  const [soAtrasados, setSoAtrasados] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [detalhe, setDetalhe] = useState<PropostaDeVenda | null>(null);
  const [detalheErro, setDetalheErro] = useState<string | null>(null);

  const carregar = useCallback(async (filtroStatus: string, atrasados: boolean) => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rPropostas, rAcompanhamentos] = await Promise.all([
        listarPropostas(filtroStatus),
        listarAcompanhamentos(atrasados),
      ]);
      setPropostas(rPropostas.proposals ?? []);
      setAcompanhamentos(rAcompanhamentos.follow_ups ?? []);
      setEstado("pronta");
    } catch (causa) {
      setPropostas([]);
      setAcompanhamentos([]);
      if (causa instanceof VendasAusenteError) {
        setErro(causa.message);
        setEstado("sem-vendas");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler as vendas.");
      setEstado("erro");
    }
  }, []);

  const abrir = useCallback(async (id: string) => {
    setDetalheErro(null);
    try {
      setDetalhe(await obterProposta(id));
    } catch (causa) {
      setDetalhe(null);
      setDetalheErro(causa instanceof Error ? causa.message : "Nao foi possivel ler a proposta.");
    }
  }, []);

  useEffect(() => {
    void carregar("", false);
  }, [carregar]);

  return (
    <section
      data-testid="vendas"
      aria-labelledby="vendas-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="vendas-titulo" className="text-sm font-semibold text-neutral-100">
          Vendas
        </h3>
        <button
          type="button"
          data-testid="vendas-actualizar"
          onClick={() => void carregar(filtro, soAtrasados)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1 text-xs text-neutral-300">
          Filtrar propostas por estado
          <input
            type="text"
            data-testid="vendas-filtro"
            value={filtro}
            onChange={(evento) => setFiltro(evento.target.value)}
            placeholder="draft, sent, accepted..."
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>
        <label className="flex items-center gap-2 text-[11px] text-neutral-400">
          <input
            type="checkbox"
            data-testid="vendas-so-atrasados"
            checked={soAtrasados}
            onChange={(evento) => setSoAtrasados(evento.target.checked)}
          />
          Só acompanhamentos atrasados
        </label>
        <button
          type="button"
          data-testid="vendas-filtrar"
          onClick={() => void carregar(filtro, soAtrasados)}
          className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800"
        >
          Filtrar
        </button>
      </div>

      {estado === "sem-vendas" && (
        <div data-testid="vendas-sem-vendas" className="text-xs text-amber-300">
          <p className="font-semibold">A força de vendas não está ligada neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há propostas»: é «não há vendas». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="vendas-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="vendas-a-carregar" className="text-xs text-neutral-400">
          A ler as vendas...
        </p>
      )}

      {estado === "pronta" && (
        <>
          <h4 className="mb-2 text-xs font-semibold text-neutral-300">
            Propostas ({propostas.length})
          </h4>

          {propostas.length === 0 ? (
            <p data-testid="vendas-sem-propostas" className="text-xs text-neutral-400">
              {filtro.trim() !== ""
                ? `Nenhuma proposta com estado «${filtro.trim()}».`
                : "As vendas estão ligadas e não há propostas nesta organização."}
            </p>
          ) : (
            <ul data-testid="vendas-propostas" className="space-y-1">
              {propostas.map((proposta) => (
                <li key={proposta.id}>
                  <button
                    type="button"
                    data-testid={`vendas-abrir-${proposta.id}`}
                    onClick={() => void abrir(proposta.id)}
                    className="w-full rounded px-2 py-1 text-left text-xs text-neutral-200 hover:bg-neutral-800"
                  >
                    <span className="font-mono text-[11px] text-neutral-400">{proposta.id}</span>{" "}
                    <span className="text-neutral-400">
                      [{descreverEstadoDaProposta(proposta.status)}] ·{" "}
                      {contarUnidades(proposta.items)} unidade(s) ·{" "}
                    </span>
                    <span className="font-semibold text-emerald-400">
                      {descreverValorDaProposta(proposta.total_cents, proposta.currency)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-2 mt-4 text-xs font-semibold text-neutral-300">
            Acompanhamentos{soAtrasados ? " atrasados" : ""} ({acompanhamentos.length})
          </h4>

          {acompanhamentos.length === 0 ? (
            <p data-testid="vendas-sem-acompanhamentos" className="text-xs text-neutral-400">
              {soAtrasados
                ? "Nenhum acompanhamento atrasado. Isto é diferente de «não há acompanhamentos»."
                : "Não há acompanhamentos de pós-venda nesta organização."}
            </p>
          ) : (
            <ul data-testid="vendas-acompanhamentos" className="space-y-1 text-xs text-neutral-300">
              {acompanhamentos.map((acompanhamento) => (
                <li key={acompanhamento.id} data-testid={`vendas-acompanhamento-${acompanhamento.id}`}>
                  <span className="text-neutral-400">
                    [{descreverEstadoDaProposta(acompanhamento.status)}]
                  </span>{" "}
                  {acompanhamento.reason} · vence {acompanhamento.due_at.slice(0, 10)}
                </li>
              ))}
            </ul>
          )}

          {detalheErro && (
            <p data-testid="vendas-detalhe-erro" role="alert" className="mt-2 text-xs text-red-400">
              {detalheErro}
            </p>
          )}

          {detalhe && (
            <div
              data-testid="vendas-detalhe"
              className="mt-3 rounded border border-neutral-700 bg-neutral-950/60 p-3 text-xs"
            >
              <h4 className="mb-1 font-semibold text-neutral-200">
                Proposta{" "}
                <span className="font-mono text-[11px] text-neutral-400">{detalhe.id}</span> ·{" "}
                {descreverEstadoDaProposta(detalhe.status)}
              </h4>

              {/* A CONTA ABERTA: subtotal, desconto, imposto, frete, total. */}
              <dl data-testid="vendas-conta" className="mt-1 space-y-0.5">
                {abrirAConta(detalhe).map((linha) => (
                  <div key={linha.etiqueta} className="flex justify-between gap-4">
                    <dt className="text-neutral-400">{linha.etiqueta}</dt>
                    <dd className="font-mono text-neutral-200">{linha.valor}</dd>
                  </div>
                ))}
              </dl>

              {(detalhe.items ?? []).length === 0 ? (
                <p data-testid="vendas-sem-itens" className="mt-2 text-neutral-400">
                  Não há itens nesta proposta.
                </p>
              ) : (
                <ul data-testid="vendas-itens" className="mt-2 space-y-0.5 text-neutral-300">
                  {(detalhe.items ?? []).map((item) => (
                    <li key={item.sku} className="flex justify-between gap-4">
                      <span>
                        {item.quantity}× {item.name}{" "}
                        <span className="font-mono text-[10px] text-neutral-500">{item.sku}</span>
                      </span>
                      <span className="font-mono text-[11px]">
                        {descreverValorDaProposta(item.total_cents, detalhe.currency)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}

              <p data-testid="vendas-validade" className="mt-2 text-[11px] text-neutral-500">
                válida até {detalhe.valid_until.slice(0, 10)}
                {detalhe.sent_at ? ` · enviada ${detalhe.sent_at.slice(0, 10)}` : " · não enviada"}
              </p>
            </div>
          )}
        </>
      )}

      <p data-testid="vendas-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê propostas, itens e acompanhamentos. Criar, enviar, decidir e completar
        continuam a ser ações separadas — e a conta aparece aberta porque um total único esconderia
        quanto foi desconto, imposto e frete.
      </p>
    </section>
  );
}
