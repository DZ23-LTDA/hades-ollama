// Painel de plugins -- catalogo, estado, prontidao, integridade e o formato aceito.
//
// O QUE ESTE PAINEL NAO FAZ:
//   - NAO mostra `requested_scopes` como se fossem permissoes concedidas: o estado nao
//     traz `granted_scopes`, e o painel diz isso.
//   - NAO mostra `ready_to_use: false` sem as razoes.
//   - NAO confunde `trusted` com `content_verified`: sao duas coisas diferentes.
//   - NAO instala, promove, reverte nem executa: 6 rotas de escrita ficam de fora.
import { useCallback, useEffect, useState } from "react";
import {
  descreverFormato,
  descreverIntegridade,
  descreverProntidao,
  descreverScopes,
  descreverScopesPedidos,
  listarEstados,
  listarPlugins,
  obterFormato,
  PluginsAusenteError,
  type EstadoDePlugin,
  type InstalacaoDePlugin,
  type RespostaDeFormato,
} from "@/lib/pluginsClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-registro" | "erro";

export function PluginsPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [formato, setFormato] = useState<RespostaDeFormato | null>(null);
  const [instalacoes, setInstalacoes] = useState<InstalacaoDePlugin[]>([]);
  const [estados, setEstados] = useState<EstadoDePlugin[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rFormato, rPlugins, rEstados] = await Promise.all([
        obterFormato(),
        listarPlugins(),
        listarEstados(),
      ]);
      setFormato(rFormato);
      setInstalacoes(rPlugins.plugins ?? []);
      setEstados(rEstados.plugins ?? []);
      setEstado("pronta");
    } catch (causa) {
      setFormato(null);
      setInstalacoes([]);
      setEstados([]);
      if (causa instanceof PluginsAusenteError) {
        setErro(causa.message);
        setEstado("sem-registro");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler os plugins.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const concedidosPorId = new Map<string, string[] | undefined>();
  for (const inst of instalacoes) {
    concedidosPorId.set(inst.manifest.id, inst.granted_scopes);
  }

  return (
    <section
      data-testid="plugins"
      aria-labelledby="plugins-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="plugins-titulo" className="text-sm font-semibold text-neutral-100">
          Plugins
        </h3>
        <button
          type="button"
          data-testid="plugins-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "sem-registro" && (
        <div data-testid="plugins-sem-registro" className="text-xs text-amber-300">
          <p className="font-semibold">O registro de plugins não está ligado neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há plugins instalados»: é «não há registro». Uma lista vazia aqui
            mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="plugins-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="plugins-a-carregar" className="text-xs text-neutral-400">
          A ler os plugins...
        </p>
      )}

      {estado === "pronta" && (
        <>
          {/* O formato e' o CONTRATO ACEITO, nao uma sugestao. */}
          {formato && (
            <p data-testid="plugins-formato" className="text-[11px] leading-relaxed text-neutral-400">
              {descreverFormato(formato)}
            </p>
          )}

          <h4 className="mb-2 mt-3 text-xs font-semibold text-neutral-300">
            Instalados ({instalacoes.length})
          </h4>

          {instalacoes.length === 0 ? (
            <p data-testid="plugins-sem-instalacoes" className="text-xs text-neutral-400">
              O registro está ligado e não há plugins instalados nesta organização.
            </p>
          ) : (
            <ul data-testid="plugins-instalacoes" className="space-y-1">
              {instalacoes.map((inst) => (
                <li
                  key={inst.manifest.id}
                  data-testid={`plugins-instalacao-${inst.manifest.id}`}
                  className="rounded border border-neutral-800 px-2 py-1 text-xs text-neutral-200"
                >
                  <span className="font-semibold">{inst.manifest.name}</span>{" "}
                  <span className="text-neutral-400">
                    v{inst.manifest.version} · {inst.manifest.kind}
                  </span>
                  <span className="block text-[11px] text-neutral-500">
                    {descreverIntegridade(inst)}
                  </span>
                  {/* Pedido vs concedido: a diferenca e' a informacao. */}
                  <span
                    data-testid={`plugins-scopes-${inst.manifest.id}`}
                    className="block text-[11px] text-neutral-400"
                  >
                    {descreverScopes(inst.requested_scopes, inst.granted_scopes)}
                  </span>
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-2 mt-4 text-xs font-semibold text-neutral-300">
            Estado ({estados.length})
          </h4>

          {estados.length === 0 ? (
            <p data-testid="plugins-sem-estados" className="text-xs text-neutral-400">
              Sem estados reportados.
            </p>
          ) : (
            <ul data-testid="plugins-estados" className="space-y-1">
              {estados.map((e) => (
                <li
                  key={e.plugin_id}
                  data-testid={`plugins-estado-${e.plugin_id}`}
                  className="rounded border border-neutral-800 px-2 py-1 text-[11px]"
                >
                  <span className="font-semibold text-neutral-200">{e.plugin_id}</span>{" "}
                  <span className="text-neutral-400">[{e.state}]</span>
                  <span
                    data-testid={`plugins-prontidao-${e.plugin_id}`}
                    className={
                      e.ready_to_use ? "block text-emerald-400" : "block text-amber-400"
                    }
                  >
                    {descreverProntidao(e)}
                  </span>
                  <span className="block text-neutral-500">{descreverIntegridade(e)}</span>
                  {/* Os PEDIDOS, nunca apresentados como concedidos. */}
                  <span
                    data-testid={`plugins-pedidos-${e.plugin_id}`}
                    className="block text-neutral-500"
                  >
                    {descreverScopesPedidos(e.requested_scopes)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="plugins-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê. Instalar, promover, reverter, executar e instalar bundle continuam a ser
        ações separadas — e um scope <em>pedido</em> nunca é apresentado como concedido.
      </p>
    </section>
  );
}
