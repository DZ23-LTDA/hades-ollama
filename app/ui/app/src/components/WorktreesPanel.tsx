/**
 * FASE 2 do adendo V3: o painel de WORKTREES dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL MOSTRA
 *
 * O inventario real do repositorio -- incluindo os worktrees que o HADES NAO criou -- e, para uma
 * missao escolhida, o estado do ciclo de vida com o que pode acontecer a seguir.
 *
 * AS TRES DECISOES QUE ESTE PAINEL TORNA VISIVEIS
 *
 *  1. **Ver antes de apagar.** Worktrees externos aparecem marcados como tal.
 *     *Um worktree externo que nao se ve' e' um worktree que se apaga por engano.*
 *
 *  2. **O estado e' fechado e o painel nao inventa.** Um estado fora da lista aparece como
 *     "Estado desconhecido" -- *um painel que inventa um rotulo para um estado que nao conhece
 *     ensina o utilizador a confiar no que nao foi verificado.*
 *
 *  3. **Diagnostico antes da accao.** O painel mostra `pode_remover` e o MOTIVO. Nao ha' botao de
 *     remover: *remover exige uma decisao sobre trabalho nao integrado, e essa decisao nao cabe
 *     num clique que o painel nao sabe explicar.*
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 * Nao cria, nao suspende, nao remove. So' le'. As rotas que consome sao GET, e o backend recusa
 * POST por teste.
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, tabela com cabecalhos, `aria-expanded` nos botoes de missao, e mensagens de
 * erro em `role="alert"`.
 */

import { useCallback, useEffect, useState } from "react";

import {
  corDoEstadoDoWorktree,
  fetchWorktreeDiagnostico,
  fetchWorktreeInventario,
  nomeCurtoDoCaminho,
  rotuloDoEstadoDoWorktree,
  type WorktreeDiagnostico,
  type WorktreeInventario,
} from "@/lib/worktreeClient";

type Estado = "a-carregar" | "pronto" | "erro";

