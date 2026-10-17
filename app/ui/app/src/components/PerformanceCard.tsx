import { useEffect, useState } from "react";

import { getPerformance, type AgentPerformance } from "../lib/agenticClient";

type Resposta = { performance: AgentPerformance; scope: string };

function percentagem(taxa: number): string {
  return `${(taxa * 100).toFixed(0)}%`;
}

function centavos(valor: number): string {
  return `R$ ${(valor / 100).toFixed(2)}`;
}

/**
 * PerformanceCard mostra o relatório que o SERVIDOR calcula (E-624) — a superfície não inventa
 * métricas, só as apresenta.
 *
 * DUAS HONESTIDADES SÃO O PONTO DESTE CARTÃO:
 *  1. custo DESCONHECIDO é DECLARADO: o servidor soma só o que sabe e conta o que não sabe. Aqui
 *     mostra-se «N missões com custo desconhecido» em vez de deixar ler o total como se fosse tudo;
 *  2. `not_measured` é RENDERIZADO: o painel diz o que NÃO mede (qualidade, satisfação, orçamento).
 *     *Métricas apresentadas sem essa lista são uma avaliação fictícia por omissão.*
 */
export function PerformanceCard() {
  const [dados, setDados] = useState<Resposta | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    let cancelado = false;
    void getPerformance()
      .then((resposta) => {
        if (!cancelado) setDados(resposta);
      })
      .catch((falha: unknown) => {
        // Falha de leitura NÃO esconde o painel: diz o que aconteceu e nada mais é afirmado.
        if (!cancelado) setErro(falha instanceof Error ? falha.message : "indisponível");
      });
    return () => {
      cancelado = true;
    };
  }, []);

  if (erro) {
    return (
      <section aria-labelledby="performance-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        <h3 id="performance-heading" className="text-sm font-semibold">Desempenho</h3>
        <p className="mt-2 text-xs text-neutral-500" role="status">Não foi possível ler o desempenho: {erro}</p>
      </section>
    );
  }

  if (!dados) {
    return (
      <section aria-labelledby="performance-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        <h3 id="performance-heading" className="text-sm font-semibold">Desempenho</h3>
        <p className="mt-2 text-xs text-neutral-500" role="status">A carregar desempenho…</p>
      </section>
    );
  }

  // E-651 — ***UM PAYLOAD SEM `performance` NÃO PODE DERRUBAR A PÁGINA.***
  //
  // Medido no E2E: o stub responde `{}` a rotas não tratadas, este componente fazia
  // `dados.performance` → `undefined` e logo `relatorio.completed` → `TypeError`, que subia até ao
  // error boundary e mostrava «Esta página encontrou um erro» — *a jornada inteira falhava por
  // causa de um cartão informativo*. O teste unitário não viu porque usava o payload CORRECTO:
  // ***um teste só com o payload bom não prova resiliência a payload mau.***
  const relatorio = dados.performance;
  if (!relatorio) {
    return (
      <section aria-labelledby="performance-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
        <h3 id="performance-heading" className="text-sm font-semibold">Desempenho</h3>
        <p className="mt-2 text-xs text-neutral-500" role="status">
          O servidor não devolveu o relatório de desempenho. Nada é apresentado para não inventar números.
        </p>
      </section>
    );
  }
  const naoMedido = relatorio.not_measured ?? [];

  return (
    <section aria-labelledby="performance-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 id="performance-heading" className="text-sm font-semibold">Desempenho</h3>
          <p className="mt-1 text-xs text-neutral-500">{dados.scope}</p>
        </div>
        <span className="rounded-full bg-white px-2 py-1 text-[11px] text-neutral-600 dark:bg-neutral-950 dark:text-neutral-300">
          {relatorio.completed}/{relatorio.missions} concluídas
        </span>
      </div>

      <dl className="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-3">
        <div><dt className="text-neutral-500">Taxa de conclusão</dt><dd className="font-medium">{percentagem(relatorio.completion_rate)}</dd></div>
        <div><dt className="text-neutral-500">Falhadas</dt><dd className="font-medium">{relatorio.failed}</dd></div>
        <div><dt className="text-neutral-500">Canceladas</dt><dd className="font-medium">{relatorio.cancelled}</dd></div>
        <div><dt className="text-neutral-500">Em curso</dt><dd className="font-medium">{relatorio.running}</dd></div>
        <div><dt className="text-neutral-500">A recuperar</dt><dd className="font-medium">{relatorio.recovering}</dd></div>
        <div><dt className="text-neutral-500">Aprovações pendentes</dt><dd className="font-medium">{relatorio.pending_approvals}</dd></div>
        <div><dt className="text-neutral-500">Retrabalho</dt><dd className="font-medium">{percentagem(relatorio.rework_rate)} ({relatorio.extra_attempts} tentativas extra)</dd></div>
        <div><dt className="text-neutral-500">Chamadas ao modelo</dt><dd className="font-medium">{relatorio.model_calls}</dd></div>
        <div><dt className="text-neutral-500">Tokens</dt><dd className="font-medium">{relatorio.prompt_tokens} → {relatorio.completion_tokens}</dd></div>
      </dl>

      <p className="mt-3 text-xs">
        <span className="text-neutral-500">Custo conhecido: </span>
        <span className="font-medium">{centavos(relatorio.known_cost_cents)}</span>
        {relatorio.missions_with_unknown_cost > 0 ? (
          <span className="text-neutral-500">
            {" "}· {relatorio.missions_with_unknown_cost} {relatorio.missions_with_unknown_cost === 1 ? "missão" : "missões"} com custo desconhecido
          </span>
        ) : null}
      </p>

      {naoMedido.length > 0 ? (
        <div className="mt-3 rounded-lg border border-neutral-200 bg-white p-2 dark:border-neutral-800 dark:bg-neutral-950">
          <p className="text-[11px] font-medium">O que este painel NÃO mede</p>
          <ul className="mt-1 list-inside list-disc text-[11px] text-neutral-500">
            {naoMedido.map((linha) => (
              <li key={linha}>{linha}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
