type DeliveryArtifact = { id: string; name: string; sha256: string; size: number };

type Props = { missionId: string; state: string; artifacts?: DeliveryArtifact[] };

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function MissionDeliveryCard({ missionId, state, artifacts = [] }: Props) {
  const completed = ["COMPLETED", "DONE", "SUCCEEDED"].includes(state.toUpperCase());
  return <section aria-labelledby="mission-delivery-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
    <div className="flex items-start justify-between gap-3"><div><h3 id="mission-delivery-heading" className="text-sm font-semibold">Entrega</h3><p className="mt-1 text-xs text-neutral-500">{completed ? "Resultado rastreável desta missão." : "Os arquivos aparecerão aqui quando a missão produzir um artefato."}</p></div><span className="rounded-full bg-white px-2 py-1 text-[11px] text-neutral-600 dark:bg-neutral-950 dark:text-neutral-300">{artifacts.length} {artifacts.length === 1 ? "arquivo" : "arquivos"}</span></div>
    {artifacts.length === 0 ? <p className="mt-3 text-xs text-neutral-500" role="status">Nenhum artefato disponível ainda. Não há arquivo para baixar.</p> : <ul className="mt-3 space-y-2">{artifacts.map((artifact) => <li key={artifact.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-950"><div className="min-w-0"><p className="truncate text-xs font-medium">{artifact.name}</p><p className="mt-1 text-[11px] text-neutral-500">{formatBytes(artifact.size)} · SHA-256 {artifact.sha256.slice(0, 12)}…</p><details className="mt-1"><summary className="cursor-pointer text-[11px] text-neutral-500">Ver checksum completo</summary><code className="mt-1 block break-all text-[10px] text-neutral-600 dark:text-neutral-400">{artifact.sha256}</code></details></div><a className="min-h-9 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white dark:bg-neutral-100 dark:text-neutral-900" href={`/api/agent/v1/missions/${encodeURIComponent(missionId)}/artifacts/${encodeURIComponent(artifact.id)}`} download={artifact.name} aria-label={`Baixar ${artifact.name}`}>Baixar arquivo</a></li>)}</ul>}
  </section>;
}
