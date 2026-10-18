/**
 * FASE 28.8 do adendo V3: o painel de BUILD & RELEASE dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL MOSTRA
 *
 * Para um projecto, o que ele PEDE (stacks, alvos) e o que FALTA nesta maquina (blockers, motivos),
 * tal como o backend o mediu. Nao ha' numeros calculados no frontend.
 *
 *	*Um painel que calcula o seu proprio health mostra o que o programador achou que ia acontecer,
 *	nao o que aconteceu.*
 *
 * AS TRES DECISOES QUE ESTE PAINEL TORNA VISIVEIS
 *
 *  1. **"Pronto" nao e' "vai construir".** O cabecalho diz que a rota so' DETECTA e SONDA. O
 *     utilizador ve' o aviso do backend, nao um botao de build que nao existe.
 *
 *  2. **Projecto sem alvos e' um resultado, nao um erro.** Aparece uma mensagem propria. *Um
 *     projecto vazio e um projecto irreconhecivel nao sao a mesma coisa, e o painel tem de os
 *     distinguir tal como o backend.*
 *
 *  3. **Blockers vem no topo.** Nao ha' obrigacao de abrir cada alvo. *Um relatorio que obriga a
 *     somar a mao e' um relatorio para quem ja' sabia a resposta.*
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 * Nao instala ferramentas, nao constroi, nao assina, nao publica. As rotas que consome sao GET.
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, tabela com cabecalhos, `aria-live` no estado de carregamento e erro em
 * `role="alert"`. Os estados das sondas sao traduzidos para texto humano.
 */

import { useCallback, useEffect, useState } from "react";

import {
  descreverEstadoDaSonda,
  obterPlanoDeRelease,
  resumirSondas,
  type PlanoDeRelease,
} from "@/lib/buildPlanClient";

type Estado = "a-carregar" | "pronto" | "erro";