export function WorktreesPanel() {
  const [repo, setRepo] = useState<string>("");
  const [inventario, setInventario] = useState<WorktreeInventario | null>(null);
  const [diagnostico, setDiagnostico] = useState<WorktreeDiagnostico | null>(null);
  const [missaoActiva, setMissaoActiva] = useState<string>("");
  const [estado, setEstado] = useState<Estado>("a-carregar");
  const [erro, setErro] = useState<string>("");

  const carregar = useCallback(async (alvo: string) => {
    setEstado("a-carregar");
    setErro("");
    try {
      const dados = await fetchWorktreeInventario(alvo);
      setInventario(dados);
      setEstado("pronto");
    } catch (e) {
      setErro(e instanceof Error ? e.message : String(e));
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    // O `repo` inicial e' vazio: o painel pede o caminho em vez de adivinhar o checkout.
    //	*Inventariar o checkout errado e' pior do que pedir o caminho.*
    if (repo.trim() === "") {
      setEstado("pronto");
      return;
    }
    void carregar(repo);
  }, [repo, carregar]);

  const diagnosticar = useCallback(
    async (missao: string) => {
      setMissaoActiva(missao);
      setDiagnostico(null);
      try {
        const d = await fetchWorktreeDiagnostico(repo, missao);
        setDiagnostico(d);
      } catch (e) {
        setErro(e instanceof Error ? e.message : String(e));
      }
    },
    [repo],
  );

  const missoes = (inventario?.lista ?? []).filter((w) => w.criado_pelo_hades && w.branch);

  return (
    <section aria-labelledby="worktrees-titulo" className="mt-10">
      <h2 id="worktrees-titulo" className="text-lg font-semibold">
        Worktrees
      </h2>
      <p className="mt-1 text-sm text-neutral-400">
        Inventario do repositorio e estado do ciclo de vida. Só leitura: remover exige uma decisão
        sobre trabalho não integrado.
      </p>

      <div className="mt-4">
        <label htmlFor="worktree-repo" className="block text-sm font-medium">
          Raiz do repositório
        </label>
        <input
          id="worktree-repo"
          type="text"
          value={repo}
          onChange={(ev) => setRepo(ev.target.value)}
          placeholder="/caminho/para/repositorio"
          className="mt-1 w-full rounded border border-neutral-700 bg-neutral-900 px-3 py-2 text-sm"
        />
      </div>

      {estado === "a-carregar" && (
        <p className="mt-4 text-sm text-neutral-400" role="status">
          A carregar worktrees…
        </p>
      )}

      {estado === "erro" && (
        <p className="mt-4 text-sm text-rose-300" role="alert">
          {erro}
        </p>
      )}

      {estado === "pronto" && repo.trim() === "" && (
        <p className="mt-4 text-sm text-neutral-400">
          Indique a raiz do repositório para ver os worktrees. O painel não adivinha o checkout.
        </p>
      )}

      {estado === "pronto" && inventario && (
        <>
          <p className="mt-4 text-sm text-neutral-300">{inventario.resumo}</p>

          {inventario.avisos.length > 0 && (
            <ul className="mt-2 space-y-1" aria-label="Avisos do inventário">
              {inventario.avisos.map((a) => (
                <li key={a} className="text-sm text-amber-300" role="alert">
                  {a}
                </li>
              ))}
            </ul>
          )}

          <table className="mt-4 w-full text-left text-sm">
            <caption className="sr-only">
              Worktrees do repositório indicado, com origem e branch
            </caption>
            <thead>
              <tr className="border-b border-neutral-800">
                <th scope="col" className="py-2">Worktree</th>
                <th scope="col" className="py-2">Branch</th>
                <th scope="col" className="py-2">Origem</th>
              </tr>
            </thead>
            <tbody>
              {inventario.lista.map((w) => (
                <tr key={w.caminho_resolvido} className="border-b border-neutral-900">
                  <td className="py-2">
                    {nomeCurtoDoCaminho(w.caminho_resolvido)}
                    {w.principal && <span className="ml-2 text-xs text-neutral-500">principal</span>}
                  </td>
                  <td className="py-2">{w.detached ? "(destacado)" : w.branch || "—"}</td>
                  <td className="py-2">
                    {w.criado_pelo_hades ? "HADES" : "externo"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {missoes.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-medium">Diagnóstico por missão</h3>
              <ul className="mt-2 space-y-2" aria-label="Missões com worktree">
                {missoes.map((w) => {
                  const missao = nomeCurtoDoCaminho(w.caminho_resolvido);
                  const activa = missaoActiva === missao;
                  return (
                    <li key={w.caminho_resolvido}>
                      <button
                        type="button"
                        aria-expanded={activa}
                        onClick={() => void diagnosticar(missao)}
                        className="rounded border border-neutral-700 px-3 py-1 text-sm"
                      >
                        {missao}
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}

          {diagnostico && (
            <div className="mt-4 rounded border border-neutral-800 p-3" aria-label="Diagnóstico">
              <p className="text-sm">
                Estado:{" "}
                <span className={corDoEstadoDoWorktree(diagnostico.estado)}>
                  {rotuloDoEstadoDoWorktree(diagnostico.estado)}
                </span>
              </p>

              {!diagnostico.estado_valido && (
                <p className="mt-1 text-sm text-rose-300" role="alert">
                  O backend devolveu um estado fora da lista fechada.
                </p>
              )}

              <p className="mt-1 text-sm text-neutral-400">{diagnostico.estado_observado}</p>

              {diagnostico.inconsistente && (
                <p className="mt-2 text-sm text-rose-300" role="alert">
                  Inconsistência: {diagnostico.inconsistente}
                </p>
              )}

              {diagnostico.tem_trabalho_nao_integrado && (
                <p className="mt-2 text-sm text-amber-300" role="alert">
                  Trabalho não integrado: {diagnostico.commits_nao_integrados} commit(s),{" "}
                  {diagnostico.ficheiros_modificados} ficheiro(s) por commitar.
                </p>
              )}

              <p className="mt-2 text-sm">
                {diagnostico.pode_remover
                  ? "Pode ser removido sem perda."
                  : `Não pode ser removido: ${diagnostico.motivo_remover ?? "motivo não indicado"}`}
              </p>

              {diagnostico.proximos.length > 0 && (
                <p className="mt-1 text-sm text-neutral-400">
                  A seguir: {diagnostico.proximos.map(rotuloDoEstadoDoWorktree).join(", ")}
                </p>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}
