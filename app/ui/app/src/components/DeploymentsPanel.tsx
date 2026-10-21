// Painel de providers de deploy -- a superfície que faltava para `/deployments`.
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - não mostra token nenhum: mostra o NOME da variável de ambiente que o contém;
//   - não mostra credenciais em URL: redige o userinfo e DIZ que o redigiu;
//   - não faz deploy: este painel só lê a configuração;
//   - não mostra lista vazia como avaria: nenhum provider configurado é um estado
//     legítimo, e o painel di-lo com a consequência (não há para onde publicar).
//
//	*"Publicar localmente" e "publicar publicamente" não são a mesma coisa, e um
//	painel que os mostra iguais ensina o utilizador a confundir os dois.*
import { useCallback, useEffect, useState } from "react";
import {
  descreverTimeout,
  descreverTokenEnv,
  listarProvidersDeDeploy,
  redigirBaseURL,
  type ProviderDeDeploy,
} from "@/lib/deploymentsClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "vazia" | "erro";

export function DeploymentsPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [providers, setProviders] = useState<ProviderDeDeploy[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await listarProvidersDeDeploy();
      const lista = resposta.providers ?? [];
      setProviders(lista);
      setEstado(lista.length === 0 ? "vazia" : "pronta");
    } catch (causa) {
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler os providers de deploy.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const algumRedigido = providers.some((p) => redigirBaseURL(p.base_url).removeuCredenciais);

  return (
    <section
      data-testid="deployments"
      aria-labelledby="deployments-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="deployments-titulo" className="text-sm font-semibold text-neutral-100">
          Providers de deploy
        </h3>
        <button
          type="button"
          data-testid="deployments-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="deployments-a-carregar" className="text-xs text-neutral-400">
          A ler os providers...
        </p>
      )}

      {estado === "erro" && erro && (
        <p data-testid="deployments-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "vazia" && (
        <p data-testid="deployments-vazio" className="text-xs text-neutral-400">
          Nenhum provider de deploy está configurado. Sem um provider, o HADES não tem para onde
          publicar — a configuração de deploy fica vazia, e não "pronta".
        </p>
      )}

      {estado === "pronta" && (
        <>
          <table
            data-testid="deployments-tabela"
            className="w-full text-left text-xs text-neutral-200"
          >
            <thead className="text-neutral-400">
              <tr>
                <th scope="col" className="py-1 pr-2">provider</th>
                <th scope="col" className="py-1 pr-2">base URL</th>
                <th scope="col" className="py-1 pr-2">token</th>
                <th scope="col" className="py-1 pr-2">timeout</th>
              </tr>
            </thead>
            <tbody>
              {providers.map((provider) => {
                const base = redigirBaseURL(provider.base_url);
                return (
                  <tr
                    key={provider.id}
                    data-testid={`deployments-provider-${provider.id}`}
                    className="border-t border-neutral-800"
                  >
                    <td className="py-1 pr-2 font-mono text-[11px]">{provider.provider}</td>
                    <td
                      data-testid={`deployments-url-${provider.id}`}
                      className="py-1 pr-2 font-mono text-[11px] text-neutral-400"
                    >
                      {base.url || "—"}
                      {base.removeuCredenciais && (
                        <span
                          data-testid={`deployments-redigido-${provider.id}`}
                          className="ml-1 rounded bg-amber-900/60 px-1 text-[10px] text-amber-200"
                        >
                          credenciais removidas
                        </span>
                      )}
                    </td>
                    <td
                      data-testid={`deployments-token-${provider.id}`}
                      className="py-1 pr-2 text-[11px] text-neutral-400"
                    >
                      {descreverTokenEnv(provider.token_env)}
                    </td>
                    <td className="py-1 pr-2 text-[11px] text-neutral-400">
                      {descreverTimeout(provider.timeout_seconds)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {algumRedigido && (
            <p
              data-testid="deployments-aviso-redigido"
              className="mt-2 text-[11px] text-amber-300"
            >
              Uma das URLs trazia credenciais e elas foram removidas desta vista. A configuração no
              servidor não foi alterada — e a credencial devia estar numa variável de ambiente, não
              na URL.
            </p>
          )}
        </>
      )}

      <p data-testid="deployments-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê a configuração. Publicar continua a exigir aprovação humana específica no
        momento da ação, e um provider configurado não é uma publicação feita.
      </p>
    </section>
  );
}
