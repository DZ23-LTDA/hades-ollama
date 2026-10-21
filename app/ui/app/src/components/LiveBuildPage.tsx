import { ArrowTopRightOnSquareIcon, BoltIcon, CodeBracketIcon, EyeIcon } from "@heroicons/react/24/outline";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useT } from "@/i18n";
import {
  agentFetchBlob,
  editarArtefacto,
  getMission,
  listMissionEvents,
  listMissions,
  type AgentEvent,
  type AgentMission,
} from "@/lib/agenticClient";

/** Um passo legivel derivado de um evento cru da missao. */
export type PassoConstrucao = {
  id: string;
  tipo: string;
  titulo: string;
  detalhe?: string;
  quando: string;
};

function extrairTexto(payload: unknown, chaves: string[]): string | undefined {
  if (!payload || typeof payload !== "object") return undefined;
  const obj = payload as Record<string, unknown>;
  for (const chave of chaves) {
    const valor = obj[chave];
    if (typeof valor === "string" && valor.trim() !== "") return valor;
  }
  return undefined;
}

/**
 * Traduz o evento do servidor num passo que o utilizador entende.
 *
 * O objetivo e' o painel contar a MESMA historia que o agente esta' a executar: plano, ferramenta,
 * artefacto, aprovacao, erro. Eventos desconhecidos mostram o tipo em vez de inventar um rotulo ---
 * um rotulo inventado esconderia um evento que ninguem tratou.
 */
export function descreverEvento(evento: AgentEvent): PassoConstrucao {
  const tipo = (evento.type || "").toLowerCase();
  const detalhe =
    extrairTexto(evento.payload, ["name", "tool", "title", "path", "summary", "message"]) ??
    extrairTexto(evento.payload, ["detail"]);
  const base = { id: evento.id, tipo, detalhe, quando: evento.created_at };

  if (tipo.includes("plan")) return { ...base, titulo: "Plano definido" };
  if (tipo.includes("approval") || tipo.includes("aprovacao")) return { ...base, titulo: "Aprovação pedida" };
  if (tipo.includes("artifact") || tipo.includes("artefact")) return { ...base, titulo: "Artefacto gerado" };
  if (tipo.includes("tool") || tipo.includes("command")) return { ...base, titulo: "Ferramenta usada" };
  if (tipo.includes("error") || tipo.includes("fail")) return { ...base, titulo: "Erro" };
  if (tipo.includes("step") || tipo.includes("step_")) return { ...base, titulo: "Passo" };
  return { ...base, titulo: evento.type || "Evento" };
}

