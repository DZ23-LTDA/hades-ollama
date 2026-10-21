/**
 * FASE 28.8 do adendo V3: o painel de SMOKE dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL FAZ, E O QUE ELE DEIXA CLARO
 *
 * Executa um artefacto **ja' produzido** e mostra o que aconteceu. Nao constroi, nao assina, nao
 * instala. A rota que consome e' POST e executa codigo -- por isso o painel tem tres obrigacoes:
 *
 *  1. **dizer que executa** antes de o utilizador carregar no botao;
 *  2. **mostrar o aviso de sandbox do backend por inteiro** -- *um aviso que fica na documentacao nao
 *     chega a quem correu o binario*;
 *  3. **nao chamar "funciona" ao que so' arrancou** -- *um binario que arranca nao e' um binario que
 *     funciona: e' um binario que arranca*.
 *
 * O QUE ELE NAO TEM
 *
 * Nao tem campo para a raiz dos artefactos: a raiz vem de `HADES_ARTEFACTOS` no servidor. *Um cliente
 * que escolhe a raiz dos binarios e' um cliente que decide o que corre.*
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, rotulos em cada campo, `aria-live` no estado de execucao, erro em `role="alert"` e
 * o resultado em `aria-live="polite"` para um leitor de tela o anunciar.
 */

import { useCallback, useState } from "react";

import {
  executarSmoke,
  provaDeFuncionamento,
  resumirSmoke,
  separarArgumentos,
  type ResultadoDeSmoke,
} from "@/lib/smokeClient";
import { mensagemDaFalhaCom } from "../lib/mensagemDeFalha";

type Estado = "inativo" | "a-executar" | "pronto" | "erro";

/** Extrai a mensagem de uma falha sem inventar uma. */
function mensagemDeFalha(falha: unknown): string {
  return mensagemDaFalhaCom(falha, "nao foi possivel executar o smoke");
}

export function SmokePanel() {
  const [caminho, setCaminho] = useState<string>("");
  const [linhaDeArgumentos, setLinhaDeArgumentos] = useState<string>("");
  const [saidaEsperada, setSaidaEsperada] = useState<string>("");
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState<string>("");
  const [resultado, setResultado] = useState<ResultadoDeSmoke | null>(null);

  const caminhoLimpo = caminho.trim();

  const executar = useCallback(async () => {
    if (caminhoLimpo === "") {
      return;
    }
    setEstado("a-executar");
    setErro("");
    setResultado(null);
    try {
      const pedido: Parameters<typeof executarSmoke>[0] = {
        caminho: caminhoLimpo,
        argumentos: separarArgumentos(linhaDeArgumentos),
      };
      if (saidaEsperada.trim() !== "") {
        pedido.saida_esperada = saidaEsperada.trim();
      }
      setResultado(await executarSmoke(pedido));
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, [caminhoLimpo, linhaDeArgumentos, saidaEsperada]);

  return (
    <section
      aria-labelledby="smoke-titulo"
      data-testid="smoke"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="smoke-titulo" className="text-lg font-semibold">
        Executar um smoke
      </h2>
      {/* Dizer que executa ANTES do botao: o utilizador tem de saber o que vai acontecer. */}
      <p className="mt-1 text-sm text-neutral-400">
        Este painel <strong>executa</strong> um binario ja' produzido e mostra o que aconteceu.
        Nao constroi, nao assina e nao instala. O caminho tem de estar dentro da raiz autorizada do
        servidor -- o painel nao escolhe a raiz.
      </p>

      <div className="mt-3 flex flex-col gap-2">
        <label className="flex flex-col gap-1 text-sm">
          <span>Caminho do artefacto</span>
          <input
            id="smoke-caminho"
            type="text"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={caminho}
            onChange={(evento) => setCaminho(evento.target.value)}
            placeholder="caminho absoluto dentro da raiz autorizada"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>Argumentos (separados por espacos)</span>
          <input
            id="smoke-argumentos"
            type="text"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={linhaDeArgumentos}
            onChange={(evento) => setLinhaDeArgumentos(evento.target.value)}
            placeholder="--version"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>Saida esperada (opcional)</span>
          <input
            id="smoke-saida-esperada"
            type="text"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={saidaEsperada}
            onChange={(evento) => setSaidaEsperada(evento.target.value)}
            placeholder="texto que a saida tem de conter"
          />
        </label>
        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void executar()}
          disabled={caminhoLimpo === "" || estado === "a-executar"}
        >
          Executar smoke
        </button>
      </div>

      {estado === "a-executar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="smoke-a-executar">
          A executar o artefacto...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="smoke-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && resultado !== null ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="smoke-resultado">
          <p className="text-sm font-medium" data-testid="smoke-veredicto">
            {resumirSmoke(resultado)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="smoke-prova">
            {provaDeFuncionamento(resultado)}
          </p>
          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Exit code</dt>
            <dd data-testid="smoke-codigo">{resultado.codigo_de_saida}</dd>
            <dt className="text-neutral-400">Duracao</dt>
            <dd data-testid="smoke-duracao">{resultado.duracao_ms} ms</dd>
            <dt className="text-neutral-400">Timeout</dt>
            <dd data-testid="smoke-timeout">{resultado.timeout ? "sim" : "nao"}</dd>
          </dl>
          {resultado.saida !== "" ? (
            <div>
              <p className="text-xs text-neutral-400">
                Saida{resultado.saida_truncada ? " (truncada)" : ""}
              </p>
              <pre
                className="mt-1 max-h-48 overflow-auto rounded bg-neutral-900 p-2 text-xs"
                data-testid="smoke-saida"
              >
                {resultado.saida}
              </pre>
            </div>
          ) : null}
          {/* O aviso de sandbox vem do backend e e' mostrado por inteiro. */}
          <p className="text-xs text-amber-400" data-testid="smoke-aviso-sandbox">
            {resultado.aviso_de_sandbox}
          </p>
        </div>
      ) : null}
    </section>
  );
}
