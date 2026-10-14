import { Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useId, useRef, useState } from "react";
import { ArrowLeftIcon, SparklesIcon } from "@heroicons/react/24/outline";
import { agentErrorMessage, createMission } from "@/lib/agenticClient";
import { AppSidebar } from "@/components/AppSidebar";
import { SidebarLayout } from "@/components/layout/layout";

// The runtime rejects objectives above 8 KiB; keep the client limit below it so
// multi-byte text is still validated by the server instead of being truncated.
export const MAX_OBJECTIVE_LENGTH = 8000;

export function NewTaskPage() {
  const navigate = useNavigate();
  const [objective, setObjective] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const errorRef = useRef<HTMLDivElement>(null);
  const objectiveRef = useRef<HTMLTextAreaElement>(null);
  const hintId = useId();
  const validationId = useId();

  useEffect(() => {
    if (submitError) errorRef.current?.focus();
  }, [submitError]);

  const submit = async () => {
    if (submitting) return;
    const trimmed = objective.trim();
    if (!trimmed) {
      setValidationError("Descreva o objetivo da tarefa antes de criar.");
      objectiveRef.current?.focus();
      return;
    }
    setValidationError(null);
    setSubmitError(null);
    setSubmitting(true);
    try {
      // Same safe defaults as the Agentic Console: read-only workspace and no
      // automatic run. Execution stays behind explicit approval.
      const mission = await createMission({
        objective: trimmed,
        capabilities: ["workspace:read"],
        auto_run: false,
      });
      await navigate({ to: "/tasks", search: { created: mission.id } });
    } catch (cause) {
      setSubmitError(
        agentErrorMessage(cause, "Não foi possível criar a tarefa."),
      );
      setSubmitting(false);
    }
  };

  return (
    <SidebarLayout title="Nova tarefa" sidebar={<AppSidebar current="tasks" />}>
      <div className="min-h-0 flex-1 overflow-y-auto bg-neutral-50 dark:bg-neutral-900">
        <div className="mx-auto w-full max-w-3xl px-4 pb-14 pt-8 sm:px-6 lg:px-12">
          <Link
            to="/tasks"
            className="inline-flex items-center gap-1.5 rounded-md text-xs font-medium text-neutral-500 hover:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 dark:text-neutral-400 dark:hover:text-white"
          >
            <ArrowLeftIcon className="h-3.5 w-3.5" />
            Voltar para Tarefas
          </Link>
          <div className="mt-5 mb-3 flex items-center gap-2 text-xs font-medium text-violet-600 dark:text-violet-300">
            <SparklesIcon className="h-4 w-4" />
            Missão agentic
          </div>
          <h2 className="font-rounded text-3xl font-semibold tracking-tight text-neutral-950 dark:text-white">
            Nova tarefa
          </h2>
          <p className="mt-3 text-sm leading-6 text-neutral-500 dark:text-neutral-400">
            A tarefa é criada no runtime local com leitura do workspace e sem
            execução automática. Você acompanha plano, approvals e artifacts em
            Tarefas.
          </p>

          <form
            className="mt-8 rounded-2xl border border-neutral-200/80 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"
            aria-busy={submitting}
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void submit();
            }}
          >
            <label
              htmlFor="new-task-objective"
              className="text-sm font-medium text-neutral-900 dark:text-white"
            >
              Objetivo da tarefa
            </label>
            <textarea
              id="new-task-objective"
              ref={objectiveRef}
              value={objective}
              onChange={(event) => {
                setObjective(event.target.value);
                if (validationError) setValidationError(null);
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
                  event.preventDefault();
                  void submit();
                }
              }}
              rows={6}
              maxLength={MAX_OBJECTIVE_LENGTH}
              required
              disabled={submitting}
              aria-invalid={validationError ? true : undefined}
              aria-describedby={
                validationError ? `${hintId} ${validationId}` : hintId
              }
              placeholder="Ex.: Revisar o README e listar o que falta para rodar o projeto localmente"
              className="mt-2 w-full resize-y rounded-xl border border-neutral-300 bg-transparent px-3 py-2.5 text-sm text-neutral-900 placeholder:text-neutral-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500 disabled:opacity-60 aria-[invalid=true]:border-red-500 dark:border-neutral-700 dark:text-white"
            />
            <p
              id={hintId}
              className="mt-2 text-xs text-neutral-500 dark:text-neutral-400"
            >
              {objective.length}/{MAX_OBJECTIVE_LENGTH} caracteres · Ctrl+Enter
              para criar
            </p>
            {validationError && (
              <p
                id={validationId}
                className="mt-2 text-xs font-medium text-red-600 dark:text-red-400"
              >
                {validationError}
              </p>
            )}
            {submitError && (
              <div
                ref={errorRef}
                tabIndex={-1}
                role="alert"
                aria-live="assertive"
                aria-atomic="true"
                className="mt-4 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300"
              >
                <p className="font-medium">A tarefa não foi criada.</p>
                <p className="mt-1 text-xs">{submitError}</p>
              </div>
            )}
            {/* Visual order matches DOM/tab order at every width (WCAG 2.4.3). */}
            <div className="mt-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-end">
              <Link
                to="/tasks"
                className="inline-flex items-center justify-center rounded-xl border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-400 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                Cancelar
              </Link>
              <button
                type="submit"
                disabled={submitting}
                aria-busy={submitting}
                className="inline-flex items-center justify-center rounded-xl bg-neutral-950 px-4 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-neutral-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-500 focus-visible:ring-offset-2 disabled:cursor-wait disabled:opacity-60 dark:bg-white dark:text-neutral-950 dark:hover:bg-neutral-200"
              >
                {submitting
                  ? "Criando tarefa…"
                  : submitError
                    ? "Tentar novamente"
                    : "Criar tarefa"}
              </button>
            </div>
          </form>
        </div>
      </div>
    </SidebarLayout>
  );
}
