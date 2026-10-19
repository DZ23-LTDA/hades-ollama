// Cancelar agendamento -- escrita destrutiva, com motivo e confirmação.
//
// NAO faz:
//   - NAO cancela ao montar (e' uma ESCRITA DESTRUTIVA);
//   - NAO chama o servidor sem motivo (o motivo e' a unica parte auditavel);
//   - NAO assume "cancelado" -- mostra o estado que o servidor devolveu.
import { useCallback, useState } from "react";
import {
  cancelarAgendamento,
  CancelamentoAusenteError,
  CancelamentoInvalidoError,
  descreverCancelamento,
  descreverOQueSePerde,
  validarIdAgendamento,
  validarMotivo,
  type RespostaDeCancelamento,
} from "@/lib/cancellationClient";

type EstadoDoPainel = "pronto" | "a-cancelar" | "cancelado" | "invalido" | "ausente" | "erro";

export function CancellationPanel() {
  const [id, setId] = useState("");
  const [motivo, setMotivo] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeCancelamento | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const cancelar = useCallback(async () => {
    const erroId = validarIdAgendamento(id);
    if (erroId) {
      setR(null);
      setErro(erroId);
      setEstado("invalido");
      return;
    }
    const erroMotivo = validarMotivo(motivo);
    if (erroMotivo) {
      setR(null);
      setErro(erroMotivo);
      setEstado("invalido");
      return;
    }
    if (!confirmado) {
      setErro("Confirme antes de cancelar: isto liberta um horário e não se desfaz sozinho.");
      setEstado("erro");
      return;
    }

    setErro(null);
    setEstado("a-cancelar");
    try {
      const resposta = await cancelarAgendamento({ appointmentId: id, reason: motivo });
      setR(resposta);
      setEstado("cancelado");
    } catch (causa) {
      setR(null);
      if (causa instanceof CancelamentoInvalidoError) {
        setErro(causa.message);
        setEstado("invalido");
        return;
      }
      if (causa instanceof CancelamentoAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível cancelar.");
      setEstado("erro");
    }
  }, [id, motivo, confirmado]);

  return (
    <section
      data-testid="cancelar"
      aria-labelledby="cancelar-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="cancelar-titulo" className="text-sm font-semibold text-neutral-100">
          Cancelar agendamento
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="cancelar-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto cancela um agendamento real
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="cancelar-id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          placeholder="appointment_id"
          aria-label="Identificador do agendamento"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="cancelar-motivo"
          value={motivo}
          onChange={(e) => setMotivo(e.target.value)}
          placeholder="motivo (obrigatório)"
          aria-label="Motivo"
          className="w-56 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="cancelar-enviar"
          onClick={() => void cancelar()}
          className="rounded border border-red-800 px-2 py-1 text-xs text-red-300 hover:bg-red-950/40"
        >
          Cancelar agendamento
        </button>
      </div>

      {/* O motivo e' dito antes do botao, nao depois do erro. */}
      <p data-testid="cancelar-o-que-se-perde" className="mb-2 text-[11px] text-neutral-500">
        {descreverOQueSePerde(null, motivo)}
      </p>

      {estado === "pronto" && (
        <p data-testid="cancelar-pronto" className="text-xs text-neutral-400">
          Nada cancelado nesta sessão. Isto é uma escrita destrutiva: preencha o motivo e confirme.
        </p>
      )}

      {estado === "a-cancelar" && (
        <p data-testid="cancelar-a-cancelar" className="text-xs text-neutral-400">
          A cancelar...
        </p>
      )}

      {estado === "invalido" && (
        <p data-testid="cancelar-invalido" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="cancelar-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="cancelar-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "cancelado" && r && (
        <p data-testid="cancelar-resultado" className="text-xs font-semibold text-emerald-300">
          {descreverCancelamento(r)}
        </p>
      )}

      <p data-testid="cancelar-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        O motivo é obrigatório porque <strong>um cancelamento sem motivo não é auditável</strong> — explica
        o que aconteceu, não o porquê. E o <code>appointment_id</code> vai no <strong>corpo</strong>, não no
        caminho: um id concatenado num caminho é um id que pode reescrever o caminho.
      </p>
    </section>
  );
}
