import { useMemo } from "react";

import {
  activatedSkillIDs,
  parseTokenBudget,
  skillSkipLabel,
  skillUsageLabel,
  type AgentSkillActivation,
  type AgentSkillSelection,
  type SkillContextInput,
} from "@/lib/skillCatalog";
import type { AgentSkill } from "@/lib/agenticClient";
import { SkillVersionsCard } from "@/components/SkillVersionsCard";

export type SkillCatalogPanelProps = {
  skills: AgentSkill[];
  activations: AgentSkillActivation[];
  selection: AgentSkillSelection | null;
  context: SkillContextInput;
  onContextChange: (context: SkillContextInput) => void;
  onResolve: () => void;
  onActivate: (skillID: string, context: SkillContextInput) => void;
  onDeactivate: (skillID: string, context: SkillContextInput) => void;
  pending?: boolean;
  error?: string | null;
};

/**
 * Catálogo de skills com ativação contextual e orçamento.
 *
 * O painel reflete o que o servidor selecionou: skill não confiável aparece
 * bloqueada para ativação, exclusões mostram o motivo, e nada é anunciado como
 * "pronto" por estar listado.
 */
export function SkillCatalogPanel({
  skills,
  activations,
  selection,
  context,
  onContextChange,
  onResolve,
  onActivate,
  onDeactivate,
  pending = false,
  error = null,
}: SkillCatalogPanelProps) {
  const activeIDs = useMemo(() => new Set(activatedSkillIDs(activations, context)), [activations, context]);
  // O erro de orçamento é DERIVADO do contexto: se a tela receber um valor
  // inválido de fora, ela precisa avisar — não só quando o usuário digita.
  const budgetError = useMemo(() => parseTokenBudget(context.tokenBudget).error, [context.tokenBudget]);

  const updateContext = (patch: Partial<SkillContextInput>) => {
    onContextChange({ ...context, ...patch });
  };

  return (
    <div className="space-y-4">
      <div className="grid gap-3 md:grid-cols-4">
        <label className="text-xs text-neutral-600 dark:text-neutral-300">
          Projeto
          <input
            aria-label="Projeto do contexto"
            value={context.projectId}
            onChange={(event) => updateContext({ projectId: event.target.value })}
            placeholder="(qualquer)"
            className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="text-xs text-neutral-600 dark:text-neutral-300">
          Agente
          <input
            aria-label="Agente do contexto"
            value={context.agentId}
            onChange={(event) => updateContext({ agentId: event.target.value })}
            placeholder="(qualquer)"
            className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="text-xs text-neutral-600 dark:text-neutral-300">
          Modelo
          <input
            aria-label="Modelo do contexto"
            value={context.model}
            onChange={(event) => updateContext({ model: event.target.value })}
            placeholder="(qualquer)"
            className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="text-xs text-neutral-600 dark:text-neutral-300">
          Orçamento de tokens
          <input
            aria-label="Orçamento de tokens"
            value={context.tokenBudget}
            onChange={(event) => updateContext({ tokenBudget: event.target.value })}
            placeholder="sem limite"
            className="mt-1 h-10 w-full rounded-xl border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="button"
          onClick={onResolve}
          disabled={pending || Boolean(budgetError)}
          aria-busy={pending}
          className="rounded-xl bg-neutral-900 px-4 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-white dark:text-neutral-900"
        >
          {pending ? "Resolvendo…" : "Resolver catálogo para este contexto"}
        </button>
        <p className="text-xs text-neutral-500 dark:text-neutral-400">{skillUsageLabel(selection)}</p>
      </div>
      {budgetError ? (
        <p role="alert" className="text-xs text-red-600 dark:text-red-400">
          {budgetError}
        </p>
      ) : null}
      {error ? (
        <p role="alert" className="text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      <ul className="space-y-3" aria-label="Skills registradas">
        {skills.length === 0 ? (
          <li className="text-xs text-neutral-500 dark:text-neutral-400">
            Nenhuma skill registrada no servidor. Instale um manifesto assinado para vê-la aqui.
          </li>
        ) : null}
        {skills.map((skill) => {
          const ativada = activeIDs.has(skill.id);
          const confiavel = skill.trusted === true;
          return (
            <li
              key={skill.id}
              className="rounded-xl border border-neutral-200/80 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-900"
            >
              {/* E-644: o HISTÓRICO de versões desta skill, com restauro. A confiança devolvida
                  pelo servidor é mostrada como vem — restaurar recupera conteúdo, não confiança. */}
              <SkillVersionsCard skillId={skill.id} activeVersion={skill.version} />
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-medium text-neutral-900 dark:text-white">
                    {skill.id} <span className="text-xs text-neutral-400">v{skill.version}</span>
                  </p>
                  <p className="mt-1 text-xs text-neutral-500 dark:text-neutral-400">
                    {skill.description || "sem descrição"}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`rounded-full px-2 py-1 text-[10px] font-medium ${
                      confiavel
                        ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300"
                        : "bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300"
                    }`}
                  >
                    {confiavel ? "confiável" : "não confiável"}
                  </span>
                  <button
                    type="button"
                    disabled={!confiavel || pending}
                    title={confiavel ? undefined : "O servidor recusa ativar skill não confiável"}
                    onClick={() => (ativada ? onDeactivate(skill.id, context) : onActivate(skill.id, context))}
                    className="rounded-lg border border-neutral-200 px-2 py-1 text-[10px] disabled:opacity-40 dark:border-neutral-700"
                  >
                    {ativada ? "Desativar" : "Ativar"}
                  </button>
                </div>
              </div>
              <p className="mt-2 text-[10px] text-neutral-400">
                categoria: {skill.category || "não declarada"} · custo:{" "}
                {skill.token_budget ? `${skill.token_budget} tokens` : "não declarado"} · escopos:{" "}
                {skill.scopes?.length ? skill.scopes.join(", ") : "nenhum"}
              </p>
            </li>
          );
        })}
      </ul>

      {selection ? (
        <div className="rounded-xl border border-neutral-200/80 bg-neutral-50/60 p-3 dark:border-neutral-800 dark:bg-neutral-950/30">
          <h3 className="text-xs font-semibold text-neutral-900 dark:text-white">
            Seleção do servidor para este contexto
          </h3>
          <p className="mt-1 text-[11px] text-neutral-500 dark:text-neutral-400">
            A ordem é determinística e o orçamento é aplicado pelo servidor — a tela não decide sozinha.
          </p>
          <ul className="mt-2 space-y-1 text-[11px] text-neutral-600 dark:text-neutral-300">
            {(selection.selected ?? []).length === 0 ? <li>Nenhuma skill entrou para este contexto.</li> : null}
            {(selection.selected ?? []).map((skill) => (
              <li key={skill.id}>
                • {skill.id} {skill.category ? `(${skill.category})` : ""} — {skill.token_budget || 0} tokens
              </li>
            ))}
          </ul>
          {selection.skipped?.length ? (
            <ul className="mt-2 space-y-1 text-[11px] text-amber-700 dark:text-amber-300">
              {selection.skipped.map((skip) => (
                <li key={skip.id}>• excluída — {skillSkipLabel(skip)}</li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
