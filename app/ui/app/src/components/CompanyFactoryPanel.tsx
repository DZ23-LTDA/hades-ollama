// Relatorio da fabrica de empresa -- o que a fabrica fez, e o que ficou para um humano.
//
// NAO faz:
//   - NAO mostra "0 clientes" como se fosse uma contagem;
//   - NAO chama ao valor de tabela do estoque "receita" nem "avaliacao";
//   - NAO esconde os itens que dependem de terceiros.
import { useCallback, useEffect, useState } from "react";
import {
  descreverClientesCriados,
  descreverValorDeCatalogo,
  dependeDeTerceiro,
  FabricaAusenteError,
  obterRelatorioDaFabrica,
  resumirChecklist,
  rotuloDoItem,
  type RelatorioDaFabrica,
} from "@/lib/companyFactoryClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "ausente" | "erro";

export function CompanyFactoryPanel() {
  const [id, setId] = useState("");
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [relatorio, setRelatorio] = useState<RelatorioDaFabrica | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async (alvo: string) => {
    const limpo = alvo.trim();
    if (limpo === "") {
      setEstado("ausente");
      setErro("Indique o id de uma empresa para ver o relatório.");
      setRelatorio(null);
      return;
    }
    setErro(null);
    setEstado("a-carregar");
    try {
      const r = await obterRelatorioDaFabrica(limpo);
      setRelatorio(r);
      setEstado("pronta");
    } catch (causa) {
      setRelatorio(null);
      if (causa instanceof FabricaAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler o relatorio.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar("");
  }, [carregar]);

  const checklist = relatorio?.checklist ?? [];
  const artefactos = relatorio?.artifacts ?? [];

  return (
    <section
      data-testid="fabrica"
      aria-labelledby="fabrica-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="fabrica-titulo" className="text-sm font-semibold text-neutral-100">
          Fábrica de empresa
        </h3>
        <div className="flex items-center gap-2">
          <input
            data-testid="fabrica-id"
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder="id da empresa"
            aria-label="Identificador da empresa"
            className="w-44 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <button
            type="button"
            data-testid="fabrica-carregar"
            onClick={() => void carregar(id)}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Ver relatório
          </button>
        </div>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="fabrica-a-carregar" className="text-xs text-neutral-400">
          A ler o relatório...
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="fabrica-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="fabrica-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && relatorio && (
        <>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-[11px]">
            <dt className="text-neutral-500">Lançada em</dt>
            <dd className="text-neutral-300">{relatorio.launched_at}</dd>

            <dt className="text-neutral-500">Produtos</dt>
            <dd data-testid="fabrica-produtos" className="text-neutral-300">
              {(relatorio.products ?? []).length}
            </dd>

            {/* O zero e' uma fronteira, nao uma contagem. */}
            <dt className="text-neutral-500">Clientes</dt>
            <dd data-testid="fabrica-clientes" className="text-neutral-300">
              {descreverClientesCriados(relatorio.customers_created)}
            </dd>

            {/* Valor de TABELA do estoque, nao receita. */}
            <dt className="text-neutral-500">Estoque inicial</dt>
            <dd data-testid="fabrica-valor" className="text-neutral-300">
              {descreverValorDeCatalogo(relatorio.catalog_value_cents)}
            </dd>
          </dl>

          <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
            Checklist — {resumirChecklist(checklist)}
          </h4>

          {checklist.length === 0 ? (
            <p data-testid="fabrica-sem-checklist" className="text-[11px] text-neutral-500">
              O relatório não trouxe checklist.
            </p>
          ) : (
            <ul data-testid="fabrica-checklist" className="space-y-0.5 text-[11px]">
              {checklist.map((item, i) => (
                <li
                  key={`${item.id ?? "item"}-${i}`}
                  data-testid={`fabrica-item-${item.id ?? i}`}
                  className={dependeDeTerceiro(item) ? "text-amber-300" : "text-neutral-400"}
                >
                  {rotuloDoItem(item)}
                  {dependeDeTerceiro(item) && (
                    <span className="text-amber-400"> — depende de terceiro</span>
                  )}
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
            Artefactos materializados ({artefactos.length})
          </h4>
          {artefactos.length === 0 ? (
            <p data-testid="fabrica-sem-artefactos" className="text-[11px] text-neutral-500">
              Nenhum artefacto no relatório.
            </p>
          ) : (
            <ul data-testid="fabrica-artefactos" className="space-y-0.5 text-[11px] text-neutral-400">
              {artefactos.map((a, i) => (
                <li key={`${a.id ?? a.kind ?? "art"}-${i}`}>
                  {a.kind ?? "artefacto"} · {a.title ?? a.name ?? a.path ?? a.url ?? "sem rótulo"}
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="fabrica-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        A fábrica materializa plano, marca, precificação e site. <strong>Não cria clientes nem
        emite registro legal</strong> — o zero em «clientes» é uma fronteira, não uma contagem
        falhada. O valor do estoque é valor de tabela, não receita.
      </p>
    </section>
  );
}
