// Relatório de publicação do Builder (E-282).
//
// Por que este componente existe separado: a página do Studio tem 1483 linhas e o aviso precisa ser TESTÁVEL
// sozinho. A propriedade que ele protege é simples e inegociável: NUNCA dizer "publicado" com cara de sucesso
// quando o relatório diz que a aplicação não é executável.
import type { BuilderPublishReport } from "@/lib/agenticClient";

export function PublishReportNotice({ report }: { report?: BuilderPublishReport | null }) {
  // Sem relatório não há o que dizer: inventar um estado aqui seria repetir o problema que o relatório corrige.
  if (!report) {
    return null;
  }
  const faltando = report.missing ?? [];
  if (!report.runnable) {
    return (
      <div
        role="status"
        aria-live="polite"
        className="mt-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
      >
        <p className="font-semibold">
          Publicação local: arquivos copiados, aplicação NÃO executável.
        </p>
        <p className="mt-1">
          {report.files_copied} arquivo(s) no diretório publicado.
          {faltando.length > 0 ? ` Falta: ${faltando.join(", ")}.` : ""}
        </p>
        {report.note && <p className="mt-1">{report.note}</p>}
      </div>
    );
  }
  return (
    <div
      role="status"
      aria-live="polite"
      className="mt-2 rounded-lg border border-neutral-200 px-3 py-2 text-xs text-neutral-600 dark:border-neutral-800 dark:text-neutral-300"
    >
      <p>
        Publicação local com backend detectado ({report.files_copied} arquivo(s)). {report.note}
      </p>
    </div>
  );
}
