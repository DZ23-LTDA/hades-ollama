// Fase de oportunidade: mover, e a regra de que fase terminal não reabre.
//
// Painel de ESCRITA, separado do `CRMPanel` de leitura: um mostra o funil, o outro move uma
// oportunidade. Juntá-los faria o ecrã de leitura carregar a responsabilidade de uma acção.
import { useCallback, useState } from "react";
import {
  descreverMudanca,
  descreverRegraTerminal,
  FaseRecusadaError,
  FaseTerminalError,
  faseTerminal,
  FASES,
  mudarFase,
  validarFase,
  validarOportunidade,
  type OportunidadeDeCRM,
} from "@/lib/crmStageClient";

type Estado = "pronto" | "a-mover" | "movida" | "terminal" | "recusada" | "erro";

export function CrmStagePanel() {
  const [id, setId] = useState("");
  const [fase, setFase] = useState<string>("qualified");
  const [faseActual, setFaseActual] = useState("");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [oportunidade, setOportunidade] = useState<OportunidadeDeCRM | null>(null);

  const mover = useCallback(async () => {
    const problema = validarOportunidade(id) ?? validarFase(fase);
    if (problema) {
      setErro(problema);
      setEstado("recusada");
      return;
    }
    if (faseTerminal(faseActual)) {
      setErro(
        `A oportunidade está em «${faseActual}»: reabrir apagaria o resultado. O servidor recusa, e o painel também.`,
      );
      setEstado("terminal");
      return;
    }

    setErro(null);
    setEstado("a-mover");
    try {
      const r = await mudarFase({ id, stage: fase, faseActual });
      setOportunidade(r);
      setEstado("movida");
    } catch (causa) {
      setOportunidade(null);
      if (causa instanceof FaseTerminalError) {
        setErro(causa.message);
        setEstado("terminal");
        return;
      }
      if (causa instanceof FaseRecusadaError) {
        setErro(causa.message);
        setEstado("recusada");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível mover.");
      setEstado("erro");
    }
  }, [id, fase, faseActual]);

  return (
    <section
      data-testid="crm-fase"
      aria-labelledby="crm-fase-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <h3 id="crm-fase-titulo" className="mb-3 text-sm font-semibold text-neutral-100">
        Mover oportunidade no funil
      </h3>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="crm-fase-id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          placeholder="id da oportunidade"
          aria-label="Id da oportunidade"
          className="w-48 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="crm-fase-actual"
          value={faseActual}
          onChange={(e) => setFaseActual(e.target.value)}
          placeholder="fase actual (opcional)"
          aria-label="Fase actual conhecida"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <select
          data-testid="crm-fase-destino"
          value={fase}
          onChange={(e) => setFase(e.target.value)}
          aria-label="Fase de destino"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        >
          {FASES.map((f) => (
            <option key={f} value={f}>
              {f}
            </option>
          ))}
        </select>
        <button
          type="button"
          data-testid="crm-fase-mover"
          onClick={() => void mover()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Mover
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="crm-fase-pronto" className="text-xs text-neutral-400">
          Nenhuma oportunidade movida nesta sessão.
        </p>
      )}

      {estado === "a-mover" && (
        <p data-testid="crm-fase-a-mover" className="text-xs text-neutral-400">
          A mover...
        </p>
      )}

      {estado === "movida" && (
        <p data-testid="crm-fase-movida" className="text-xs font-semibold text-emerald-300">
          movida · {descreverMudanca(oportunidade)}
        </p>
      )}

      {/* Fase terminal: e' a REGRA, nao uma avaria. */}
      {estado === "terminal" && (
        <p data-testid="crm-fase-terminal" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusada" && (
        <p data-testid="crm-fase-recusada" className="text-xs text-amber-300">
          O servidor recusou: {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="crm-fase-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="crm-fase-regra" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverRegraTerminal()}
      </p>
    </section>
  );
}
