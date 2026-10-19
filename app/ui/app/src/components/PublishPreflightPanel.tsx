/**
 * FASE 28.8 / E-1298: o painel de `publish.preflight`.
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 *  1. **Nao publica.** O provider tem UM metodo -- `Preflight` -- e nao tem `Publicar()`. Nao ha'
 *     botao de publicar porque nao ha' rota que publique.
 *  2. **Nao escolhe o artefacto.** O caminho vem de `HADES_ARTEFACTO_A_PUBLICAR` no servidor. Um
 *     painel que tem um campo para o caminho acaba por receber um caminho que o servidor nao autoriza.
 *  3. **Nao escolhe a credencial.** Vem de `HADES_CREDENCIAL_DE_PUBLICACAO`. Nem o nome ela ve^.
 *  4. **Nao trata `LOCAL_ONLY` como publicacao.** E' ensaio local: `publico: false`, nada sai da
 *     maquina.
 *
 *	*Um painel que tem um campo para o segredo acaba por o receber.*
 *	*Publicar localmente e publicar publicamente nao sao a mesma coisa, e um painel que os mostra
 *	iguais ensina o utilizador a confundir os dois.*
 */

import { useCallback, useState } from "react";

import {
  canaisDePublicacao,
  canalEhPublico,
  contarCondicoes,
  descreverCanal,
  fetchPreflightDePublicacao,
  resumirPreflight,
  type CanalDePublicacao,
  type RespostaDoPreflightDePublicacao,
} from "@/lib/publishPreflightClient";

type Estado = "inativo" | "a-verificar" | "pronto" | "erro";

function mensagemDeFalha(falha: unknown): string {
  if (falha instanceof Error && falha.message.trim() !== "") {
    return falha.message;
  }
  return "nao foi possivel verificar o canal";
}

