import { useEffect, useState } from "react";

import { getCapabilities, type CapabilityReport, type SingleCapabilityReport } from "../lib/agenticClient";

type Props = { model?: string };

function eRelatorioCompleto(dados: CapabilityReport | SingleCapabilityReport): dados is CapabilityReport {
  return Array.isArray((dados as CapabilityReport).models);
}

/**
 * CapabilitiesCard mostra a VERDADE de capacidade (§9): o que o produto sabe de cada modelo, com a
 * ORIGEM do dado — `medido` ou `declarado`.
 *
 * A HONESTIDADE CENTRAL: ***«não medido» não é «não suporta»***. Quando o modelo não está no
 * registo, o cartão diz que não foi medido e que o runtime cai em modo compatível; **não** afirma
 * suporte nem o nega. E cada capacidade mostra a origem, porque *capacidade sem origem é
 * exactamente o que o V2.2 proíbe*.
 */
export function CapabilitiesCard({ model }: Props) {
  const [dados, setDados] = useState<CapabilityReport | SingleCapabilityReport | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    let cancelado = false;
    void getCapabilities(model)
      .then((resposta) => {
        if (!cancelado) setDados(resposta);
      })
      .catch((falha: unknown) => {
        if (!cancelado) setErro(falha instanceof Error ? falha.message : "indisponível");
      });
    return () => {
      cancelado = true;
    };
  }, [model]);

  const cabecalho = (
    <div>
      <h3 id="capabilities-heading" className="text-sm font-semibold">Capacidade dos modelos</h3>
      <p className="mt-1 text-xs text-neutral-500">O que o produto sabe — e de onde o sabe.</p>
    </div>
  );

  if (erro) {
    return (
      <section aria-labelledby="capabilities-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        {cabecalho}
        <p className="mt-2 text-xs text-neutral-500" role="status">Não foi possível ler as capacidades: {erro}</p>
      </section>
    );
  }

  if (!dados) {
    return (
      <section aria-labelledby="capabilities-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        {cabecalho}
        <p className="mt-2 text-xs text-neutral-500" role="status">A carregar capacidades…</p>
      </section>
    );
  }

  // E-655: um payload sem `models`/`known` NÃO pode derrubar a página (a lição do E-651: o
  // `PerformanceCard` caiu a página inteira no E2E por confiar na forma do payload).
  if (!eRelatorioCompleto(dados) && typeof dados.known !== "boolean") {
    return (
      <section aria-labelledby="capabilities-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        {cabecalho}
        <p className="mt-2 text-xs text-neutral-500" role="status">
          O servidor não devolveu o relatório de capacidade. Nada é apresentado para não inventar.
        </p>
      </section>
    );
  }

  if (!eRelatorioCompleto(dados)) {
    return (
      <section aria-labelledby="capabilities-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        {cabecalho}
        <dl className="mt-3 grid grid-cols-2 gap-3 text-xs">
          <div><dt className="text-neutral-500">Modelo</dt><dd className="font-medium">{dados.model}</dd></div>
          <div>
            <dt className="text-neutral-500">Estado</dt>
            <dd className="font-medium">
              {dados.known ? "medido/declarado" : "NÃO MEDIDO (modo compatível, por texto)"}
            </dd>
          </div>
        </dl>
        <p className="mt-2 text-xs text-neutral-500">{dados.reason}</p>
        {dados.capabilities ? (
          <p className="mt-2 text-xs">
            <span className="text-neutral-500">Origem do dado: </span>
            <span className="font-medium">{dados.capabilities.source}</span>
          </p>
        ) : null}
      </section>
    );
  }

  return (
    <section aria-labelledby="capabilities-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
      {cabecalho}
      <ul className="mt-3 space-y-2">
        {dados.models.map((modelo) => (
          <li key={modelo.model} className="rounded-lg border border-neutral-200 bg-white p-3 text-xs dark:border-neutral-800 dark:bg-neutral-950">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="font-medium">{modelo.model}</span>
              {/* A ORIGEM SEMPRE VISÍVEL: capacidade sem origem não é capacidade. */}
              <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[11px] text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
                {modelo.source}
              </span>
            </div>
            <p className="mt-1 text-neutral-500">
              {modelo.tool_calling ? "tool calls nativos" : "sem tool call nativo (modo compatível)"}
              {modelo.vision ? " · visão" : ""}
              {modelo.structured_output ? " · saída estruturada" : ""}
            </p>
            {modelo.note ? <p className="mt-1 text-neutral-500">{modelo.note}</p> : null}
          </li>
        ))}
      </ul>
      <p className="mt-3 text-[11px] text-neutral-500">{dados.policy}</p>
      <p className="mt-1 text-[11px] text-neutral-400">{dados.scope}</p>
    </section>
  );
}
