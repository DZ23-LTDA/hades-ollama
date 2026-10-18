// FASE 28.8 do adendo V3: a area "Capacidades e Ferramentas" do Mission Control.
//
// Esta pagina e' a UI do `CapabilityPlan` que ja' existe no motor (FASE 28.6), na CLI
// (28.7) e na API (28.7). *Uma capacidade que nao chega a UI nao aparece no produto -- fica
// no codigo.*
//
// Regras que a pagina respeita, e que os testes cobram:
//   - tres estados, nao dois: `pronto`, `bloqueado`, `incompleto`. Pintar `incompleto` de
//     vermelho faria o utilizador procurar ferramentas que ja' tem.
//   - o trabalho independente aparece SEMPRE que ha' um bloqueio: *"depende de conta externa"
//     nao e' razao para parar o que nao depende.*
//   - nenhum botao morto: esta pagina NAO instala nada. Instalar exige approval e pertence a
//     outra superficie; um botao "Instalar" aqui seria um botao que mente.
//   - `UNKNOWN` nao e' tratado como `ABSENT`.

import { useCallback, useEffect, useState } from "react";
import { AppSidebar } from "@/components/AppSidebar";
import { InstallProfilesPanel } from "@/components/InstallProfilesPanel";
import { WorktreesPanel } from "@/components/WorktreesPanel";
import { SidebarLayout } from "@/components/layout/layout";
import {
  fetchToolchainDoctor,
  fetchToolchainPlan,
  corDoVeredicto,
  rotuloDoEstado,
  estadoEhAusente,
  type ToolchainDoctor,
  type ToolchainPlano,
  type ToolchainResumoAlvo,
} from "@/lib/toolchainClient";
import {
  ArrowPathIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  WrenchScrewdriverIcon,
  CpuChipIcon,
} from "@heroicons/react/24/outline";

type EstadoDoCarregamento = "a-carregar" | "pronto" | "erro";

function IconeDoVeredicto({ veredicto }: { veredicto: ToolchainResumoAlvo["veredicto"] }) {
  switch (veredicto) {
    case "pronto":
      return <CheckCircleIcon className="h-5 w-5 text-emerald-400" aria-hidden="true" />;
    case "incompleto":
      return <ExclamationTriangleIcon className="h-5 w-5 text-amber-400" aria-hidden="true" />;
    default:
      return <XCircleIcon className="h-5 w-5 text-red-400" aria-hidden="true" />;
  }
}

