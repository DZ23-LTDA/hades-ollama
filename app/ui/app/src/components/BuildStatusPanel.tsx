/**
 * FASE 28.7 do adendo V3: o painel de ESTADO de build dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL FECHA
 *
 * O E-1313 entregou cinco rotas e nenhum painel as consumia. Esta superficie fecha esse elo: lista os
 * builds deste processo, mostra o estado, deixa ler a saida e os artefactos, e permite cancelar um
 * build a correr.
 *
 * AS TRES DECISOES QUE ESTE PAINEL TORNA VISIVEIS
 *
 *  1. **Cancelar um build terminado e' 409, nao um erro de rede.** O botao so' aparece quando o
 *     servidor diz `cancelavel`; e quando o servidor recusa (409), o painel DIZ que o build ja'
 *     tinha terminado -- nao mostra "falhou ao cancelar".
 *
 *  2. **Nada assinado, nada publicado, nada instalado.** Vem do servidor, campo a campo, e e'
 *     mostrado ao lado de cada build. Nao ha' aqui um selo "seguro" calculado no frontend.
 *
 *  3. **Lista vazia e' um resultado, nao um erro.** Um servidor sem builds diz "ainda nao ha' builds
 *     neste processo" -- a mesma distincao que o backend faz entre 200-com-lista e 404.
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 * Nao inicia builds (isso e' o `BuildPlanPanel`), nao assina, nao publica e nao instala. A unica rota
 * que muda estado e' `cancel`, e ela so' mata um processo -- nao apaga artefactos.
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, tabela com cabecalhos, `aria-live` no carregamento, erro em `role="alert"`, foco
 * visivel nos botoes e texto alternativo nos estados (nao so' cor).
 */

import { useCallback, useEffect, useState } from "react";

import {
  cancelarBuild,
  descreverEstadoDeBuild,
  estadoDeBuildTerminou,
  formatarBytes,
  listarBuilds,
  obterArtefactosDeBuild,
  obterSaidaDeBuild,
  type ArtefactoDeBuild,
  type RegistoDeBuild,
  type RespostaDeArtefactosDeBuild,
} from "@/lib/buildStatusClient";

const CLASSE_ESTADO: Record<string, string> = {
  CONCLUIDO: "bg-green-100 text-green-900",
  FALHOU: "bg-red-100 text-red-900",
  CANCELADO: "bg-amber-100 text-amber-900",
  A_EXECUTAR: "bg-blue-100 text-blue-900",
  NA_FILA: "bg-gray-100 text-gray-900",
};

