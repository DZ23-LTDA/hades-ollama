import { useState } from "react";
import {
  OPERACOES_DE_MIDIA,
  motivoDeIndisponibilidade,
  operacoesDisponiveis,
  resumoDoEstudio,
  type DescritorDeOperacao,
  type OperacaoDeMidia,
} from "@/lib/mediaOperacoes";
import {
  criarMissaoDeMidia,
  executarMissaoDeMidia,
} from "@/lib/mediaStudioApi";

/**
 * Multimodal Studio (Estagio 45 / item 45).
 *
 * *Nao ha motor aqui e nao ha rota nova de geracao:* o painel conduz uma MISSAO com a capacidade
 * `media:execute`, e a tool `media.process` faz o resto (aprovacao, workspace fixado, artefacto).
 *
 * **O que o painel nunca faz** (proibido pela meta): fingir video/TTS que nao existem. Sem provider
 * so^ aparece o que corre localmente, e o resto diz a razao LITERAL do servidor.
 */

/** Texto do objetivo enviado a missao - *a operacao e os campos vao declarados, nao adivinhados.* */
export function objetivoDaOperacao(
  descritor: DescritorDeOperacao,
  entradas: Record<string, string>,
): string {
  const partes = descritor.campos
    .filter((campo) => (entradas[campo] ?? "").trim() !== "")
    .map((campo) => `${campo}=${entradas[campo]}`);
  const detalhe = partes.length > 0 ? ` (${partes.join(", ")})` : "";
  return `Midia: ${descritor.operacao}${detalhe}`;
}

/** Linha de estado de cada operacao no ecra^ - *vazia nunca, e a razao e' sempre dita.* */
export function estadoDaOperacao(
  descritor: DescritorDeOperacao,
  providerConfigurado: boolean,
): string {
  const motivo = motivoDeIndisponibilidade(
    descritor.operacao,
    providerConfigurado,
  );
  if (motivo) {
    return `Indisponivel agora: ${motivo}`;
  }
  return descritor.exigeProvider
    ? "Disponivel (provider configurado)"
    : "Disponivel (local)";
}

export function MediaStudioPanel({
  workspace,
  organizationId,
  providerConfigurado,
}: {
  workspace: string;
  organizationId?: string;
  providerConfigurado: boolean;
}) {
  const [escolhida, setEscolhida] = useState<OperacaoDeMidia | null>(null);
  const [entradas, setEntradas] = useState<Record<string, string>>({});
  const [missao, setMissao] = useState<string | null>(null);
  const [recado, setRecado] = useState<string | null>(null);

  const disponiveis = operacoesDisponiveis(providerConfigurado);
  const descritor =
    OPERACOES_DE_MIDIA.find((d) => d.operacao === escolhida) ?? null;

  const lancar = async () => {
    if (!descritor) return;
    setRecado(null);
    setMissao(null);
    try {
      const criada = await criarMissaoDeMidia(
        objetivoDaOperacao(descritor, entradas),
        workspace,
        organizationId,
      );
      await executarMissaoDeMidia(criada.id);
      setMissao(criada.id);
      // A tool exige aprovacao: dizer isto e' parte da honestidade, nao um detalhe tecnico.
      setRecado(
        "Missao criada e lancada. A operacao de midia exige aprovacao antes de correr.",
      );
    } catch (erro) {
      setRecado(
        `O servidor recusou: ${erro instanceof Error ? erro.message : String(erro)}`,
      );
    }
  };

  return (
    <section className="flex flex-col gap-3 rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
      <header className="flex flex-col gap-1">
        <h3 className="font-semibold">Estudio multimodal</h3>
        <p className="text-neutral-500" data-estudio-resumo>
          {resumoDoEstudio(providerConfigurado)}
        </p>
      </header>

      {recado ? (
        <p
          className="rounded bg-neutral-100 p-2 dark:bg-neutral-900"
          data-estudio-recado
        >
          {recado}
        </p>
      ) : null}
      {missao ? (
        <p className="font-mono text-[11px] text-neutral-500">
          missao: {missao}
        </p>
      ) : null}

      <ul className="flex flex-col gap-1">
        {OPERACOES_DE_MIDIA.map((item) => {
          const utilizavel = disponiveis.some(
            (d) => d.operacao === item.operacao,
          );
          return (
            <li key={item.operacao} className="flex flex-col gap-0.5">
              <button
                type="button"
                className="w-fit rounded border border-neutral-300 px-2 py-1 text-left hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:hover:bg-neutral-900"
                disabled={!utilizavel}
                onClick={() => {
                  setEscolhida(item.operacao);
                  setEntradas({});
                }}
              >
                {item.rotulo}{" "}
                <span className="text-neutral-500">({item.operacao})</span>
              </button>
              <span
                className="text-[11px] text-neutral-500"
                data-estado-operacao={item.operacao}
              >
                {estadoDaOperacao(item, providerConfigurado)}
              </span>
            </li>
          );
        })}
      </ul>

      {descritor ? (
        <div className="flex flex-col gap-1 border-t border-neutral-200 pt-2 dark:border-neutral-800">
          {descritor.campos.map((campo) => (
            <label key={campo} className="flex flex-col gap-0.5">
              <span className="text-[11px] text-neutral-500">{campo}</span>
              <input
                className="rounded border border-neutral-300 px-2 py-1 dark:border-neutral-700 dark:bg-neutral-950"
                value={entradas[campo] ?? ""}
                onChange={(evento) =>
                  setEntradas((anterior) => ({
                    ...anterior,
                    [campo]: evento.target.value,
                  }))
                }
              />
            </label>
          ))}
          <button
            type="button"
            className="mt-1 w-fit rounded bg-neutral-900 px-3 py-1 font-semibold text-white hover:bg-neutral-800 dark:bg-white dark:text-neutral-900"
            onClick={() => void lancar()}
          >
            Lancar missao de midia
          </button>
        </div>
      ) : null}
    </section>
  );
}
