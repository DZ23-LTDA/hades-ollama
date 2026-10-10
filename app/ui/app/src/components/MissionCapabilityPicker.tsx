import { useId } from "react";
import { elevatedWarning, type ScopeOption } from "@/lib/missionCapabilities";

const RISK_LABEL: Record<string, string> = {
  unknown: "risco não classificado",
  low: "risco baixo",
  medium: "risco médio",
  high: "risco alto",
  critical: "risco crítico",
};

type Props = {
  /** Escopos que o runtime declara; derivados de GET /api/agent/v1/tools. */
  options: ScopeOption[];
  /** Escopos elevados escolhidos pelo usuário (o baseline não entra aqui). */
  selected: string[];
  onToggle: (name: string) => void;
  /** true quando o catálogo não pôde ser lido do runtime. */
  catalogUnavailable?: boolean;
  disabled?: boolean;
};

/**
 * Seletor de permissões da missão. Não inventa vocabulário: lista exatamente
 * os escopos publicados pelo catálogo de ferramentas do runtime, mostrando
 * risco, exigência de aprovação e o efeito de cada um. O baseline de leitura é
 * sempre concedido e aparece marcado como fixo.
 */
export function MissionCapabilityPicker({
  options,
  selected,
  onToggle,
  catalogUnavailable = false,
  disabled = false,
}: Props) {
  const groupID = useId();
  const chosen = new Set(
    options
      .filter((option) => selected.includes(option.name))
      .map((option) => option.name),
  );
  const warning = elevatedWarning(
    options.filter((option) => chosen.has(option.name)),
  );
  const granted =
    options.filter((option) => option.elevated && chosen.has(option.name))
      .length + 1;

  return (
    <details className="mt-1 rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-[11px] text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900/40 dark:text-neutral-300">
      <summary className="cursor-pointer select-none font-medium text-neutral-700 dark:text-neutral-200">
        Permissões da missão · {granted} escopo(s)
      </summary>
      <p className="mt-2">
        A missão sempre recebe <code className="font-mono">workspace:read</code>
        . Marque escopos adicionais apenas quando a tarefa exigir; o servidor
        nega qualquer escopo que não esteja no catálogo de ferramentas.
      </p>
      {catalogUnavailable ? (
        <p
          role="status"
          className="mt-2 rounded-lg bg-amber-50 px-2 py-1 text-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
        >
          Catálogo de ferramentas indisponível: somente{" "}
          <code className="font-mono">workspace:read</code> será concedido. Rode
          a missão pela API para conceder escopos elevados.
        </p>
      ) : options.length === 0 ? (
        <p
          role="status"
          className="mt-2 rounded-lg bg-amber-50 px-2 py-1 text-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
        >
          O runtime não publicou nenhuma ferramenta ainda: somente{" "}
          <code className="font-mono">workspace:read</code> será concedido.
        </p>
      ) : (
        <fieldset className="mt-2 min-w-0">
          <legend className="font-medium text-neutral-700 dark:text-neutral-200">
            Escopos concedíveis
          </legend>
          <ul className="mt-2 flex flex-col gap-2">
            {options.map((option) => {
              const descriptionID = `${groupID}-${option.name.replace(/[^a-zA-Z0-9]+/g, "-")}-help`;
              if (!option.elevated) {
                return (
                  <li key={option.name} className="flex items-start gap-2">
                    <input
                      type="checkbox"
                      checked
                      disabled
                      aria-label={option.name}
                    />
                    <span>
                      <span className="font-mono">{option.name}</span> · sempre
                      concedido ({RISK_LABEL[option.risk] ?? option.risk})
                      <span id={descriptionID} className="block">
                        {option.tools.length > 0
                          ? `Ferramentas: ${option.tools.join(", ")}.`
                          : "Nenhuma ferramenta declarou este escopo."}
                      </span>
                    </span>
                  </li>
                );
              }
              return (
                <li key={option.name} className="flex items-start gap-2">
                  <input
                    type="checkbox"
                    checked={chosen.has(option.name)}
                    disabled={disabled}
                    onChange={() => onToggle(option.name)}
                    aria-label={option.name}
                    aria-describedby={descriptionID}
                  />
                  <span>
                    <span className="font-mono">{option.name}</span> ·{" "}
                    {RISK_LABEL[option.risk] ?? option.risk}
                    {option.requiresApproval ? " · exige aprovação" : ""}
                    <span id={descriptionID} className="block">
                      {option.tools.length > 0
                        ? `Ferramentas: ${option.tools.join(", ")}.`
                        : "Nenhuma ferramenta declarou este escopo."}
                    </span>
                  </span>
                </li>
              );
            })}
          </ul>
        </fieldset>
      )}
      {warning ? (
        <p
          role="alert"
          className="mt-2 rounded-lg bg-red-50 px-2 py-1 text-red-700 dark:bg-red-950/30 dark:text-red-300"
        >
          {warning}
        </p>
      ) : null}
    </details>
  );
}

export default MissionCapabilityPicker;