export function PublishPreflightPanel() {
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState("");
  const [canal, setCanal] = useState<CanalDePublicacao>("LOCAL_ONLY");
  const [resposta, setResposta] = useState<RespostaDoPreflightDePublicacao | null>(null);

  const verificar = useCallback(async () => {
    setEstado("a-verificar");
    setErro("");
    setResposta(null);
    try {
      setResposta(await fetchPreflightDePublicacao(canal));
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, [canal]);

  const preflight = resposta?.preflight;
  const contagem = preflight ? contarCondicoes(preflight) : { satisfeitas: 0, total: 0 };

  return (
    <section
      aria-labelledby="preflight-titulo"
      data-testid="preflight"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="preflight-titulo" className="text-lg font-semibold">
        Preflight de publicacao
      </h2>

      {/* O que a rota faz ANTES do botao -- e o que ela nao faz. */}
      <p className="mt-1 text-sm text-neutral-400">
        Pergunta ao servidor se um canal esta' pronto. <strong>Nao publica.</strong> O provider tem um
        unico metodo -- <code>Preflight</code> -- e nao tem <code>Publicar()</code>: nao ha' botao de
        publicar porque nao ha' rota que publique.
      </p>
      <p className="mt-1 text-xs text-neutral-500" data-testid="preflight-sem-campos">
        O caminho do artefacto (<code>HADES_ARTEFACTO_A_PUBLICAR</code>) e a credencial
        (<code>HADES_CREDENCIAL_DE_PUBLICACAO</code>) vem do{" "}
        <strong>ambiente do servidor</strong>, nunca deste painel. Este painel escolhe o canal, e mais
        nada.
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <label htmlFor="preflight-canal" className="text-sm">
          Canal
        </label>
        <select
          id="preflight-canal"
          data-testid="preflight-select-canal"
          className="rounded border border-neutral-600 bg-transparent px-2 py-1 text-sm"
          value={canal}
          onChange={(evento) => setCanal(evento.target.value as CanalDePublicacao)}
          disabled={estado === "a-verificar"}
        >
          {canaisDePublicacao().map((opcao) => (
            <option key={opcao} value={opcao}>
              {descreverCanal(opcao)}
            </option>
          ))}
        </select>
        <button
          type="button"
          data-testid="preflight-verificar"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void verificar()}
          disabled={estado === "a-verificar"}
        >
          Verificar canal
        </button>
      </div>

      {/* Aviso permanente para canal publico: a aprovacao humana nao e' opcional. */}
      {canalEhPublico(canal) ? (
        <p className="mt-2 text-xs text-amber-500" data-testid="preflight-canal-publico">
          Canal publico: mesmo com o preflight pronto, a publicacao exige confirmacao humana no momento
          da accao. Este painel nunca a substitui.
        </p>
      ) : (
        <p className="mt-2 text-xs text-neutral-400" data-testid="preflight-canal-local">
          Canal <strong>somente local</strong>: e' um ensaio. Nada sai desta maquina, e um preflight
          pronto aqui <strong>nao</strong> significa pronto para publicar.
        </p>
      )}

      {estado === "a-verificar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="preflight-a-correr">
          A verificar o canal...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="preflight-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && resposta !== null && preflight !== undefined ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="preflight-resultado">
          <p className="text-sm font-medium" data-testid="preflight-resumo">
            {resumirPreflight(preflight)}
          </p>

          {/* As DUAS garantias vem separadas do veredicto e sao mostradas como tal. */}
          <div className="text-xs" data-testid="preflight-garantias">
            <p className="text-neutral-400">
              Garantias declaradas pelo servidor:
            </p>
            <ul className="list-inside list-disc">
              <li data-testid="preflight-somente-leitura">
                somente leitura: {resposta.somente_leitura ? "sim" : "NAO"}
              </li>
              <li data-testid="preflight-nada-publicado">
                nada publicado: {resposta.nada_publicado ? "sim" : "NAO"}
              </li>
              <li data-testid="preflight-publicado">
                publicado: {preflight.publicado ? "SIM (inconsistente)" : "nao"}
              </li>
            </ul>
          </div>

          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Canal</dt>
            <dd className="font-mono" data-testid="preflight-canal-escolhido">
              {preflight.canal}
            </dd>
            <dt className="text-neutral-400">Publico</dt>
            <dd data-testid="preflight-publico">{preflight.publico ? "sim" : "nao"}</dd>
            <dt className="text-neutral-400">Pronto</dt>
            <dd data-testid="preflight-pronto">{preflight.pronto ? "sim" : "nao"}</dd>
            <dt className="text-neutral-400">Requer aprovacao</dt>
            <dd data-testid="preflight-requer-aprovacao">
              {preflight.requer_aprovacao ? "sim" : "nao"}
            </dd>
            <dt className="text-neutral-400">Condicoes</dt>
            <dd data-testid="preflight-condicoes-contagem">
              {contagem.satisfeitas}/{contagem.total}
            </dd>
          </dl>

          {preflight.condicoes.length > 0 ? (
            <div className="text-xs" data-testid="preflight-condicoes">
              <p className="text-neutral-400">Condicoes:</p>
              <ul className="list-inside list-disc">
                {preflight.condicoes.map((condicao) => (
                  <li
                    key={condicao.nome}
                    className={condicao.satisfeita ? "" : "text-red-400"}
                    data-testid={`preflight-condicao-${condicao.nome}`}
                  >
                    {condicao.satisfeita ? "OK" : "PENDENTE"} {condicao.nome}
                    {condicao.detalhe.trim() !== "" ? ` -- ${condicao.detalhe}` : ""}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}

          {preflight.explicacao.trim() !== "" ? (
            <p className="text-xs text-neutral-400" data-testid="preflight-explicacao">
              {preflight.explicacao}
            </p>
          ) : null}

          {preflight.aviso.trim() !== "" ? (
            <p className="text-xs text-amber-500" data-testid="preflight-aviso">
              {preflight.aviso}
            </p>
          ) : null}

          <p className="text-xs text-neutral-400" data-testid="preflight-aviso-fronteira">
            {resposta.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
