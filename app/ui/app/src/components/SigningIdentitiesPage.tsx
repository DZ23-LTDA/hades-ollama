// FASE 28.5 do adendo V3: a area "Identidades de Assinatura" do Mission Control.
//
// Mostra SO' METADADOS. Nao existe campo nesta pagina, nem no cliente, nem na rota, que possa
// transportar o valor privado de uma chave. *Um painel de identidades que mostra a chave nao e'
// um painel: e' um vazamento com interface.*
//
// Regras que a pagina respeita, e que os testes cobram:
//   - tres estados de registo distintos: nao configurado, carregado e vazio, carregado com
//     identidades. *Um ecra vazio que significa tres coisas diferentes ensina o utilizador a
//     nao olhar.*
//   - `desconhecida` em ambar, nao em vermelho.
//   - identidade de producao NUNCA aparece como "pode assinar agora" sem aprovacao.
//   - nenhum botao de assinar, importar ou revogar: essas accoes exigem confirmacao humana no
//     momento da accao e pertencem a outra superficie.

import { useCallback, useEffect, useState } from "react";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";
import {
  fetchSigningIdentities,
  corDoEstadoDaIdentidade,
  rotuloDoEstadoDaIdentidade,
  rotuloDoAmbiente,
  validadeLegivel,
  podeAssinarAgora,
  type IdentidadeDeAssinatura,
  type ProblemaDeIdentidade,
  type SigningIdentitiesResposta,
} from "@/lib/signingClient";
import {
  ArrowPathIcon,
  FingerPrintIcon,
  ExclamationTriangleIcon,
  CheckCircleIcon,
  XCircleIcon,
  KeyIcon,
  LockClosedIcon,
} from "@heroicons/react/24/outline";

type EstadoDoCarregamento = "a-carregar" | "pronto" | "erro";

function IconeDoEstado({ estado }: { estado: IdentidadeDeAssinatura["estado"] }) {
  switch (estado) {
    case "ativa":
      return <CheckCircleIcon className="h-5 w-5 text-emerald-400" aria-hidden="true" />;
    case "expira_breve":
    case "desconhecida":
      return <ExclamationTriangleIcon className="h-5 w-5 text-amber-400" aria-hidden="true" />;
    default:
      return <XCircleIcon className="h-5 w-5 text-red-400" aria-hidden="true" />;
  }
}

