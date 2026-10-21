import { useState } from "react";
import { getMissionContext, type MissionContextView } from "../lib/agenticClient";

type DeliveryArtifact = { id: string; name: string; sha256: string; size: number };

type Props = { missionId: string; state: string; artifacts?: DeliveryArtifact[] };

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function MissionDeliveryCard({ missionId, state, artifacts = [] }: Props) {
  const completed = ["COMPLETED", "DONE", "SUCCEEDED"].includes(state.toUpperCase());
  // D-8: o CONTEXTO da missão — o que ela SABE, não o que entregou. Carregado só quando o
  // utilizador abre a secção (o contexto é longo e não é o que se olha primeiro), e o que se
  // renderiza vem de `MissionContextView`, cujo TIPO não declara identificadores nem caminhos
  // (lista branca) — a garantia está no tipo, não na disciplina de quem escreve o JSX.
  const [contexto, setContexto] = useState<MissionContextView | null>(null);
  const [erroDoContexto, setErroDoContexto] = useState<string | null>(null);
  const [aCarregarContexto, setACarregarContexto] = useState(false);

  const carregarContexto = async () => {
    if (aCarregarContexto || contexto) return;
    setACarregarContexto(true);
    setErroDoContexto(null);
    try {
      setContexto(await getMissionContext(missionId));
    } catch (erro) {
      // Contexto indisponível NÃO esconde a entrega: a secção diz o que falhou e o resto do
      // cartão continua utilizável — a mesma regra do servidor, onde contexto indisponível não
      // bloqueia a conversa.
      setErroDoContexto(erro instanceof Error ? erro.message : "contexto indisponível");
    } finally {
      setACarregarContexto(false);
    }
  };
  return <section aria-labelledby="mission-delivery-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
    <div className="flex items-start justify-between gap-3"><div><h3 id="mission-delivery-heading" className="text-sm font-semibold">Entrega</h3><p className="mt-1 text-xs text-neutral-500">{completed ? "Resultado rastreável desta missão." : "Os arquivos aparecerão aqui quando a missão produzir um artefato."}</p></div><span className="rounded-full bg-white px-2 py-1 text-[11px] text-neutral-600 dark:bg-neutral-950 dark:text-neutral-300">{artifacts.length} {artifacts.length === 1 ? "arquivo" : "arquivos"}</span></div>
    {artifacts.length === 0 ? <p className="mt-3 text-xs text-neutral-500" role="status">Nenhum artefato disponível ainda. Não há arquivo para baixar.</p> : <ul className="mt-3 space-y-2">{artifacts.map((artifact) => <li key={artifact.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-950"><div className="min-w-0"><p className="truncate text-xs font-medium">{artifact.name}</p><p className="mt-1 text-[11px] text-neutral-500">{formatBytes(artifact.size)} · SHA-256 {artifact.sha256.slice(0, 12)}…</p><details className="mt-1"><summary className="cursor-pointer text-[11px] text-neutral-500">Ver checksum completo</summary><code className="mt-1 block break-all text-[10px] text-neutral-600 dark:text-neutral-400">{artifact.sha256}</code></details></div><a className="min-h-9 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white dark:bg-neutral-100 dark:text-neutral-900" href={`/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(artifact.id)}`} download={artifact.name} aria-label={`Baixar ${artifact.name}`}>Baixar arquivo</a></li>)}</ul>}
    <details className="mt-4" onToggle={(evento) => { if ((evento.currentTarget as HTMLDetailsElement).open) void carregarContexto(); }}>
      <summary className="cursor-pointer text-xs font-medium">Contexto desta missão (o que ela sabe)</summary>
      {erroDoContexto ? <p className="mt-2 text-xs text-neutral-500" role="status">Contexto indisponível: {erroDoContexto}</p> : null}
      {aCarregarContexto ? <p className="mt-2 text-xs text-neutral-500" role="status">A carregar contexto…</p> : null}
      {contexto ? <div className="mt-2 space-y-2 text-xs">
        {contexto.objective ? <p><span className="font-medium">Objetivo:</span> {contexto.objective}</p> : null}
        {contexto.plan && contexto.plan.length > 0 ? <div><p className="font-medium">Plano registado:</p><ul className="mt-1 list-inside list-disc">{contexto.plan.map((titulo, indice) => <li key={indice}>{titulo}</li>)}</ul></div> : null}
        {contexto.checkpoints && contexto.checkpoints.length > 0 ? <div><p className="font-medium">Checkpoints:</p><ul className="mt-1 list-inside list-disc">{contexto.checkpoints.map((marco, indice) => <li key={indice}>{marco}</li>)}</ul></div> : null}
        {contexto.handoff ? <div><p className="font-medium">Handoff para outra IA:</p><pre className="mt-1 whitespace-pre-wrap break-words rounded-lg bg-white p-2 text-[11px] text-neutral-600 dark:bg-neutral-950 dark:text-neutral-300">{contexto.handoff}</pre></div> : null}
      </div> : null}
    </details>
  </section>;
}