function DetalheDoAlvo({ alvo }: { alvo: string }) {
  const [plano, setPlano] = useState<ToolchainPlano | null>(null);
  const [estado, setEstado] = useState<EstadoDoCarregamento>("a-carregar");
  const [erro, setErro] = useState<string>("");

  useEffect(() => {
    let vivo = true;
    setEstado("a-carregar");
    fetchToolchainPlan(alvo)
      .then((dados) => {
        if (!vivo) return;
        setPlano(dados);
        setEstado("pronto");
      })
      .catch((e: unknown) => {
        if (!vivo) return;
        setErro(e instanceof Error ? e.message : String(e));
        setEstado("erro");
      });
    return () => {
      vivo = false;
    };
  }, [alvo]);

  if (estado === "a-carregar") {
    return (
      <p className="px-4 py-3 text-sm text-neutral-500 dark:text-neutral-400">
        A resolver o alvo {alvo}…
      </p>
    );
  }
  if (estado === "erro") {
    return (
      <p className="px-4 py-3 text-sm text-red-400" role="alert">
        Nao foi possivel resolver o alvo {alvo}: {erro}
      </p>
    );
  }
  if (!plano) return null;

  return (
    <div className="space-y-4 px-4 py-3">
      <table className="w-full text-left text-sm">
        <caption className="sr-only">
          Ferramentas exigidas pelo alvo {alvo} e o estado de cada uma neste host
        </caption>
        <thead>
          <tr className="text-xs uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
            <th scope="col" className="py-1 pr-3 font-medium">
              Ferramenta
            </th>
            <th scope="col" className="py-1 pr-3 font-medium">
              Estado
            </th>
            <th scope="col" className="py-1 font-medium">
              Versao
            </th>
          </tr>
        </thead>
        <tbody>
          {plano.probes.map((probe) => (
            <tr key={probe.nome} className="border-t border-neutral-200 dark:border-neutral-800">
              <td className="py-1.5 pr-3">
                <span className="font-medium text-neutral-900 dark:text-white">{probe.nome}</span>
                {probe.obrigatorio ? (
                  <span className="ml-1 text-xs text-neutral-500">(obrigatoria)</span>
                ) : null}
                <span className="block text-xs text-neutral-500">{probe.motivo}</span>
              </td>
              <td className="py-1.5 pr-3">
                <span className={corDoVeredicto(
                  probe.estado === "PRESENT_COMPATIBLE_HEALTHY"
                    ? "pronto"
                    : estadoEhAusente(probe.estado)
                      ? "bloqueado"
                      : "incompleto",
                )}>
                  {rotuloDoEstado(probe.estado)}
                </span>
                {probe.detalhe ? (
                  <span className="block text-xs text-neutral-500">{probe.detalhe}</span>
                ) : null}
              </td>
              <td className="py-1.5 text-xs text-neutral-500">{probe.versao || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {plano.blockers.length > 0 ? (
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-red-400">
            Bloqueios
          </h4>
          <ul className="mt-1 list-inside list-disc text-sm text-neutral-700 dark:text-neutral-300">
            {plano.blockers.map((b) => (
              <li key={b}>{b}</li>
            ))}
          </ul>
        </div>
      ) : null}

      {plano.trabalho_independente.length > 0 ? (
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-emerald-400">
            Pode avancar sem esperar
          </h4>
          <ul className="mt-1 list-inside list-disc text-sm text-neutral-700 dark:text-neutral-300">
            {plano.trabalho_independente.map((t) => (
              <li key={t}>{t}</li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}

export function CapabilitiesPage() {
  const [doctor, setDoctor] = useState<ToolchainDoctor | null>(null);
  const [estado, setEstado] = useState<EstadoDoCarregamento>("a-carregar");
  const [erro, setErro] = useState<string>("");
  const [aberto, setAberto] = useState<string>("");

  const carregar = useCallback(async () => {
    setEstado("a-carregar");
    setErro("");
    try {
      const dados = await fetchToolchainDoctor();
      setDoctor(dados);
      setEstado("pronto");
    } catch (e: unknown) {
      setErro(e instanceof Error ? e.message : String(e));
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  return (
    <SidebarLayout title="Capacidades e Ferramentas" sidebar={<AppSidebar current="capabilities" />}>
      <div className="mx-auto max-w-6xl space-y-6 px-4 py-8 lg:px-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-neutral-900 dark:text-white">
              <WrenchScrewdriverIcon className="h-6 w-6" aria-hidden="true" />
              Capacidades e Ferramentas
            </h1>
            <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
              O que este computador ja' tem para construir cada alvo. Nada aqui instala ou altera
              nada — o preflight apenas mede.
            </p>
          </div>
          <button
            type="button"
            onClick={() => void carregar()}
            className="inline-flex items-center gap-2 rounded-lg border border-neutral-300 px-3 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            <ArrowPathIcon className="h-4 w-4" aria-hidden="true" />
            Medir de novo
          </button>
        </div>

        {estado === "a-carregar" ? (
          <p className="text-sm text-neutral-500 dark:text-neutral-400" role="status">
            A medir as ferramentas deste computador…
          </p>
        ) : null}

        {estado === "erro" ? (
          <div
            role="alert"
            className="rounded-lg border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-300"
          >
            <p className="font-medium">Nao foi possivel medir as ferramentas.</p>
            <p className="mt-1 text-red-200/80">{erro}</p>
          </div>
        ) : null}

        {estado === "pronto" && doctor ? (
          <>
            <div className="rounded-lg border border-neutral-200 bg-neutral-50 p-4 text-sm dark:border-neutral-800 dark:bg-neutral-900">
              <p className="flex items-center gap-2 font-medium text-neutral-900 dark:text-white">
                <CpuChipIcon className="h-4 w-4" aria-hidden="true" />
                {doctor.host} · {doctor.goos}
              </p>
              <p className="mt-1 text-neutral-600 dark:text-neutral-400">{doctor.resumo}</p>
            </div>

            {doctor.alvos.length === 0 ? (
              <p className="text-sm text-neutral-500">Nenhum alvo de build declarado.</p>
            ) : (
              <ul className="space-y-2" aria-label="Alvos de build">
                {doctor.alvos.map((alvo) => {
                  const expandido = aberto === alvo.alvo;
                  return (
                    <li
                      key={alvo.alvo}
                      className="rounded-lg border border-neutral-200 dark:border-neutral-800"
                    >
                      <button
                        type="button"
                        onClick={() => setAberto(expandido ? "" : alvo.alvo)}
                        aria-expanded={expandido}
                        className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-neutral-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 dark:hover:bg-neutral-900"
                      >
                        {expandido ? (
                          <ChevronDownIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
                        ) : (
                          <ChevronRightIcon className="h-4 w-4 shrink-0" aria-hidden="true" />
                        )}
                        <IconeDoVeredicto veredicto={alvo.veredicto} />
                        <span className="flex-1 font-medium text-neutral-900 dark:text-white">
                          {alvo.alvo}
                        </span>
                        <span className={`text-sm ${corDoVeredicto(alvo.veredicto)}`}>
                          {alvo.veredicto}
                        </span>
                        {alvo.exigencias_bloqueantes > 0 ? (
                          <span className="text-xs text-neutral-500">
                            {alvo.exigencias_bloqueantes} bloqueante
                            {alvo.exigencias_bloqueantes === 1 ? "" : "s"}
                          </span>
                        ) : null}
                      </button>
                      {expandido ? <DetalheDoAlvo alvo={alvo.alvo} /> : null}
                    </li>
                  );
                })}
              </ul>
            )}
          </>
        ) : null}
      </div>

      {/* FASE 28.6-A: os perfis de instalacao na MESMA pagina, nao num painel paralelo.
          Responde a mesma pergunta do preflight -- o que falta nesta maquina -- e por isso
          vive aqui. *Dois paineis para a mesma pergunta divergem no primeiro dia.* */}
      <InstallProfilesPanel />

      {/* FASE 2: os worktrees vivem na MESMA pagina de capacidades. O worktree e' o ambiente onde
          o trabalho acontece; responde ao contexto da tarefa do mesmo jeito que o preflight.
          *Um mecanismo que ninguem consegue ver e' um mecanismo que ninguem consegue usar.* */}
      <WorktreesPanel />
    </SidebarLayout>
  );
}