export function SigningIdentitiesPage() {
  const [dados, setDados] = useState<SigningIdentitiesResposta | null>(null);
  const [estado, setEstado] = useState<EstadoDoCarregamento>("a-carregar");
  const [erro, setErro] = useState<string>("");

  const carregar = useCallback(async () => {
    setEstado("a-carregar");
    setErro("");
    try {
      const resposta = await fetchSigningIdentities();
      setDados(resposta);
      setEstado("pronto");
    } catch (e: unknown) {
      setErro(e instanceof Error ? e.message : String(e));
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const problemas: ProblemaDeIdentidade[] = dados?.problemas ?? [];
  const identidades: IdentidadeDeAssinatura[] = dados?.identidades ?? [];

  return (
    <SidebarLayout
      title="Identidades de Assinatura"
      sidebar={<AppSidebar current="signing" />}
    >
      <div className="mx-auto max-w-6xl space-y-6 px-4 py-8 lg:px-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
              <KeyIcon className="h-6 w-6" aria-hidden="true" />
              Identidades de Assinatura
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              Metadados das chaves que o HADES pode usar para assinar. O valor privado vive no
              cofre do sistema e nunca aparece aqui.
            </p>
          </div>
          <button
            type="button"
            onClick={() => void carregar()}
            className="inline-flex items-center gap-2 rounded-lg border border-neutral-300 px-3 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <ArrowPathIcon className="h-4 w-4" aria-hidden="true" />
            Recarregar
          </button>
        </div>

        {estado === "a-carregar" ? (
          <p className="text-sm text-neutral-500 dark:text-neutral-400" role="status">
            A ler o registo de identidades…
          </p>
        ) : null}

        {estado === "erro" ? (
          <div
            role="alert"
            className="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300"
          >
            <p className="font-medium">Não foi possível ler o registo de identidades.</p>
            <p className="mt-1 text-red-200/80">{erro}</p>
          </div>
        ) : null}

        {estado === "pronto" && dados ? (
          <>
            {/* O aviso vem do servidor: o consumidor nao deve ter de adivinhar o contrato. */}
            <div className="flex items-start gap-3 rounded-lg border border-neutral-200 bg-neutral-50 p-4 text-sm dark:border-neutral-800 dark:bg-neutral-900">
              <LockClosedIcon
                className="mt-0.5 h-4 w-4 shrink-0 text-neutral-500"
                aria-hidden="true"
              />
              <div>
                <p className="font-medium text-neutral-900 dark:text-white">{dados.aviso}</p>
                <p className="mt-1 text-neutral-600 dark:text-neutral-400">
                  {dados.estado.mensagem}
                </p>
              </div>
            </div>

            {/* NAO CONFIGURADO: explicar o que fazer, nao mostrar um ecra vazio. */}
            {!dados.estado.configurado ? (
              <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-4 text-sm">
                <p className="font-medium text-amber-300">Registo não configurado</p>
                <p className="mt-1 text-amber-200/80">
                  Nenhuma identidade declarada. O registo é um ficheiro de metadados ao lado do
                  cofre de credenciais; criá-lo exige uma acção consciente do proprietário.
                </p>
                {dados.estado.caminho ? (
                  <p className="mt-2 break-all font-mono text-xs text-amber-200/60">
                    {dados.estado.caminho}
                  </p>
                ) : null}
              </div>
            ) : null}

            {/* CARREGADO E VAZIO: estado diferente de "nao configurado". */}
            {dados.estado.configurado && identidades.length === 0 ? (
              <p className="text-sm text-neutral-500 dark:text-neutral-400">
                O registo está configurado e não declara nenhuma identidade.
              </p>
            ) : null}

            {identidades.length > 0 ? (
              <div className="overflow-x-auto rounded-lg border border-neutral-200 dark:border-neutral-800">
                <table className="w-full text-left text-sm">
                  <caption className="sr-only">
                    Identidades de assinatura registadas, com plataforma, ambiente, validade e
                    estado
                  </caption>
                  <thead className="bg-neutral-50 dark:bg-neutral-900">
                    <tr className="text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
                      <th scope="col" className="px-4 py-2 font-medium">
                        Identidade
                      </th>
                      <th scope="col" className="px-4 py-2 font-medium">
                        Plataforma
                      </th>
                      <th scope="col" className="px-4 py-2 font-medium">
                        Ambiente
                      </th>
                      <th scope="col" className="px-4 py-2 font-medium">
                        Validade
                      </th>
                      <th scope="col" className="px-4 py-2 font-medium">
                        Estado
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {identidades.map((identidade) => (
                      <tr
                        key={identidade.nome}
                        className="border-t border-neutral-200 dark:border-neutral-800"
                      >
                        <td className="px-4 py-3">
                          <span className="flex items-center gap-2 font-medium text-neutral-900 dark:text-white">
                            <IconeDoEstado estado={identidade.estado} />
                            {identidade.nome}
                          </span>
                          <span className="mt-1 flex items-center gap-1 font-mono text-xs text-neutral-500">
                            <FingerPrintIcon className="h-3 w-3" aria-hidden="true" />
                            …{identidade.fingerprint_publico_fim}
                          </span>
                          {identidade.pacote_ou_bundle ? (
                            <span className="block text-xs text-neutral-500">
                              {identidade.pacote_ou_bundle}
                            </span>
                          ) : null}
                        </td>
                        <td className="px-4 py-3 text-neutral-700 dark:text-neutral-300">
                          {identidade.plataforma}
                        </td>
                        <td className="px-4 py-3 text-neutral-700 dark:text-neutral-300">
                          {rotuloDoAmbiente(identidade.ambiente)}
                          {identidade.exige_aprovacao ? (
                            <span className="ml-1 text-xs text-amber-400">(aprovação)</span>
                          ) : null}
                        </td>
                        <td className="px-4 py-3 text-neutral-700 dark:text-neutral-300">
                          {validadeLegivel(identidade.expira_em)}
                        </td>
                        <td className="px-4 py-3">
                          <span className={corDoEstadoDaIdentidade(identidade.estado)}>
                            {rotuloDoEstadoDaIdentidade(identidade.estado)}
                          </span>
                          <span className="block text-xs text-neutral-500">
                            {podeAssinarAgora(identidade)
                              ? "pode assinar agora"
                              : identidade.exige_aprovacao
                                ? "requer aprovação"
                                : "não disponível"}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : null}

            {problemas.length > 0 ? (
              <div>
                <h2 className="text-sm font-semibold uppercase tracking-wide text-amber-400">
                  Pontos de atenção
                </h2>
                <ul className="mt-2 space-y-2">
                  {problemas.map((problema) => (
                    <li
                      key={`${problema.nome}-${problema.mensagem}`}
                      className="rounded-lg border border-neutral-200 p-3 text-sm dark:border-neutral-800"
                    >
                      <span className="font-medium text-neutral-900 dark:text-white">
                        {problema.nome}
                      </span>
                      <span
                        className={
                          problema.severidade === "critico" ? "text-red-400" : "text-amber-400"
                        }
                      >
                        {" "}
                        · {problema.severidade}
                      </span>
                      <p className="mt-1 text-neutral-600 dark:text-neutral-400">
                        {problema.mensagem}
                      </p>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </>
        ) : null}
      </div>
    </SidebarLayout>
  );
}
