// Remarcar agendamento -- escrita com confirmação, e a recusa vem do MOTOR.
//
// NAO faz:
//   - NAO remarca ao montar (e' uma ESCRITA);
//   - NAO valida turno/sobreposicao/fechamento (isso e' do motor, e ele tem a informacao);
//   - NAO apresenta remarcacao como cancelar+criar (a identidade do agendamento mantem-se).
import { useCallback, useState } from "react";
import {
  descreverFronteiraDaRemarcacao,
  descreverMovimento,
  descreverRemarcacao,
  remarcarAgendamento,
  RemarcacaoAusenteError,
  RemarcacaoRecusadaError,
  type RespostaDeRemarcacao,
} from "@/lib/rescheduleMutationClient";

type EstadoDoPainel = "pronto" | "a-remarcar" | "remarcado" | "recusado" | "ausente" | "erro";

export function RescheduleMutationPanel() {
  const [id, setId] = useState("");
  const [novoInicio, setNovoInicio] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeRemarcacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const remarcar = useCallback(async () => {
    if (!confirmado) {
      setErro("Confirme antes de remarcar: isto move um agendamento real.");
      setEstado("erro");
      return;
    }

    setErro(null);
    setEstado("a-remarcar");
    try {
      const resposta = await remarcarAgendamento({ appointmentId: id, startsAt: novoInicio });
      setR(resposta);
      setEstado("remarcado");
    } catch (causa) {
      setR(null);
      if (causa instanceof RemarcacaoAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      if (causa instanceof RemarcacaoRecusadaError) {
        setErro(causa.message);
        setEstado("recusado");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível remarcar.");
      setEstado("erro");
    }
  }, [id, novoInicio, confirmado]);

  const ag = r?.appointment ?? null;

  return (
    <section
      data-testid="remarcar"
      aria-labelledby="remarcar-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="remarcar-titulo" className="text-sm font-semibold text-neutral-100">
          Remarcar agendamento
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="remarcar-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto move um agendamento real
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="remarcar-id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          placeholder="appointment_id"
          aria-label="Identificador do agendamento"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="remarcar-inicio"
          value={novoInicio}
          onChange={(e) => setNovoInicio(e.target.value)}
          placeholder="novo início (RFC 3339)"
          aria-label="Novo início"
          className="w-52 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="remarcar-enviar"
          onClick={() => void remarcar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Remarcar
        </button>
      </div>

      <p data-testid="remarcar-movimento" className="mb-2 text-[11px] text-neutral-500">
        {descreverMovimento("", novoInicio)}
      </p>

      {estado === "pronto" && (
        <p data-testid="remarcar-pronto" className="text-xs text-neutral-400">
          Nada remarcado nesta sessão. Preencha o novo instante e confirme.
        </p>
      )}

      {estado === "a-remarcar" && (
        <p data-testid="remarcar-a-remarcar" className="text-xs text-neutral-400">
          A remarcar...
        </p>
      )}

      {/* A recusa e' do MOTOR -- mostrada como recusa, nao como erro do painel. */}
      {estado === "recusado" && (
        <p data-testid="remarcar-recusado" className="text-xs text-amber-300">
          O motor recusou este instante: {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="remarcar-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="remarcar-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "remarcado" && ag && (
        <>
          <p data-testid="remarcar-resultado" className="text-xs font-semibold text-emerald-300">
            {descreverRemarcacao({ appointment: ag })}
          </p>
          <p data-testid="remarcar-identidade" className="mt-1 text-[11px] text-neutral-400">
            É o MESMO agendamento ({ag.id}) noutro instante — não um cancelamento seguido de uma marcação.
          </p>
        </>
      )}

      <p data-testid="remarcar-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverFronteiraDaRemarcacao()} Uma rota que decidisse o que cabe <strong>duplicaria o motor com
        menos informação</strong>.
      </p>
    </section>
  );
}
