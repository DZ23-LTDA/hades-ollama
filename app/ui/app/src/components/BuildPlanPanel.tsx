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

import { useT, type Translator } from "@/i18n";

// FASE 28.7: o `build.start`. Importado do cliente de ESTADO porque e' la' que o
// contrato do build vive; o painel de plano nao duplica os tipos.
import { iniciarBuild } from "@/lib/buildStatusClient";

import {
  descreverEstadoDaSonda,
  obterPlanoDeRelease,
  resumirSondas,
  type PlanoDeRelease,
} from "@/lib/buildPlanClient";
import {
  descreverAssinatura,
  obterChecksum,
  resumirVerificacao,
  tamanhoLegivel,
  verificarArtefacto,
  type ChecksumDeArtefacto,
  type VerificacaoDeArtefacto,
} from "@/lib/artifactClient";
import { mensagemDaFalhaCom } from "../lib/mensagemDeFalha";

type Estado = "a-carregar" | "pronto" | "erro";

/** Estado da seccao de verificacao de artefacto. Separado do estado do plano. */
type EstadoDoArtefacto = "inativo" | "a-medir" | "pronto" | "erro";

export function BuildPlanPanel() {
  const { t } = useT();
  const [repo, setRepo] = useState<string>("");
  // FASE 28.7: o estado do INICIO de build. Separado do `aMedir` porque sao duas accoes
  // diferentes: medir um checksum e iniciar uma build de minutos.
  const [aIniciar, setAIniciar] = useState<string | null>(null);
  const [avisoDeInicio, setAvisoDeInicio] = useState<string | null>(null);
  const [erroDeInicio, setErroDeInicio] = useState<string | null>(null);
  const [plano, setPlano] = useState<PlanoDeRelease | null>(null);
  const [estado, setEstado] = useState<Estado>("a-carregar");
  const [erro, setErro] = useState<string>("");

  // FASE 28.7/28.8: verificacao de um artefacto ja' produzido.
  const [caminhoDoArtefacto, setCaminhoDoArtefacto] = useState<string>("");
  const [shaEsperado, setShaEsperado] = useState<string>("");
  const [estadoDoArtefacto, setEstadoDoArtefacto] = useState<EstadoDoArtefacto>("inativo");
  const [erroDoArtefacto, setErroDoArtefacto] = useState<string>("");
  const [medicao, setMedicao] = useState<ChecksumDeArtefacto | null>(null);
  const [verificacao, setVerificacao] = useState<VerificacaoDeArtefacto | null>(null);

  // FASE 28.7: inicia uma build ASSINCRONA no alvo. O servidor responde 202 com `build_id`; o estado
  // e a saida aparecem no painel «Estado dos builds».
  //
  //	*Um `build.start` sincrono e' um pedido que so' responde quando ja' nao ha' nada a cancelar.*
  const iniciarNoAlvo = useCallback(
    async (alvo: string) => {
      if (!plano) return;
      setAIniciar(alvo);
      setAvisoDeInicio(null);
      setErroDeInicio(null);
      try {
        const resposta = await iniciarBuild({ raiz: plano.repo, alvo });
        const sufixo = resposta.build_id ? ` (${resposta.build_id.slice(0, 8)})` : "";
        setAvisoDeInicio(
          t("buildPlan.start.accepted", { target: alvo, suffix: sufixo }),
        );
      } catch (e) {
        // 412 e' um RESULTADO (plano incompleto), nao uma falha do servidor. O painel diz o que falta
        // em vez de "erro de rede".
        const mensagem = e instanceof Error ? e.message : String(e);
        setErroDeInicio(
          mensagem.includes("412")
            ? t("buildPlan.start.incompletePlan", { target: alvo, message: mensagem })
            : mensagem,
        );
      } finally {
        setAIniciar(null);
      }
    },
    [plano, t],
  );

  const medir = useCallback(async () => {
    const caminho = caminhoDoArtefacto.trim();
    if (!caminho) return;
    setEstadoDoArtefacto("a-medir");
    setErroDoArtefacto("");
    setMedicao(null);
    setVerificacao(null);
    try {
      setMedicao(await obterChecksum(caminho));
      setEstadoDoArtefacto("pronto");
    } catch (falha) {
      setErroDoArtefacto(mensagemDeFalha(t, falha));
      setEstadoDoArtefacto("erro");
    }
  }, [caminhoDoArtefacto, t]);

  const verificar = useCallback(async () => {
    const caminho = caminhoDoArtefacto.trim();
    const referencia = shaEsperado.trim();
    // *Sem referencia isto seria uma medicao, nao uma verificacao.* O botao ja' esta' desactivado,
    // mas a guarda fica aqui para que um `disabled` removido por CSS nao invente uma verificacao.
    if (!caminho || !referencia) return;
    setEstadoDoArtefacto("a-medir");
    setErroDoArtefacto("");
    setMedicao(null);
    setVerificacao(null);
    try {
      setVerificacao(await verificarArtefacto(caminho, referencia));
      setEstadoDoArtefacto("pronto");
    } catch (falha) {
      setErroDoArtefacto(mensagemDeFalha(t, falha));
      setEstadoDoArtefacto("erro");
    }
  }, [caminhoDoArtefacto, shaEsperado, t]);

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
      <h2 id="build-plan-titulo">{t("buildPlan.title")}</h2>

      <p>
        {t("buildPlan.intro.before")} <strong>{t("buildPlan.intro.detect")}</strong>{" "}
        {t("buildPlan.intro.middle")} <strong>{t("buildPlan.intro.probe")}</strong>{" "}
        {t("buildPlan.intro.after")}
      </p>

      <label htmlFor="build-plan-repo">{t("buildPlan.repo.label")}</label>
      <input
        id="build-plan-repo"
        type="text"
        value={repo}
        onChange={(evento) => setRepo(evento.target.value)}
        placeholder={t("buildPlan.repo.placeholder")}
        autoComplete="off"
        spellCheck={false}
      />

      <div aria-live="polite">
        {estado === "a-carregar" && <p>{t("buildPlan.loading")}</p>}
      </div>

      {estado === "erro" && (
        <p role="alert">
          {t("buildPlan.error.analyze")} {erro}
        </p>
      )}

      {estado === "pronto" && plano && (
        <>
          {plano.aviso_de_fronteira && <p>{plano.aviso_de_fronteira}</p>}

          <dl>
            <dt>{t("buildPlan.field.project")}</dt>
            <dd>{plano.repo}</dd>
            <dt>{t("buildPlan.field.stacks")}</dt>
            <dd>{plano.plano.projeto.stacks.join(", ") || t("buildPlan.stacks.none")}</dd>
            <dt>{t("buildPlan.field.packageManagers")}</dt>
            <dd>
              {plano.plano.projeto.gestores.join(", ") || t("buildPlan.packageManagers.none")}
            </dd>
            <dt>{t("buildPlan.field.targets")}</dt>
            <dd>{plano.plano.projeto.alvos.join(", ") || t("buildPlan.targets.none")}</dd>
            {resumo && (
              <>
                <dt>{t("buildPlan.field.tools")}</dt>
                <dd>
                  {t("buildPlan.tools.ready", { ready: resumo.prontas, total: resumo.total })}
                  {resumo.emFalta > 0
                    ? t("buildPlan.tools.missing", { count: resumo.emFalta })
                    : ""}
                </dd>
              </>
            )}
          </dl>

          {plano.plano.sem_alvos ? (
            <p>{t("buildPlan.noTargets")}</p>
          ) : (
            <p>{plano.plano.explicacao}</p>
          )}

          {plano.blockers.length > 0 && (
            <section aria-labelledby="build-plan-blockers">
              <h3 id="build-plan-blockers">{t("buildPlan.blockers.title")}</h3>
              <ul>
                {plano.blockers.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            </section>
          )}

          {plano.motivos.length > 0 && (
            <section aria-labelledby="build-plan-motivos">
              <h3 id="build-plan-motivos">{t("buildPlan.reasons.title")}</h3>
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
                <h3 id="build-plan-independente">{t("buildPlan.independent.title")}</h3>
                <ul>
                  {plano.plano.trabalho_independente.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </section>
            )}

          <div aria-live="polite">
            {avisoDeInicio ? (
              <p role="status" data-testid="build-plan-aviso-inicio">
                {avisoDeInicio}
              </p>
            ) : null}
            {erroDeInicio ? (
              <p role="alert" data-testid="build-plan-erro-inicio">
                {erroDeInicio}
              </p>
            ) : null}
          </div>
          <table>
            <caption>{t("buildPlan.plan.caption")}</caption>
            <thead>
              <tr>
                <th scope="col">{t("buildPlan.col.target")}</th>
                <th scope="col">{t("buildPlan.col.status")}</th>
                <th scope="col">{t("buildPlan.blockers.title")}</th>
                <th scope="col">{t("buildPlan.col.actions")}</th>
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
                        ? t("buildPlan.status.planFailed")
                        : pronto
                          ? t("buildPlan.status.ready")
                          : t("buildPlan.status.blocked")}
                    </td>
                    <td>
                      {entrada.erro_de_plano
                        ? entrada.erro_de_plano
                        : (entrada.plano.Blockers ?? []).join(", ") ||
                          t("buildPlan.blockers.none")}
                    </td>
                    <td>
                      {pronto ? (
                        <button
                          type="button"
                          onClick={() => void iniciarNoAlvo(entrada.alvo)}
                          disabled={aIniciar !== null}
                          data-testid={`build-plan-iniciar-${entrada.alvo}`}
                        >
                          {aIniciar === entrada.alvo
                            ? t("buildPlan.start.starting")
                            : t("buildPlan.start.action")}
                        </button>
                      ) : (
                        // Um alvo bloqueado NAO ganha um botao desactivado: um botao que nunca
                        // funciona ensina a ignorar botoes.
                        <span data-testid={`build-plan-sem-iniciar-${entrada.alvo}`}>—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          <details>
            <summary>{t("buildPlan.probes.details")}</summary>
            <table>
              <caption>{t("buildPlan.probes.caption")}</caption>
              <thead>
                <tr>
                  <th scope="col">{t("buildPlan.col.target")}</th>
                  <th scope="col">{t("buildPlan.col.tool")}</th>
                  <th scope="col">{t("buildPlan.col.status")}</th>
                  <th scope="col">{t("buildPlan.col.detail")}</th>
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
              <h3 id="build-plan-avisos">{t("buildPlan.warnings.title")}</h3>
              <ul>
                {plano.plano.avisos.map((aviso) => (
                  <li key={aviso}>{aviso}</li>
                ))}
              </ul>
            </section>
          )}
        </>
      )}

      {/*
        FASE 28.7/28.8: verificacao de um artefacto JA' produzido.
        A seccao fica DENTRO deste painel, e nao num painel paralelo -- *nao crie painel paralelo*.

        A E-1258 criou as rotas e provou-as com 8 testes, mas nenhum painel as chamava. Este bloco
        fecha esse elo: *tres pecas provadas em separado nao sao uma integracao provada.*
      */}
      <section aria-labelledby="build-plan-artefacto">
        <h3 id="build-plan-artefacto">{t("buildPlan.artifact.title")}</h3>
        <p>
          {t("buildPlan.artifact.intro.before")}{" "}
          <strong>{t("buildPlan.artifact.intro.exists")}</strong>{" "}
          {t("buildPlan.artifact.intro.after")}{" "}
          <strong>{t("buildPlan.artifact.intro.absolute")}</strong>.
        </p>

        <label htmlFor="build-artefacto-caminho">{t("buildPlan.artifact.path.label")}</label>
        <input
          id="build-artefacto-caminho"
          type="text"
          value={caminhoDoArtefacto}
          onChange={(evento) => setCaminhoDoArtefacto(evento.target.value)}
          placeholder={t("buildPlan.artifact.path.placeholder")}
          autoComplete="off"
          spellCheck={false}
        />

        <label htmlFor="build-artefacto-sha256">{t("buildPlan.artifact.sha.label")}</label>
        <input
          id="build-artefacto-sha256"
          type="text"
          value={shaEsperado}
          onChange={(evento) => setShaEsperado(evento.target.value)}
          placeholder={t("buildPlan.artifact.sha.placeholder")}
          autoComplete="off"
          spellCheck={false}
        />

        <button type="button" onClick={() => void medir()} disabled={!caminhoDoArtefacto.trim()}>
          {t("buildPlan.artifact.measure")}
        </button>
        <button
          type="button"
          onClick={() => void verificar()}
          disabled={!caminhoDoArtefacto.trim() || !shaEsperado.trim()}
        >
          {t("buildPlan.artifact.verify")}
        </button>

        <div aria-live="polite">
          {estadoDoArtefacto === "a-medir" && <p>{t("buildPlan.artifact.measuring")}</p>}
        </div>

        {erroDoArtefacto && <p role="alert">{erroDoArtefacto}</p>}

        {medicao && (
          <dl aria-label={t("buildPlan.artifact.checksumAria")}>
            <dt>{t("buildPlan.artifact.file")}</dt>
            <dd>{medicao.caminho}</dd>
            <dt>{t("buildPlan.artifact.size")}</dt>
            <dd>{tamanhoLegivel(medicao.tamanho_bytes)}</dd>
            <dt>SHA-256</dt>
            <dd>
              <code>{medicao.sha256}</code>
            </dd>
          </dl>
        )}

        {verificacao && (
          <>
            <dl aria-label={t("buildPlan.artifact.resultAria")}>
              <dt>{t("buildPlan.artifact.verdict")}</dt>
              <dd>{resumirVerificacao(verificacao.resultado)}</dd>
              <dt>{t("buildPlan.artifact.shaMeasured")}</dt>
              <dd>
                <code>{verificacao.resultado.sha256_medido}</code>
              </dd>
              <dt>{t("buildPlan.artifact.shaExpected")}</dt>
              <dd>
                <code>{verificacao.resultado.sha256_esperado}</code>
              </dd>
              <dt>{t("buildPlan.artifact.signature")}</dt>
              <dd>{descreverAssinatura(verificacao.resultado.assinatura)}</dd>
            </dl>
            {/*
              O aviso de fronteira vem do BACKEND e e' mostrado tal e qual. *
              Um painel que mostra "pronto" sem poder construir e' um painel que promete o que nao
              faz -- a menos que diga que so' le'.*
            */}
            <p>{verificacao.aviso_de_fronteira}</p>
          </>
        )}
      </section>
    </section>
  );
}

/**
 * Traduz uma falha do cliente para uma frase que o utilizador entende.
 *
 * *Um estado que mente e' pior do que nenhum* -- e uma mensagem "erro desconhecido" obriga quem le'
 * a adivinhar. O `agentFetch` ja' lanca com a mensagem do backend quando ha' corpo de erro.
 */
function mensagemDeFalha(t: Translator, falha: unknown): string {
  return mensagemDaFalhaCom(falha, t("buildPlan.artifact.measureFailed"));
}
