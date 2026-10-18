/**
 * FASE 28.8: o painel de verificacao de build.
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 *  1. **Nao assina.** A rota so' compara checksums.
 *  2. **Nao escolhe a raiz.** Os caminhos vem do build; o painel nao os inventa.
 *  3. **Nao constroi nem instala.** Diz isso antes de qualquer botao.
 *
 * A DISTINCAO QUE O PAINEL NAO PODE PERDER
 *
 * `sem_assinatura` (olhou, nao havia) e `nao_verificadas` (ninguem olhou) aparecem **separados**.
 *
 *	*Um estado que mente e' pior do que nenhum.*
 */

import { useCallback, useState } from "react";

import {
  descreverEstadoDeAssinatura,
  provaDaVerificacao,
  resumirVerificacao,
  verificarArtefactos,
  type RelatorioDeVerificacao,
} from "@/lib/verificationClient";

type Estado = "inativo" | "a-verificar" | "pronto" | "erro";

function mensagemDeFalha(falha: unknown): string {
  if (falha instanceof Error && falha.message.trim() !== "") {
    return falha.message;
  }
  return "nao foi possivel verificar";
}

/**
 * Artefactos de demonstracao: **caminhos e checksums ficticios**.
 *
 * O painel nao tem como descobrir artefactos reais sem um build. Estes servem para exercitar a rota
 * com dados deterministicos, e o painel di-lo -- nao os apresenta como reais.
 */
const ARTEFACTOS_DE_DEMONSTRACAO = [
  { caminho: "/artefactos/hades-windows-amd64.exe", sha256: "a".repeat(64) },
  { caminho: "/artefactos/hades-linux-amd64", sha256: "b".repeat(64) },
];

export function VerificationPanel() {
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState("");
  const [relatorio, setRelatorio] = useState<RelatorioDeVerificacao | null>(null);

  const verificar = useCallback(async () => {
    setEstado("a-verificar");
    setErro("");
    setRelatorio(null);
    try {
      setRelatorio(await verificarArtefactos(ARTEFACTOS_DE_DEMONSTRACAO));
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, []);

  const verificacao = relatorio?.verificacao;
  const resultados = verificacao?.resultados ?? [];

  return (
    <section
      aria-labelledby="verificacao-titulo"
      data-testid="verificacao"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="verificacao-titulo" className="text-lg font-semibold">
        Verificacao de artefactos
      </h2>
      {/* O que a rota faz ANTES do botao. */}
      <p className="mt-1 text-sm text-neutral-400">
        Esta rota <strong>apenas compara</strong> cada artefacto com o SHA-256 medido no build.{" "}
        <strong>Nao constroi, nao assina, nao instala</strong> e nao procura o build por ID.{" "}
        <strong>Aprovada nao significa assinada.</strong>
      </p>
      <p className="mt-1 text-xs text-neutral-500" data-testid="verificacao-demonstracao">
        Os caminhos e checksums abaixo sao de <strong>demonstracao</strong>, nao de um build real.
      </p>

      <div className="mt-3">
        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void verificar()}
          disabled={estado === "a-verificar"}
        >
          Verificar artefactos
        </button>
      </div>

      {estado === "a-verificar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="verificacao-a-correr">
          A medir os ficheiros...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="verificacao-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && relatorio !== null && verificacao !== undefined ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="verificacao-relatorio">
          <p className="text-sm font-medium" data-testid="verificacao-resumo">
            {resumirVerificacao(relatorio)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="verificacao-prova">
            {provaDaVerificacao(relatorio)}
          </p>
          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Total</dt>
            <dd data-testid="verificacao-total">{verificacao.total}</dd>
            <dt className="text-neutral-400">Integros</dt>
            <dd data-testid="verificacao-integros">{verificacao.integros}</dd>
            <dt className="text-neutral-400">Adulterados</dt>
            <dd data-testid="verificacao-adulterados">{verificacao.adulterados}</dd>
            <dt className="text-neutral-400">Sem assinatura</dt>
            <dd data-testid="verificacao-sem-assinatura">{verificacao.sem_assinatura}</dd>
            {/* SEPARADO de sem_assinatura, de proposito. */}
            <dt className="text-neutral-400">Nao inspeccionadas</dt>
            <dd data-testid="verificacao-nao-verificadas">{verificacao.nao_verificadas}</dd>
            <dt className="text-neutral-400">Aprovada</dt>
            <dd data-testid="verificacao-aprovada">{verificacao.aprovada ? "sim" : "nao"}</dd>
          </dl>

          {resultados.length > 0 ? (
            <ul className="max-h-64 space-y-1 overflow-y-auto text-xs" data-testid="verificacao-resultados">
              {resultados.map((resultado) => (
                <li key={resultado.caminho} className="space-y-0.5">
                  <span className="font-mono">{resultado.caminho}</span>
                  <span className="ml-2 text-neutral-500">{resultado.tamanho_bytes} B</span>
                  <span className={resultado.adulterado ? "ml-2 text-red-400" : "ml-2 text-neutral-500"}>
                    {resultado.adulterado ? "ADULTERADO" : resultado.integro ? "integro" : "nao integro"}
                  </span>
                  <span className="ml-2 text-neutral-500">
                    {descreverEstadoDeAssinatura(resultado.assinatura)}
                  </span>
                </li>
              ))}
            </ul>
          ) : null}

          <p className="text-xs text-neutral-400" data-testid="verificacao-aviso-fronteira">
            {relatorio.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
