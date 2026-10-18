/**
 * FASE 28.8 do adendo V3: o painel de ROLLBACK dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL FAZ, E O QUE ELE DEIXA CLARO
 *
 * Pede o **plano** de rollback para um canal e mostra o que o servidor mediu. **Nao restaura.**
 *
 *  1. **Diz que so' planeia** antes de haver resultado -- *um plano nao e' uma restauracao*.
 *  2. **Mostra o aviso de fronteira do backend por inteiro** -- e o aviso garante que nenhum ficheiro
 *     foi movido.
 *  3. **Um canal nao suportado nao e' um erro do produto:** 409 com a explicacao. O painel mostra a
 *     explicacao, nao um alerta vermelho.
 *  4. **Nao existe botao de restaurar.** A rota nao oferece, o cliente nao tem, o painel nao inventa:
 *
 *	*Um botao morto e' pior do que um botao ausente: o ausente nao mente.*
 *
 * ACESSIBILIDADE
 *
 * Regiao com nome, rotulo no seletor, `aria-live` no estado e no resultado, erro em `role="alert"`.
 */

import { useCallback, useState } from "react";

import {
  canaisDeRollback,
  descreverCanalDeRollback,
  planearRollback,
  provaDoPlanoDeRollback,
  resumirPlanoDeRollback,
  type PlanoDeRollback,
  type TipoDeRollback,
} from "@/lib/rollbackClient";

type Estado = "inativo" | "a-planear" | "pronto" | "erro";

/** Extrai a mensagem de uma falha sem inventar uma. */
function mensagemDeFalha(falha: unknown): string {
  if (falha instanceof Error && falha.message.trim() !== "") {
    return falha.message;
  }
  return "nao foi possivel obter o plano de rollback";
}

export function RollbackPanel() {
  const [tipo, setTipo] = useState<TipoDeRollback>("WINDOWS_INSTALLER");
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState<string>("");
  const [plano, setPlano] = useState<PlanoDeRollback | null>(null);

  const planear = useCallback(async () => {
    setEstado("a-planear");
    setErro("");
    setPlano(null);
    try {
      setPlano(await planearRollback(tipo));
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, [tipo]);

  return (
    <section
      aria-labelledby="rollback-titulo"
      data-testid="rollback"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="rollback-titulo" className="text-lg font-semibold">
        Plano de rollback
      </h2>
      {/* Dizer o que faz ANTES do botao: o utilizador tem de saber que isto NAO restaura. */}
      <p className="mt-1 text-sm text-neutral-400">
        Este painel apenas <strong>planeia</strong> o rollback: mede o SHA-256 dos artefactos
        anteriores e propoe o mais recente. Nada e' restaurado, movido ou apagado -- restaurar exige
        aprovacao humana no momento da accao, e este painel nao tem essa accao.
      </p>

      <div className="mt-3 flex flex-col gap-2">
        <label className="flex flex-col gap-1 text-sm">
          <span>Canal</span>
          <select
            id="rollback-canal"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={tipo}
            onChange={(evento) => setTipo(evento.target.value as TipoDeRollback)}
          >
            {canaisDeRollback().map((canal) => (
              <option key={canal} value={canal}>
                {descreverCanalDeRollback(canal)}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void planear()}
          disabled={estado === "a-planear"}
        >
          Planear rollback
        </button>
      </div>

      {estado === "a-planear" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="rollback-a-planear">
          A medir os candidatos...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="rollback-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && plano !== null ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="rollback-plano">
          <p className="text-sm font-medium" data-testid="rollback-resumo">
            {resumirPlanoDeRollback(plano)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="rollback-prova">
            {provaDoPlanoDeRollback(plano)}
          </p>
          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Canal</dt>
            <dd data-testid="rollback-canal-escolhido">{descreverCanalDeRollback(plano.tipo)}</dd>
            <dt className="text-neutral-400">Candidatos</dt>
            <dd data-testid="rollback-candidatos">{plano.total_de_candidatos}</dd>
            <dt className="text-neutral-400">Integro</dt>
            <dd data-testid="rollback-integro">{plano.integro ? "sim" : "nao"}</dd>
            <dt className="text-neutral-400">Executado</dt>
            <dd data-testid="rollback-executado">{plano.executado ? "SIM" : "nao"}</dd>
          </dl>
          {plano.alvo !== undefined ? (
            <div className="text-xs">
              <p className="text-neutral-400">Alvo proposto</p>
              <p className="font-mono" data-testid="rollback-alvo">
                {plano.alvo.nome}
              </p>
              <p className="break-all text-neutral-500">{plano.alvo.sha256}</p>
            </div>
          ) : null}
          {plano.requer_aprovacao !== undefined && plano.requer_aprovacao !== "" ? (
            <p className="text-xs text-amber-400" data-testid="rollback-requer-aprovacao">
              {plano.requer_aprovacao}
            </p>
          ) : null}
          {/* O aviso vem do backend, por inteiro. */}
          <p className="text-xs text-neutral-400" data-testid="rollback-aviso-fronteira">
            {plano.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
