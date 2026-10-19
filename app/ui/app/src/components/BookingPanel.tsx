// Marcar agendamento -- escrita com confirmação, idempotência visível e buffer dito.
//
// NAO faz:
//   - NAO marca ao montar (e' uma ESCRITA);
//   - NAO inventa o id quando o servidor responde 409 (a reentrega e' dita como reentrega);
//   - NAO mostra `ends_at` como se fosse o fim da ocupacao (o buffer de `busy_until` e' dito).
import { useCallback, useState } from "react";
import {
  bufferEmMinutos,
  descreverIdempotencia,
  descreverMarcacao,
  descreverOcupacao,
  marcarAgendamento,
  MarcacaoAusenteError,
  MarcacaoInvalidaError,
  ReentregaDetectadaError,
  validarAlvo,
  validarInicio,
  type RespostaDeMarcacao,
} from "@/lib/bookingClient";

type EstadoDoPainel = "pronto" | "a-marcar" | "marcado" | "reentrega" | "invalido" | "ausente" | "erro";

export function BookingPanel() {
  const [recurso, setRecurso] = useState("");
  const [servico, setServico] = useState("");
  const [inicio, setInicio] = useState("");
  const [contacto, setContacto] = useState("");
  const [canal, setCanal] = useState("");
  const [chave, setChave] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeMarcacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const marcar = useCallback(async () => {
    const erroAlvo = validarAlvo(recurso, servico);
    if (erroAlvo) {
      setR(null);
      setErro(erroAlvo);
      setEstado("invalido");
      return;
    }
    const erroInicio = validarInicio(inicio);
    if (erroInicio) {
      setR(null);
      setErro(erroInicio);
      setEstado("invalido");
      return;
    }
    if (!confirmado) {
      setErro("Confirme antes de marcar: isto cria um agendamento real.");
      setEstado("erro");
      return;
    }

    setErro(null);
    setEstado("a-marcar");
    try {
      const resposta = await marcarAgendamento({
        resourceId: recurso,
        serviceId: servico,
        startsAt: inicio,
        contactId: contacto,
        channel: canal,
        idempotencyKey: chave,
      });
      setR(resposta);
      setEstado(resposta.idempotent_replay ? "reentrega" : "marcado");
    } catch (causa) {
      setR(null);
      if (causa instanceof ReentregaDetectadaError) {
        setErro(causa.message);
        setEstado("reentrega");
        return;
      }
      if (causa instanceof MarcacaoInvalidaError) {
        setErro(causa.message);
        setEstado("invalido");
        return;
      }
      if (causa instanceof MarcacaoAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível marcar.");
      setEstado("erro");
    }
  }, [recurso, servico, inicio, contacto, canal, chave, confirmado]);

  const ag = r?.appointment ?? null;
  const buffer = ag ? bufferEmMinutos(ag) : null;

  return (
    <section
      data-testid="marcar"
      aria-labelledby="marcar-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="marcar-titulo" className="text-sm font-semibold text-neutral-100">
          Marcar agendamento
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="marcar-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto cria um agendamento
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="marcar-recurso"
          value={recurso}
          onChange={(e) => setRecurso(e.target.value)}
          placeholder="recurso"
          aria-label="Recurso"
          className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="marcar-servico"
          value={servico}
          onChange={(e) => setServico(e.target.value)}
          placeholder="serviço"
          aria-label="Serviço"
          className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="marcar-inicio"
          value={inicio}
          onChange={(e) => setInicio(e.target.value)}
          placeholder="RFC 3339"
          aria-label="Início"
          className="w-44 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="marcar-contacto"
          value={contacto}
          onChange={(e) => setContacto(e.target.value)}
          placeholder="contacto"
          aria-label="Contacto"
          className="w-24 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="marcar-canal"
          value={canal}
          onChange={(e) => setCanal(e.target.value)}
          placeholder="canal"
          aria-label="Canal"
          className="w-20 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="marcar-chave"
          value={chave}
          onChange={(e) => setChave(e.target.value)}
          placeholder="idempotency_key"
          aria-label="Chave de idempotência"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="marcar-enviar"
          onClick={() => void marcar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Marcar
        </button>
      </div>

      {/* A chave de idempotência, dita antes de alguém carregar no botão. */}
      <p data-testid="marcar-idempotencia" className="mb-2 text-[11px] text-neutral-500">
        {descreverIdempotencia(chave)}
      </p>

      {estado === "pronto" && (
        <p data-testid="marcar-pronto" className="text-xs text-neutral-400">
          Nada marcado nesta sessão. Isto é uma escrita: preencha e confirme para marcar.
        </p>
      )}

      {estado === "a-marcar" && (
        <p data-testid="marcar-a-marcar" className="text-xs text-neutral-400">
          A marcar...
        </p>
      )}

      {estado === "invalido" && (
        <p data-testid="marcar-invalido" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="marcar-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="marcar-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {/* 409: o servidor tem o agendamento, mas o transporte nao o entrega. Dizemos o que sabemos. */}
      {estado === "reentrega" && (
        <p data-testid="marcar-reentrega" className="text-xs text-emerald-300">
          {r ? descreverMarcacao(r) : erro}
        </p>
      )}

      {estado === "marcado" && r && ag && (
        <>
          <p data-testid="marcar-resultado" className="text-xs font-semibold text-emerald-300">
            {descreverMarcacao(r)}
          </p>

          <p data-testid="marcar-ocupacao" className="mt-1 text-[11px] text-neutral-300">
            {descreverOcupacao(ag)}
          </p>

          {buffer !== null && buffer > 0 && (
            <p data-testid="marcar-buffer" className="mt-1 text-[11px] text-amber-300">
              o atendimento termina às {ag.ends_at}, mas o recurso fica ocupado até {ag.busy_until}
            </p>
          )}
        </>
      )}

      <p data-testid="marcar-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        O contrato desta superfície é <strong>separado</strong> do motor: um campo com o nome errado
        daria um corpo vazio e um erro a culpar o cliente. E <code>busy_until</code> inclui o buffer —
        o recurso não fica livre quando o atendimento acaba.
      </p>
    </section>
  );
}
