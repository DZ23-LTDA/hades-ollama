// Fila de espera e no-show -- três escritas, e a fila vazia não é falha.
//
// NAO faz:
//   - NAO oferece/confirma/marca ao montar (são três ESCRITAS);
//   - NAO transforma `offered: false` em erro (fila vazia é ausência de oferta);
//   - NAO junta "faltou" com "cancelou" num só estado;
//   - NAO calcula o prazo da oferta (não tem o relógio do servidor).
import { useCallback, useState } from "react";
import {
  AgendaAusenteError,
  AgendaRecusadaError,
  confirmarOferta,
  descreverNoShow,
  descreverOferta,
  descreverPrazoDaOferta,
  marcarNoShow,
  oferecerProximo,
  validarVaga,
  type OfertaDeEspera,
  type RespostaDeConfirmacao,
} from "@/lib/waitlistOpsClient";

type EstadoDoPainel = "pronto" | "a-correr" | "feito" | "recusado" | "ausente" | "erro";
type Operacao = "offer" | "confirm" | "no-show";

export function WaitlistOpsPanel() {
  const [ordem, setOrdem] = useState("");
  const [recurso, setRecurso] = useState("");
  const [slot, setSlot] = useState("");
  const [entrada, setEntrada] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [op, setOp] = useState<Operacao>("offer");
  const [oferta, setOferta] = useState<OfertaDeEspera | null>(null);
  const [ofereceu, setOfereceu] = useState(false);
  const [agendamento, setAgendamento] = useState<RespostaDeConfirmacao | null>(null);

  const correr = useCallback(
    async (operacao: Operacao) => {
      setOp(operacao);

      if (!confirmado) {
        setErro("Confirme antes de correr: isto escreve na agenda.");
        setEstado("erro");
        return;
      }

      if (operacao === "offer") {
        const e = validarVaga(recurso, slot);
        if (e) {
          setErro(e);
          setEstado("recusado");
          return;
        }
      }

      setErro(null);
      setEstado("a-correr");
      try {
        if (operacao === "offer") {
          const r = await oferecerProximo({ resourceId: recurso, slot });
          setOferta(r.offer ?? null);
          setOfereceu(r.offered);
          // `offered: false` NAO e' falha: e' ausencia de oferta.
          setEstado("feito");
          return;
        }
        if (operacao === "confirm") {
          const r = await confirmarOferta({ entryId: entrada });
          setAgendamento(r);
          setEstado("feito");
          return;
        }
        const r = await marcarNoShow({ appointmentId: ordem });
        setAgendamento(r);
        setEstado("feito");
      } catch (causa) {
        setOferta(null);
        setOfereceu(false);
        setAgendamento(null);
        if (causa instanceof AgendaAusenteError) {
          setErro(causa.message);
          setEstado("ausente");
          return;
        }
        if (causa instanceof AgendaRecusadaError) {
          setErro(causa.message);
          setEstado("recusado");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível correr.");
        setEstado("erro");
      }
    },
    [ordem, recurso, slot, entrada, confirmado],
  );

  return (
    <section
      data-testid="fila"
      aria-labelledby="fila-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="fila-titulo" className="text-sm font-semibold text-neutral-100">
          Fila de espera e faltas
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="fila-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto escreve na agenda
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="fila-ordem"
          value={ordem}
          onChange={(e) => setOrdem(e.target.value)}
          placeholder="appointment_id"
          aria-label="Identificador do agendamento"
          className="w-36 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="fila-recurso"
          value={recurso}
          onChange={(e) => setRecurso(e.target.value)}
          placeholder="recurso"
          aria-label="Recurso"
          className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="fila-slot"
          value={slot}
          onChange={(e) => setSlot(e.target.value)}
          placeholder="horário (RFC 3339)"
          aria-label="Horário"
          className="w-44 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="fila-entrada"
          value={entrada}
          onChange={(e) => setEntrada(e.target.value)}
          placeholder="entry_id"
          aria-label="Identificador da entrada"
          className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="fila-oferecer"
          onClick={() => void correr("offer")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Oferecer próximo
        </button>
        <button
          type="button"
          data-testid="fila-confirmar-oferta"
          onClick={() => void correr("confirm")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Confirmar oferta
        </button>
        <button
          type="button"
          data-testid="fila-faltou"
          onClick={() => void correr("no-show")}
          className="rounded border border-amber-800 px-2 py-1 text-xs text-amber-300 hover:bg-amber-950/40"
        >
          Marcar falta
        </button>
      </div>

      <p data-testid="fila-prazo" className="mb-2 text-[11px] text-neutral-500">
        {descreverPrazoDaOferta(oferta?.expires_at)}
      </p>

      {estado === "pronto" && (
        <p data-testid="fila-pronto" className="text-xs text-neutral-400">
          Nada corrido nesta sessão. São três escritas com regras diferentes — confirme para correr.
        </p>
      )}

      {estado === "a-correr" && (
        <p data-testid="fila-a-correr" className="text-xs text-neutral-400">
          A correr...
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="fila-recusado" className="text-xs text-amber-300">
          O motor recusou: {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="fila-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="fila-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "feito" && op === "offer" && (
        <p
          data-testid={ofereceu ? "fila-oferta" : "fila-vazia"}
          className={`text-xs font-semibold ${ofereceu ? "text-emerald-300" : "text-neutral-300"}`}
        >
          {descreverOferta({ offered: ofereceu, offer: oferta ?? undefined })}
        </p>
      )}

      {estado === "feito" && op === "confirm" && agendamento && (
        <p data-testid="fila-confirmada" className="text-xs font-semibold text-emerald-300">
          oferta confirmada · agendamento {agendamento.appointment?.id ?? "(sem id)"} criado
        </p>
      )}

      {estado === "feito" && op === "no-show" && agendamento && (
        <p data-testid="fila-falta" className="text-xs font-semibold text-amber-300">
          {descreverNoShow(agendamento)}
        </p>
      )}

      <p data-testid="fila-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        <strong>Quem faltou não é quem cancelou</strong> — a agenda distingue os dois. E a fila vazia
        devolve <code>offered: false</code> com 200: <strong>não é falha</strong>, é ausência de oferta.
        A oferta expira, e quem decide o prazo é o motor.
      </p>
    </section>
  );
}
