/**
 * FASE 28.8: o painel de deteccao de projecto.
 *
 * O QUE ESTE PAINEL NAO FAZ
 *
 *  1. **Nao escolhe a raiz.** Vem de `HADES_RAIZ_PROJECTO` no servidor.
 *  2. **Nao instala nem constroi.** Le' nomes de ficheiros e classifica.
 *  3. **Nao promete que a toolchain exista.** Um `go.mod` diz que o projecto e' Go; nao diz que o Go
 *     esta' instalado.
 *
 *	*Um alvo derivado de um ficheiro nao prova que a toolchain exista.*
 */

import { useCallback, useState } from "react";

import {
  descreverGestor,
  descreverStack,
  detectarProjecto,
  provaDaDeteccao,
  resumirDeteccao,
  type RelatorioDeDeteccao,
} from "@/lib/projectDetectClient";

type Estado = "inativo" | "a-detectar" | "pronto" | "erro";

function mensagemDeFalha(falha: unknown): string {
  if (falha instanceof Error && falha.message.trim() !== "") {
    return falha.message;
  }
  return "nao foi possivel detectar o projecto";
}

export function ProjectDetectPanel() {
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState("");
  const [relatorio, setRelatorio] = useState<RelatorioDeDeteccao | null>(null);

  const detectar = useCallback(async () => {
    setEstado("a-detectar");
    setErro("");
    setRelatorio(null);
    try {
      setRelatorio(await detectarProjecto());
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, []);

  const detectado = relatorio?.detectado;
  const stacks = detectado?.stacks ?? [];
  const gestores = detectado?.gestores ?? [];
  const alvos = detectado?.alvos ?? [];
  const manifests = detectado?.manifests ?? [];
  const avisos = detectado?.avisos ?? [];

  return (
    <section
      aria-labelledby="deteccao-titulo"
      data-testid="deteccao"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="deteccao-titulo" className="text-lg font-semibold">
        Deteccao de projecto
      </h2>
      {/* O que a rota faz ANTES do botao. */}
      <p className="mt-1 text-sm text-neutral-400">
        Esta rota <strong>le' os nomes dos ficheiros e o conteudo dos manifests</strong> para
        classificar a stack. <strong>Nao escreve, nao instala e nao constroi.</strong> A raiz vem do
        ambiente do servidor, nunca deste painel.
      </p>
      <p className="mt-1 text-xs text-neutral-500" data-testid="deteccao-alvos-sugestao">
        Os alvos listados sao uma <strong>sugestao</strong> derivada dos ficheiros presentes. Nao
        significam que a toolchain esteja instalada.
      </p>

      <div className="mt-3">
        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void detectar()}
          disabled={estado === "a-detectar"}
        >
          Detectar projecto
        </button>
      </div>

      {estado === "a-detectar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="deteccao-a-correr">
          A ler os manifests...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="deteccao-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && relatorio !== null && detectado !== undefined ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="deteccao-relatorio">
          <p className="text-sm font-medium" data-testid="deteccao-resumo">
            {resumirDeteccao(relatorio)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="deteccao-prova">
            {provaDaDeteccao(relatorio)}
          </p>

          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Raiz</dt>
            <dd className="font-mono" data-testid="deteccao-raiz">
              {detectado.raiz}
            </dd>
            <dt className="text-neutral-400">Stacks</dt>
            <dd data-testid="deteccao-stacks">{stacks.length}</dd>
            <dt className="text-neutral-400">Gestores</dt>
            <dd data-testid="deteccao-gestores">{gestores.length}</dd>
            <dt className="text-neutral-400">Alvos</dt>
            <dd data-testid="deteccao-alvos">{alvos.length}</dd>
          </dl>

          {stacks.length > 0 ? (
            <div className="text-xs" data-testid="deteccao-lista-stacks">
              <p className="text-neutral-400">Stacks:</p>
              <ul className="list-inside list-disc">
                {stacks.map((stack) => (
                  <li key={stack}>{descreverStack(stack)}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {gestores.length > 0 ? (
            <div className="text-xs" data-testid="deteccao-lista-gestores">
              <p className="text-neutral-400">Gestores de pacotes:</p>
              <ul className="list-inside list-disc">
                {gestores.map((gestor) => (
                  <li key={gestor}>{descreverGestor(gestor)}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {alvos.length > 0 ? (
            <div className="text-xs" data-testid="deteccao-lista-alvos">
              <p className="text-neutral-400">Alvos sugeridos:</p>
              <ul className="list-inside list-disc font-mono">
                {alvos.map((alvo) => (
                  <li key={alvo}>{alvo}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {manifests.length > 0 ? (
            <div className="text-xs" data-testid="deteccao-lista-manifests">
              <p className="text-neutral-400">Manifests lidos:</p>
              <ul className="list-inside list-disc font-mono text-neutral-500">
                {manifests.map((manifest) => (
                  <li key={manifest}>{manifest}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {avisos.length > 0 ? (
            <div className="text-xs" data-testid="deteccao-avisos">
              <p className="text-neutral-400">Avisos do detector:</p>
              <ul className="list-inside list-disc text-amber-500">
                {avisos.map((aviso) => (
                  <li key={aviso}>{aviso}</li>
                ))}
              </ul>
            </div>
          ) : null}

          <p className="text-xs text-neutral-400" data-testid="deteccao-aviso-fronteira">
            {relatorio.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