export function BuildPlanPanel() {
  const [repo, setRepo] = useState<string>("");
  const [plano, setPlano] = useState<PlanoDeRelease | null>(null);
  const [estado, setEstado] = useState<Estado>("a-carregar");
  const [erro, setErro] = useState<string>("");

  const carregar = useCallback(async (alvo: string) => {
    if (!alvo.trim()) {
      setEstado("a-carregar");
      setPlano(null);
      setErro("");
      return;
    }
    setEstado("a-carregar");
    setErro("");
    try {
      const resposta = await obterPlanoDeRelease(alvo.trim());
      setPlano(resposta);
      setEstado("pronto");
    } catch (causa) {
      setPlano(null);
      setErro(causa instanceof Error ? causa.message : String(causa));
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar(repo);
  }, [carregar, repo]);

  const resumo = plano ? resumirSondas(plano.plano) : null;

  return (
    <section aria-labelledby="build-plan-titulo">
      <h2 id="build-plan-titulo">Build &amp; Release</h2>

      <p>
        Este painel apenas <strong>detecta</strong> o projeto e <strong>sonda</strong> as ferramentas
        instaladas. Não instala nada, não constrói nada e não assina nada.
      </p>

      <label htmlFor="build-plan-repo">Caminho do projeto</label>
      <input
        id="build-plan-repo"
        type="text"
        value={repo}
        onChange={(evento) => setRepo(evento.target.value)}
        placeholder="ex.: C:\caminho\do\projeto"
        autoComplete="off"
        spellCheck={false}
      />

      <div aria-live="polite">
        {estado === "a-carregar" && <p>A analisar o projeto…</p>}
      </div>

      {estado === "erro" && (
        <p role="alert">
          Não foi possível analisar o projeto: {erro}
        </p>
      )}

      {estado === "pronto" && plano && (
        <>
          {plano.aviso_de_fronteira && <p>{plano.aviso_de_fronteira}</p>}

          <dl>
            <dt>Projeto</dt>
            <dd>{plano.repo}</dd>
            <dt>Stacks</dt>
            <dd>{plano.plano.projeto.stacks.join(", ") || "nenhuma reconhecida"}</dd>
            <dt>Gestores de pacotes</dt>
            <dd>{plano.plano.projeto.gestores.join(", ") || "nenhum detetado"}</dd>
            <dt>Alvos</dt>
            <dd>{plano.plano.projeto.alvos.join(", ") || "nenhum"}</dd>
            {resumo && (
              <>
                <dt>Ferramentas</dt>
                <dd>
                  {resumo.prontas} de {resumo.total} prontas
                  {resumo.emFalta > 0 ? `, ${resumo.emFalta} em falta` : ""}
                </dd>
              </>
            )}
          </dl>

          {plano.plano.sem_alvos ? (
            <p>
              O projeto não declara nenhum alvo que o HADES conheça. Não há plano de build a
              construir — isto não é um erro, é um resultado.
            </p>
          ) : (
            <p>{plano.plano.explicacao}</p>
          )}

          {plano.blockers.length > 0 && (
            <section aria-labelledby="build-plan-blockers">
              <h3 id="build-plan-blockers">O que falta</h3>
              <ul>
                {plano.blockers.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            </section>
          )}

          {plano.motivos.length > 0 && (
            <section aria-labelledby="build-plan-motivos">
              <h3 id="build-plan-motivos">Porquê</h3>
              <ul>
                {plano.motivos.map((motivo) => (
                  <li key={motivo}>{motivo}</li>
                ))}
              </ul>
            </section>
          )}

          {plano.plano.trabalho_independente &&
            plano.plano.trabalho_independente.length > 0 && (
              <section aria-labelledby="build-plan-independente">
                <h3 id="build-plan-independente">O que pode avançar sem desbloquear nada</h3>
                <ul>
                  {plano.plano.trabalho_independente.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </section>
            )}

          <table>
            <caption>Plano por alvo</caption>
            <thead>
              <tr>
                <th scope="col">Alvo</th>
                <th scope="col">Estado</th>
                <th scope="col">O que falta</th>
              </tr>
            </thead>
            <tbody>
              {plano.plano.alvos.map((entrada) => {
                const pronto = plano.plano.alvos_prontos.includes(entrada.alvo);
                return (
                  <tr key={entrada.alvo}>
                    <th scope="row">{entrada.alvo}</th>
                    <td>
                      {entrada.erro_de_plano
                        ? "não foi possível planear"
                        : pronto
                          ? "pronto"
                          : "bloqueado"}
                    </td>
                    <td>
                      {entrada.erro_de_plano
                        ? entrada.erro_de_plano
                        : (entrada.plano.Blockers ?? []).join(", ") || "nada"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          <details>
            <summary>Detalhe das ferramentas sondadas</summary>
            <table>
              <caption>Ferramentas por alvo</caption>
              <thead>
                <tr>
                  <th scope="col">Alvo</th>
                  <th scope="col">Ferramenta</th>
                  <th scope="col">Estado</th>
                  <th scope="col">Detalhe</th>
                </tr>
              </thead>
              <tbody>
                {plano.plano.alvos.flatMap((entrada) =>
                  (entrada.plano.Probes ?? []).map((sonda) => (
                    <tr key={`${entrada.alvo}-${sonda.Requisito.Nome}`}>
                      <td>{entrada.alvo}</td>
                      <th scope="row">{sonda.Requisito.Nome}</th>
                      <td>{descreverEstadoDaSonda(sonda.Estado)}</td>
                      <td>{sonda.Detalhe ?? sonda.Requisito.Motivo}</td>
                    </tr>
                  )),
                )}
              </tbody>
            </table>
          </details>

          {plano.plano.avisos && plano.plano.avisos.length > 0 && (
            <section aria-labelledby="build-plan-avisos">
              <h3 id="build-plan-avisos">Avisos da deteção</h3>
              <ul>
                {plano.plano.avisos.map((aviso) => (
                  <li key={aviso}>{aviso}</li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </section>
  );
}
