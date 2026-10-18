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
import {
  descreverAssinatura,
  obterChecksum,
  resumirVerificacao,
  tamanhoLegivel,
  verificarArtefacto,
  type ChecksumDeArtefacto,
  type VerificacaoDeArtefacto,
} from "@/lib/artifactClient";

type Estado = "a-carregar" | "pronto" | "erro";

/** Estado da seccao de verificacao de artefacto. Separado do estado do plano. */
type EstadoDoArtefacto = "inativo" | "a-medir" | "pronto" | "erro";

export function BuildPlanPanel() {
  const [repo, setRepo] = useState<string>("");
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
      setErroDoArtefacto(mensagemDeFalha(falha));
      setEstadoDoArtefacto("erro");
    }
  }, [caminhoDoArtefacto]);

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
      setErroDoArtefacto(mensagemDeFalha(falha));
      setEstadoDoArtefacto("erro");
    }
  }, [caminhoDoArtefacto, shaEsperado]);

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

      {/*
        FASE 28.7/28.8: verificacao de um artefacto JA' produzido.
        A seccao fica DENTRO deste painel, e nao num painel paralelo -- *nao crie painel paralelo*.

        A E-1258 criou as rotas e provou-as com 8 testes, mas nenhum painel as chamava. Este bloco
        fecha esse elo: *tres pecas provadas em separado nao sao uma integracao provada.*
      */}
      <section aria-labelledby="build-plan-artefacto">
        <h3 id="build-plan-artefacto">Verificar um artefacto</h3>
        <p>
          Mede o SHA-256 de um ficheiro que <strong>já existe</strong> e, se der a referência,
          compara. Não constrói, não assina e não instala. O caminho tem de ser{" "}
          <strong>absoluto</strong>.
        </p>

        <label htmlFor="build-artefacto-caminho">Caminho do artefacto</label>
        <input
          id="build-artefacto-caminho"
          type="text"
          value={caminhoDoArtefacto}
          onChange={(evento) => setCaminhoDoArtefacto(evento.target.value)}
          placeholder="ex.: C:\caminho\hades.exe"
          autoComplete="off"
          spellCheck={false}
        />

        <label htmlFor="build-artefacto-sha256">SHA-256 esperado (opcional para medir)</label>
        <input
          id="build-artefacto-sha256"
          type="text"
          value={shaEsperado}
          onChange={(evento) => setShaEsperado(evento.target.value)}
          placeholder="64 caracteres hexadecimais"
          autoComplete="off"
          spellCheck={false}
        />

        <button type="button" onClick={() => void medir()} disabled={!caminhoDoArtefacto.trim()}>
          Medir
        </button>
        <button
          type="button"
          onClick={() => void verificar()}
          disabled={!caminhoDoArtefacto.trim() || !shaEsperado.trim()}
        >
          Verificar
        </button>

        <div aria-live="polite">
          {estadoDoArtefacto === "a-medir" && <p>A medir o artefacto…</p>}
        </div>

        {erroDoArtefacto && <p role="alert">{erroDoArtefacto}</p>}

        {medicao && (
          <dl aria-label="Checksum medido">
            <dt>Ficheiro</dt>
            <dd>{medicao.caminho}</dd>
            <dt>Tamanho</dt>
            <dd>{tamanhoLegivel(medicao.tamanho_bytes)}</dd>
            <dt>SHA-256</dt>
            <dd>
              <code>{medicao.sha256}</code>
            </dd>
          </dl>
        )}

        {verificacao && (
          <>
            <dl aria-label="Resultado da verificação">
              <dt>Veredito</dt>
              <dd>{resumirVerificacao(verificacao.resultado)}</dd>
              <dt>SHA-256 medido</dt>
              <dd>
                <code>{verificacao.resultado.sha256_medido}</code>
              </dd>
              <dt>SHA-256 esperado</dt>
              <dd>
                <code>{verificacao.resultado.sha256_esperado}</code>
              </dd>
              <dt>Assinatura</dt>
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
function mensagemDeFalha(falha: unknown): string {
  if (falha instanceof Error && falha.message.trim()) return falha.message;
  return "não foi possível medir o artefacto";
}