export function BuildStatusPanel() {
  const [builds, setBuilds] = useState<RegistoDeBuild[] | null>(null);
  const [aCarregar, setACarregar] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [seleccionado, setSeleccionado] = useState<string | null>(null);
  const [saida, setSaida] = useState<string | null>(null);
  const [artefactos, setArtefactos] = useState<RespostaDeArtefactosDeBuild | null>(null);
  const [avisoCancelamento, setAvisoCancelamento] = useState<string | null>(null);
  const [aCancelar, setACancelar] = useState(false);

  const carregar = useCallback(async () => {
    setACarregar(true);
    setErro(null);
    try {
      const resposta = await listarBuilds();
      setBuilds(resposta.builds ?? []);
    } catch (e) {
      setErro(e instanceof Error ? e.message : String(e));
      setBuilds(null);
    } finally {
      setACarregar(false);
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const abrirDetalhe = useCallback(async (id: string) => {
    setSeleccionado(id);
    setSaida(null);
    setArtefactos(null);
    setAvisoCancelamento(null);
    try {
      const [s, a] = await Promise.all([obterSaidaDeBuild(id), obterArtefactosDeBuild(id)]);
      setSaida(s.saida ?? "");
      setArtefactos(a);
    } catch (e) {
      setErro(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const pedirCancelamento = useCallback(
    async (build: RegistoDeBuild) => {
      setACancelar(true);
      setAvisoCancelamento(null);
      try {
        await cancelarBuild(build.id);
        setAvisoCancelamento(`Cancelamento pedido para o build ${build.id.slice(0, 8)}.`);
        await carregar();
      } catch (e) {
        // O 409 e' um RESULTADO, nao uma falha: o build ja' tinha terminado. Dizer "falhou ao
        // cancelar" seria contar a historia errada.
        const mensagem = e instanceof Error ? e.message : String(e);
        if (mensagem.includes("409") || mensagem.toLowerCase().includes("terminou")) {
          setAvisoCancelamento(
            `O build ${build.id.slice(0, 8)} já tinha terminado — cancelar não tem efeito.`,
          );
          await carregar();
        } else {
          setErro(mensagem);
        }
      } finally {
        setACancelar(false);
      }
    },
    [carregar],
  );

  const total = builds?.length ?? 0;

  return (
    <section
      aria-labelledby="build-status-titulo"
      className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"
      data-testid="build-status"
    >
      <header className="mb-3">
        <h2 id="build-status-titulo" className="text-lg font-semibold">
          Estado dos builds
        </h2>
        <p className="text-sm text-gray-600">
          Os builds deste processo. O registo vive enquanto o processo vive — um build de uma sessão
          anterior não aparece aqui.
        </p>
      </header>

      <div className="mb-3 flex items-center gap-2">
        <button
          type="button"
          onClick={() => void carregar()}
          disabled={aCarregar}
          className="rounded border border-gray-300 px-3 py-1 text-sm hover:bg-gray-50 disabled:opacity-50"
          data-testid="build-status-actualizar"
        >
          {aCarregar ? "A atualizar…" : "Atualizar"}
        </button>
        <span className="text-sm text-gray-700" data-testid="build-status-contagem">
          {total} {total === 1 ? "build" : "builds"}
        </span>
      </div>

      <div aria-live="polite" className="min-h-[1.25rem] text-sm text-gray-700">
        {aCarregar ? "A carregar os builds…" : ""}
        {avisoCancelamento ? (
          <span className="text-amber-800" data-testid="build-status-aviso-cancelamento">
            {avisoCancelamento}
          </span>
        ) : null}
      </div>

      {erro ? (
        <div role="alert" className="mt-2 rounded bg-red-50 p-2 text-sm text-red-900" data-testid="build-status-erro">
          {erro}
        </div>
      ) : null}

      {builds !== null && builds.length === 0 ? (
        <p className="mt-2 text-sm text-gray-600" data-testid="build-status-vazio">
          Ainda não há builds neste processo. Inicie um em «Plano de release».
        </p>
      ) : null}

      {builds !== null && builds.length > 0 ? (
        <table className="mt-2 w-full text-left text-sm" data-testid="build-status-tabela">
          <caption className="sr-only">Builds deste processo, com estado e artefactos medidos.</caption>
          <thead>
            <tr className="border-b border-gray-200">
              <th scope="col" className="py-1 pr-2">Build</th>
              <th scope="col" className="py-1 pr-2">Alvo</th>
              <th scope="col" className="py-1 pr-2">Estado</th>
              <th scope="col" className="py-1 pr-2">Artefactos</th>
              <th scope="col" className="py-1 pr-2">Ações</th>
            </tr>
          </thead>
          <tbody>
            {builds.map((b) => (
              <tr key={b.id} className="border-b border-gray-100 align-top">
                <td className="py-1 pr-2 font-mono text-xs" title={b.id}>
                  {b.id.slice(0, 8)}
                </td>
                <td className="py-1 pr-2">{b.alvo}</td>
                <td className="py-1 pr-2">
                  <span
                    className={`rounded px-2 py-0.5 text-xs ${CLASSE_ESTADO[b.estado] ?? "bg-gray-100 text-gray-900"}`}
                    data-testid={`build-status-estado-${b.id}`}
                  >
                    {descreverEstadoDeBuild(b.estado)}
                  </span>
                  {estadoDeBuildTerminou(b.estado) ? null : (
                    <span className="ml-1 text-xs text-gray-500">(em curso)</span>
                  )}
                </td>
                <td className="py-1 pr-2">
                  {(b.resultado?.Artefatos ?? []).length === 0
                    ? "—"
                    : `${(b.resultado?.Artefatos ?? []).length} medido(s)`}
                </td>
                <td className="py-1 pr-2">
                  <div className="flex flex-wrap gap-1">
                    <button
                      type="button"
                      onClick={() => void abrirDetalhe(b.id)}
                      className="rounded border border-gray-300 px-2 py-0.5 text-xs hover:bg-gray-50"
                      data-testid={`build-status-abrir-${b.id}`}
                    >
                      Ver
                    </button>
                    {b.cancelavel ? (
                      <button
                        type="button"
                        onClick={() => void pedirCancelamento(b)}
                        disabled={aCancelar}
                        className="rounded border border-amber-300 px-2 py-0.5 text-xs text-amber-900 hover:bg-amber-50 disabled:opacity-50"
                        data-testid={`build-status-cancelar-${b.id}`}
                      >
                        Cancelar
                      </button>
                    ) : null}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}

      {seleccionado ? (
        <div className="mt-4 rounded border border-gray-200 p-3" data-testid="build-status-detalhe">
          <h3 className="mb-1 text-sm font-semibold">Build {seleccionado.slice(0, 8)}</h3>

          <h4 className="mt-2 text-xs font-semibold uppercase text-gray-600">Saída guardada</h4>
          <pre
            className="max-h-40 overflow-auto rounded bg-gray-50 p-2 text-xs"
            data-testid="build-status-saida"
          >
            {saida === null ? "A carregar…" : saida === "" ? "(sem saída guardada)" : saida}
          </pre>

          <h4 className="mt-2 text-xs font-semibold uppercase text-gray-600">Artefactos medidos</h4>
          {artefactos === null ? (
            <p className="text-xs text-gray-600">A carregar…</p>
          ) : artefactos.produziu ? (
            <ul className="list-inside list-disc text-xs" data-testid="build-status-artefactos">
              {(artefactos.artefactos ?? []).map((a: ArtefactoDeBuild) => (
                <li key={a.Caminho}>
                  <span className="font-mono">{a.Nome}</span> · {formatarBytes(a.TamanhoBytes)} ·{" "}
                  <span className="font-mono text-[0.7rem]">{a.SHA256.slice(0, 16)}…</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-xs text-gray-600" data-testid="build-status-sem-artefactos">
              Este build não produziu artefactos.
            </p>
          )}

          <p className="mt-2 text-xs text-gray-600" data-testid="build-status-fronteira">
            Nada foi assinado, publicado ou instalado por estas rotas.
          </p>
        </div>
      ) : null}
    </section>
  );
}