function horaCurta(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

const ESTADOS_TERMINAIS = ["done", "completed", "failed", "cancelled", "error"];

export function LiveBuildPanel() {
  const { t } = useT();
  const [missoes, setMissoes] = useState<AgentMission[]>([]);
  const [missaoId, setMissaoId] = useState<string>("");
  const [missao, setMissao] = useState<AgentMission | null>(null);
  const [eventos, setEventos] = useState<AgentEvent[]>([]);
  const [erro, setErro] = useState<string | null>(null);
  const [aba, setAba] = useState<"preview" | "codigo">("preview");
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewTipo, setPreviewTipo] = useState<"html" | "imagem" | "outro" | null>(null);
  const [codigo, setCodigo] = useState<string>("");
  // I1b parte 4: edicao inline. O codigo editado vai para o PUT (versao nova) e a pre-visualizacao
  // recarrega -- o objetivo e' "editar e ver a mudanca", nao guardar num sitio que ninguem le.
  const [salvandoEdicao, setSalvandoEdicao] = useState(false);
  const [edicaoSalva, setEdicaoSalva] = useState<string | null>(null);
  const [recarregar, setRecarregar] = useState(0);
  const feedRef = useRef<HTMLDivElement | null>(null);

  const passos = useMemo(() => eventos.map(descreverEvento), [eventos]);
  const emExecucao = useMemo(() => {
    const estado = (missao?.state ?? "").toLowerCase();
    return estado !== "" && !ESTADOS_TERMINAIS.includes(estado);
  }, [missao]);

  const carregarMissoes = useCallback(async () => {
    try {
      const { missions } = await listMissions();
      setMissoes(missions);
      setMissaoId((atual) => atual || missions[0]?.id || "");
    } catch (e) {
      setErro(String(e));
    }
  }, []);

  useEffect(() => {
    void carregarMissoes();
  }, [carregarMissoes]);

  // Sondagem: eventos + estado da missao. Sem isto o painel seria uma fotografia, nao um andamento.
  useEffect(() => {
    if (!missaoId) return;
    let cancelado = false;
    const ciclo = async () => {
      try {
        const [lista, atual] = await Promise.all([listMissionEvents(missaoId), getMission(missaoId)]);
        if (cancelado) return;
        setEventos(lista.events ?? []);
        setMissao(atual);
        setErro(null);
      } catch (e) {
        if (!cancelado) setErro(String(e));
      }
    };
    void ciclo();
    const timer = setInterval(() => void ciclo(), 2000);
    return () => {
      cancelado = true;
      clearInterval(timer);
    };
  }, [missaoId]);

  // A previa acompanha a construcao: recarrega quando aparece artefacto novo ou a missao avanca.
  const artefactoAtual = missao?.artifacts?.length ? missao.artifacts[missao.artifacts.length - 1] : null;
  const chaveArtefacto = artefactoAtual ? `${artefactoAtual.id}:${missao?.artifacts?.length ?? 0}` : "";

  useEffect(() => {
    if (!missaoId || !artefactoAtual) {
      setPreviewUrl(null);
      setPreviewTipo(null);
      setCodigo("");
      return;
    }
    let cancelado = false;
    let url: string | null = null;
    const buscar = async () => {
      try {
        const blob = await agentFetchBlob(
          `/api/agent/v1/missions/${encodeURIComponent(missaoId)}/artifacts/${encodeURIComponent(artefactoAtual.id)}`,
        );
        if (cancelado) return;
        const tipo: "html" | "imagem" | "outro" = blob.type.includes("html")
          ? "html"
          : blob.type.startsWith("image/")
            ? "imagem"
            : "outro";
        if (url) URL.revokeObjectURL(url);
        url = URL.createObjectURL(blob);
        setPreviewUrl(url);
        setPreviewTipo(tipo);
        if (tipo === "html" || blob.type.startsWith("text/")) setCodigo(await blob.text());
      } catch (e) {
        if (!cancelado) {
          setPreviewUrl(null);
          setPreviewTipo(null);
          setErro(String(e));
        }
      }
    };
    void buscar();
    const timer = emExecucao ? setInterval(() => void buscar(), 4000) : null;
    return () => {
      cancelado = true;
      if (timer) clearInterval(timer);
      if (url) URL.revokeObjectURL(url);
    };
  }, [missaoId, chaveArtefacto, emExecucao, recarregar]);

  const handleSalvarEdicao = async () => {
    if (!missaoId || !artefactoAtual) return;
    if (artefactoAtual.version === undefined) {
      // Sem a versao corrente nao ha CAS: gravar seria sobrescrever as cegas.
      setErro(t("liveBuild.error", { error: "versao do artefacto desconhecida" }));
      return;
    }
    setSalvandoEdicao(true);
    setEdicaoSalva(null);
    try {
      const r = await editarArtefacto(missaoId, artefactoAtual.id, codigo, artefactoAtual.version);
      setEdicaoSalva(t("liveBuild.saved", { version: r.version }));
      setRecarregar((n) => n + 1);
    } catch (e) {
      setErro(String(e));
    } finally {
      setSalvandoEdicao(false);
    }
  };

  useEffect(() => {
    const el = feedRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [passos.length]);

  return (
    <div className="h-full min-h-0">
      <div className="flex h-full min-h-0 flex-col">
        <header className="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200/70 px-4 py-3 dark:border-neutral-800">
          <div className="min-w-0">
            <h1 className="flex items-center gap-2 text-sm font-semibold text-neutral-900 dark:text-white">
              <BoltIcon className="h-4 w-4" aria-hidden="true" />
              {t("liveBuild.title")}
            </h1>
            <p className="mt-0.5 text-xs text-neutral-500 dark:text-neutral-400">{t("liveBuild.subtitle")}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <label className="text-xs text-neutral-500 dark:text-neutral-400" htmlFor="missao-ao-vivo">
              {t("liveBuild.selectMission")}
            </label>
            <select
              id="missao-ao-vivo"
              value={missaoId}
              onChange={(e) => setMissaoId(e.target.value)}
              className="max-w-[22rem] truncate rounded-lg border border-neutral-300 bg-white px-2 py-1 text-xs dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"
            >
              {missoes.length === 0 ? <option value="">{t("liveBuild.noMissions")}</option> : null}
              {missoes.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.objective}
                </option>
              ))}
            </select>
            {missao ? (
              <span
                className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${
                  emExecucao
                    ? "bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"
                    : "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
                }`}
              >
                {missao.state}
              </span>
            ) : null}
          </div>
        </header>

        {erro ? (
          <p role="alert" className="border-b border-red-200 bg-red-50 px-4 py-2 text-xs text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300">
            {t("liveBuild.error", { error: erro })}
          </p>
        ) : null}

        <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[minmax(0,28rem)_minmax(0,1fr)]">
          <section aria-label={t("liveBuild.steps")} className="flex min-h-0 flex-col border-b border-neutral-200/70 lg:border-b-0 lg:border-r dark:border-neutral-800">
            <h2 className="px-4 pt-3 text-[11px] font-semibold uppercase tracking-wide text-neutral-500 dark:text-neutral-400">
              {t("liveBuild.steps")}
            </h2>
            <div ref={feedRef} className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
              {passos.length === 0 ? (
                <p className="text-xs text-neutral-500 dark:text-neutral-400">{t("liveBuild.waiting")}</p>
              ) : (
                <ol className="space-y-2">
                  {passos.map((passo) => (
                    <li key={passo.id} className="rounded-lg border border-neutral-200/70 bg-white p-2 dark:border-neutral-800 dark:bg-neutral-900">
                      <div className="flex items-baseline justify-between gap-2">
                        <span className="text-xs font-medium text-neutral-900 dark:text-white">{passo.titulo}</span>
                        <time className="shrink-0 text-[10px] text-neutral-400" dateTime={passo.quando}>
                          {horaCurta(passo.quando)}
                        </time>
                      </div>
                      {passo.detalhe ? (
                        <p className="mt-0.5 break-words text-[11px] text-neutral-600 dark:text-neutral-300">{passo.detalhe}</p>
                      ) : null}
                      <code className="mt-0.5 block text-[10px] text-neutral-400">{passo.tipo}</code>
                    </li>
                  ))}
                </ol>
              )}
            </div>
          </section>

          <section aria-label={t("liveBuild.preview")} className="flex min-h-0 flex-col">
            <div className="flex items-center justify-between gap-2 border-b border-neutral-200/70 px-3 py-2 dark:border-neutral-800">
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => setAba("preview")}
                  aria-pressed={aba === "preview"}
                  className={`flex items-center gap-1 rounded-lg px-2 py-1 text-xs ${
                    aba === "preview" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-600 dark:text-neutral-300"
                  }`}
                >
                  <EyeIcon className="h-3.5 w-3.5" aria-hidden="true" />
                  {t("liveBuild.view")}
                </button>
                <button
                  type="button"
                  onClick={() => setAba("codigo")}
                  aria-pressed={aba === "codigo"}
                  className={`flex items-center gap-1 rounded-lg px-2 py-1 text-xs ${
                    aba === "codigo" ? "bg-neutral-900 text-white dark:bg-white dark:text-neutral-900" : "text-neutral-600 dark:text-neutral-300"
                  }`}
                >
                  <CodeBracketIcon className="h-3.5 w-3.5" aria-hidden="true" />
                  {t("liveBuild.code")}
                </button>
              </div>
              {previewUrl ? (
                <a
                  href={previewUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="flex items-center gap-1 text-xs text-neutral-600 underline decoration-dotted dark:text-neutral-300"
                >
                  <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" aria-hidden="true" />
                  {t("liveBuild.openNewTab")}
                </a>
              ) : null}
            </div>
            <div className="min-h-0 flex-1 bg-neutral-100 dark:bg-neutral-950">
              {!previewUrl ? (
                <p className="p-4 text-xs text-neutral-500 dark:text-neutral-400">{t("liveBuild.previewEmpty")}</p>
              ) : aba === "codigo" ? (
                <div className="flex h-full flex-col">
                  <div className="flex items-center justify-between gap-2 border-b border-neutral-200/70 px-3 py-2 dark:border-neutral-800">
                    <span className="text-[10px] text-neutral-500 dark:text-neutral-400">{t("liveBuild.editHint")}</span>
                    <button
                      type="button"
                      onClick={() => void handleSalvarEdicao()}
                      disabled={salvandoEdicao || codigo.trim() === ""}
                      className="rounded-lg bg-emerald-600 px-3 py-1 text-xs font-medium text-white disabled:opacity-40 hover:bg-emerald-700"
                    >
                      {salvandoEdicao ? t("liveBuild.saving") : t("liveBuild.save")}
                    </button>
                  </div>
                  {edicaoSalva ? (
                    <p className="px-3 pt-2 text-[11px] text-emerald-600" role="status">{edicaoSalva}</p>
                  ) : null}
                  <textarea
                    value={codigo}
                    onChange={(e) => setCodigo(e.target.value)}
                    spellCheck={false}
                    aria-label={t("liveBuild.code")}
                    className="min-h-0 flex-1 resize-none bg-white p-3 font-mono text-[11px] text-neutral-800 outline-none dark:bg-neutral-950 dark:text-neutral-200"
                  />
                </div>
              ) : previewTipo === "imagem" ? (
                <img src={previewUrl} alt={t("liveBuild.preview")} className="max-h-full max-w-full object-contain p-3" />
              ) : (
                // sandbox sem same-origin: o conteudo gerado nao herda a sessao da app.
                <iframe
                  title={t("liveBuild.preview")}
                  src={missaoId && artefactoAtual ? urlExecucao(missaoId, artefactoAtual.id) : previewUrl}
                  sandbox="allow-scripts"
                  className="h-full w-full border-0 bg-white"
                />
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

/**
 * URL que EXECUTA o artefacto no servidor (modo inline), em vez de o descarregar.
 *
 * I1a: a rota do artefacto usa c.FileAttachment (Content-Disposition: attachment), por isso um
 * iframe apontado a ela DESCARREGAVA o ficheiro -- era a razao de o painel usar um blob
 * temporario. Com ?inline=1 o servidor serve o mesmo conteudo com Content-Disposition: inline e
 * o content-type certo, e o iframe passa a correr a app gerada de verdade, same-origin.
 */
export function urlExecucao(missionId: string, artifactId: string): string {
  return (
    "/api/agent/v1/missions/" +
    encodeURIComponent(missionId) +
    "/artifacts/" +
    encodeURIComponent(artifactId) +
    "?inline=1"
  );
}
