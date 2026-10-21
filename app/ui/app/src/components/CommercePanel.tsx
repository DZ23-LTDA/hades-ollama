// Painel de comércio -- a superfície do maior domínio restante (12 rotas).
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - NÃO mostra centavos como reais: `centavosParaReais` converte, e um valor de
//     dinheiro na unidade errada é uma mentira com duas casas decimais;
//   - NÃO trata loja ausente (503) como lista vazia;
//   - NÃO conta devoluções idempotentes (`duplicated`) como pedidos novos;
//   - NÃO inventa rótulo para estado desconhecido;
//   - NÃO cria, paga, reconcilia, cumpre nem cancela: só lê.
import { useCallback, useEffect, useState } from "react";
import {
  centavosParaReais,
  contarPedidos,
  descreverEstadoDoPedido,
  descreverPreco,
  LojAusenteError,
  listarPedidos,
  listarProdutos,
  resumoDoComercio,
  type PedidoDeComercio,
  type ProdutoDeComercio,
  type ResumoDeComercio,
} from "@/lib/commerceClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-loja" | "erro";

export function CommercePanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [produtos, setProdutos] = useState<ProdutoDeComercio[]>([]);
  const [pedidos, setPedidos] = useState<PedidoDeComercio[]>([]);
  const [resumo, setResumo] = useState<ResumoDeComercio | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rProdutos, rPedidos, rResumo] = await Promise.all([
        listarProdutos(),
        listarPedidos(),
        resumoDoComercio(),
      ]);
      setProdutos(rProdutos.products ?? []);
      setPedidos(rPedidos.orders ?? []);
      setResumo(rResumo);
      setEstado("pronta");
    } catch (causa) {
      setProdutos([]);
      setPedidos([]);
      setResumo(null);
      if (causa instanceof LojAusenteError) {
        setErro(causa.message);
        setEstado("sem-loja");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler o comércio.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const contagem = contarPedidos(pedidos);

  return (
    <section
      data-testid="commerce"
      aria-labelledby="commerce-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="commerce-titulo" className="text-sm font-semibold text-neutral-100">
          Comércio
        </h3>
        <button
          type="button"
          data-testid="commerce-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="commerce-a-carregar" className="text-xs text-neutral-400">
          A ler o comércio...
        </p>
      )}

      {estado === "sem-loja" && (
        <div data-testid="commerce-sem-loja" className="text-xs text-amber-300">
          <p className="font-semibold">A loja não está ligada neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há produtos»: é «não há loja». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="commerce-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && resumo && (
        <>
          <p data-testid="commerce-resumo" className="mb-2 text-xs text-neutral-300">
            {resumo.products} produto(s) · {resumo.orders} pedido(s) ·{" "}
            <span className="text-emerald-400">
              receita {centavosParaReais(resumo.revenue_cents)}
            </span>
          </p>
          <p data-testid="commerce-por-estado" className="mb-3 text-xs text-neutral-400">
            {resumo.pending} pendente(s) · {resumo.paid} pago(s) · {resumo.fulfilled} cumprido(s) ·{" "}
            {resumo.cancelled} cancelado(s)
          </p>

          {resumo.low_stock_skus && resumo.low_stock_skus.length > 0 && (
            <p data-testid="commerce-stock-baixo" className="mb-3 text-xs text-amber-300">
              Sem disponibilidade: {resumo.low_stock_skus.join(", ")}
            </p>
          )}

          <h4 className="mb-1 text-xs font-semibold text-neutral-300">
            Produtos ({produtos.length})
          </h4>
          {produtos.length === 0 ? (
            <p data-testid="commerce-sem-produtos" className="mb-3 text-xs text-neutral-400">
              A loja está ligada e não há produtos nesta organização.
            </p>
          ) : (
            <table
              data-testid="commerce-produtos"
              className="mb-4 w-full text-left text-xs text-neutral-200"
            >
              <thead className="text-neutral-400">
                <tr>
                  <th scope="col" className="py-1 pr-2">SKU</th>
                  <th scope="col" className="py-1 pr-2">nome</th>
                  <th scope="col" className="py-1 pr-2">preço</th>
                  <th scope="col" className="py-1 pr-2">estado</th>
                </tr>
              </thead>
              <tbody>
                {produtos.map((produto) => (
                  <tr
                    key={produto.id}
                    data-testid={`commerce-produto-${produto.sku}`}
                    className="border-t border-neutral-800"
                  >
                    <td className="py-1 pr-2 font-mono text-[11px]">{produto.sku}</td>
                    <td className="py-1 pr-2">{produto.name}</td>
                    <td className="py-1 pr-2">{descreverPreco(produto)}</td>
                    <td className="py-1 pr-2 text-neutral-400">
                      {produto.active ? "activo" : "inactivo"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          <h4 className="mb-1 text-xs font-semibold text-neutral-300">
            Pedidos ({contagem.total})
          </h4>
          <p data-testid="commerce-contagem-pedidos" className="mb-2 text-xs text-neutral-400">
            {contagem.novos} novo(s)
            {contagem.devolvidos > 0 &&
              ` · ${contagem.devolvidos} devolução(ões) idempotente(s) — contadas à parte`}
          </p>

          {pedidos.length === 0 ? (
            <p data-testid="commerce-sem-pedidos" className="text-xs text-neutral-400">
              A loja está ligada e não há pedidos nesta organização.
            </p>
          ) : (
            <table
              data-testid="commerce-pedidos"
              className="w-full text-left text-xs text-neutral-200"
            >
              <thead className="text-neutral-400">
                <tr>
                  <th scope="col" className="py-1 pr-2">número</th>
                  <th scope="col" className="py-1 pr-2">estado</th>
                  <th scope="col" className="py-1 pr-2">total</th>
                  <th scope="col" className="py-1 pr-2">nota</th>
                </tr>
              </thead>
              <tbody>
                {pedidos.map((pedido) => (
                  <tr
                    key={pedido.id}
                    data-testid={`commerce-pedido-${pedido.number}`}
                    className="border-t border-neutral-800"
                  >
                    <td className="py-1 pr-2 font-mono text-[11px]">{pedido.number}</td>
                    <td className="py-1 pr-2">{descreverEstadoDoPedido(pedido.status)}</td>
                    <td className="py-1 pr-2">{centavosParaReais(pedido.total_cents, pedido.currency)}</td>
                    <td className="py-1 pr-2 text-[10px] text-amber-300">
                      {pedido.duplicated ? "devolução idempotente" : ""}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}

      <p data-testid="commerce-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê. Criar pedido, confirmar pagamento, reconciliar, cumprir e cancelar continuam
        a ser ações separadas — e os valores são mostrados em reais a partir dos cêntimos (centavos) que o
        servidor envia.
      </p>
    </section>
  );
}
