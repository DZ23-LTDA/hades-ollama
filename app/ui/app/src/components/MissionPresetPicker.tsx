import { useId } from "react";
import {
  describeSchedule,
  summarizePreset,
  type MissionPreset,
} from "@/lib/missionPresets";

type Props = {
  /** Presets servidos por GET /api/agent/v1/presets. */
  presets: MissionPreset[];
  /** Id escolhido, ou `null` para correr sem preset. */
  selected: string | null;
  onSelect: (id: string | null) => void;
  /** `true` quando o catálogo não pôde ser lido do runtime. */
  unavailable?: boolean;
  disabled?: boolean;
};

/**
 * Seletor de PRESETS DE MISSÃO (FASE 23 do adendo V3).
 *
 * O que este componente faz e' o que o adendo exige: *o usuário pode inspecionar e
 * personalizar antes da execução.* Por isso cada preset aparece com o que ele PEDE
 * (permissões), o que ele CUSTA (orçamento) e quando ele ACABA (critérios de conclusão).
 *
 * Não cria missão, não autoriza nada e não esconde a escolha: só devolve o id.
 */
export function MissionPresetPicker({
  presets,
  selected,
  onSelect,
  unavailable = false,
  disabled = false,
}: Props) {
  const groupID = useId();

  if (unavailable) {
    return (
      <p
        role="status"
        className="mt-1 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200"
      >
        Catálogo de presets indisponível: a missão vai correr com o objetivo escrito à
        mão. Nenhum preset foi aplicado.
      </p>
    );
  }

  if (presets.length === 0) {
    return (
      <p
        role="status"
        className="mt-1 rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-[11px] text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300"
      >
        O runtime não publicou nenhum preset de missão. Escreva o objetivo à mão.
      </p>
    );
  }

  return (
    <fieldset
      className="mt-1 rounded-xl border border-neutral-200 bg-neutral-50 px-3 py-2 text-[11px] dark:border-neutral-800 dark:bg-neutral-900"
      disabled={disabled}
    >
      <legend className="px-1 font-medium text-neutral-700 dark:text-neutral-200">
        Preset de missão
      </legend>
      <p className="mb-2 text-neutral-600 dark:text-neutral-300">
        O preset define agentes, ferramentas, memória, permissões, orçamento e critérios de
        conclusão. Escolher um preset <strong>não</strong> concede permissão: o servidor
        aplica a política da mesma forma.
      </p>

      <label className="flex items-start gap-2 rounded-lg px-1 py-1">
        <input
          type="radio"
          name={groupID}
          checked={selected === null}
          onChange={() => onSelect(null)}
          className="mt-[2px]"
        />
        <span>
          <span className="font-medium text-neutral-800 dark:text-neutral-100">
            Sem preset
          </span>
          <span className="block text-neutral-600 dark:text-neutral-400">
            Usa apenas o objetivo escrito à mão.
          </span>
        </span>
      </label>

      {presets.map((preset) => {
        const summary = summarizePreset(preset);
        const schedule = describeSchedule(preset);
        return (
          <label
            key={preset.id}
            className="mt-1 flex items-start gap-2 rounded-lg px-1 py-1"
          >
            <input
              type="radio"
              name={groupID}
              checked={selected === preset.id}
              onChange={() => onSelect(preset.id)}
              className="mt-[2px]"
            />
            <span>
              <span className="font-medium text-neutral-800 dark:text-neutral-100">
                {preset.name}
                {schedule ? (
                  <span className="ml-2 rounded bg-neutral-200 px-1 py-[1px] text-[10px] font-normal text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300">
                    {schedule}
                  </span>
                ) : null}
              </span>
              {preset.description ? (
                <span className="block text-neutral-600 dark:text-neutral-400">
                  {preset.description}
                </span>
              ) : null}
              <span className="mt-1 block text-neutral-500 dark:text-neutral-400">
                {summary.budget ? (
                  <>Orçamento: {summary.budget}. </>
                ) : (
                  <>Sem orçamento declarado. </>
                )}
                {summary.permissionCount > 0 ? (
                  <>{summary.permissionCount} permissão(ões) pedida(s). </>
                ) : (
                  <>Nenhuma permissão extra pedida. </>
                )}
                {summary.criteriaCount} critério(s) de conclusão.
              </span>
            </span>
          </label>
        );
      })}
    </fieldset>
  );
}

export default MissionPresetPicker;