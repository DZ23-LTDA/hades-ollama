// Painel do Inbox -- conversas, mensagens, metricas e bloqueio por consentimento.
//
// O QUE ESTE PAINEL NAO FAZ, DE PROPOSITO:
//   - NAO trata `reply_blocked` como erro: a conversa continua visivel e legivel.
//   - NAO conta `deduplicated` como mensagem nova.
//   - NAO apresenta `last_message` como a conversa: e' uma previa, e o painel di-lo.
//   - NAO mostra segundos como segundos crus: formata.
//   - NAO trata inbox ausente (503) como lista vazia.
//   - NAO responde, atribui, faz handoff nem fecha: so' le.
import { useCallback, useEffect, useState } from "react";
import {
  descreverBloqueio,
  descreverEstadoConversa,
  descreverPrevia,
  formatarSegundos,
  InboxAusenteError,
  listarConversas,
  metricasDoInbox,
  obterConversa,
  type ConversaDoInbox,
  type DetalheDeConversa,
  type MetricasDoInbox,
} from "@/lib/inboxClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-inbox" | "erro";

export function InboxPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [conversas, setConversas] = useState<ConversaDoInbox[]>([]);
  const [metricas, setMetricas] = useState<MetricasDoInbox | null>(null);
  const [filtroEstado, setFiltroEstado] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [detalhe, setDetalhe] = useState<DetalheDeConversa | null>(null);
  const [detalheErro, setDetalheErro] = useState<string | null>(null);

  const carregar = useCallback(async (filtro: string) => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rConversas, rMetricas] = await Promise.all([
        listarConversas(filtro),
        metricasDoInbox(),
      ]);
      setConversas(rConversas.conversations ?? []);
      setMetricas(rMetricas);
      setEstado("pronta");
    } catch (causa) {
      setConversas([]);
      setMetricas(null);
      if (causa instanceof InboxAusenteError) {
        setErro(causa.message);
        setEstado("sem-inbox");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler o inbox.");
      setEstado("erro");
    }
  }, []);

  const abrir = useCallback(async (id: string) => {
    setDetalheErro(null);
    try {
      setDetalhe(await obterConversa(id));
    } catch (causa) {
      setDetalhe(null);
      setDetalheErro(
        causa instanceof Error ? causa.message : "Nao foi possivel ler a conversa.",
      );
    }
  }, []);

  useEffect(() => {
    void carregar("");
  }, [carregar]);

  return (
    <section
      data-testid="inbox"
      aria-labelledby="inbox-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="inbox-titulo" className="text-sm font-semibold text-neutral-100">
          Inbox
        </h3>
        <button
          type="button"
          data-testid="inbox-actualizar"
          onClick={() => void carregar(filtroEstado)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        <label className="flex flex-1 flex-col gap-1 text-xs text-neutral-300">
          Filtrar por estado
          <input
            type="text"
            data-testid="inbox-filtro"
            value={filtroEstado}
            onChange={(evento) => setFiltroEstado(evento.target.value)}
            placeholder="open, pending, closed..."
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>
        <button
          type="button"
          data-testid="inbox-filtrar"
          onClick={() => void carregar(filtroEstado)}
          className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800"
        >
          Filtrar
        </button>
      </div>

      {estado === "sem-inbox" && (
        <div data-testid="inbox-sem-inbox" className="text-xs text-amber-300">
          <p className="font-semibold">O inbox não está ligado neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há conversas»: é «não há inbox». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="inbox-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="inbox-a-carregar" className="text-xs text-neutral-400">
          A ler o inbox...
        </p>
      )}

      {estado === "pronta" && metricas && (
        <>
          <p data-testid="inbox-metricas" className="mb-1 text-xs text-neutral-300">
            {metricas.open_conversations} aberta(s) · {metricas.pending_human} à espera de humano ·{" "}
            <span className={metricas.sla_breached > 0 ? "text-red-400" : "text-emerald-400"}>
              {metricas.sla_breached} SLA de resposta violado(s)
            </span>{" "}
            · {metricas.resolution_overdue} resolução em atraso
          </p>
          <p data-testid="inbox-tempos" className="mb-3 text-[11px] text-neutral-400">
            primeira resposta média: {formatarSegundos(metricas.first_response_avg_seconds)} ·
            resolução média: {formatarSegundos(metricas.resolution_avg_seconds)} ·{" "}
            {metricas.messages_inbound} entrada(s) / {metricas.messages_outbound} saída(s)
          </p>

          <h4 className="mb-2 text-xs font-semibold text-neutral-300">
            Conversas ({conversas.length})
          </h4>

          {conversas.length === 0 ? (
            <p data-testid="inbox-sem-conversas" className="text-xs text-neutral-400">
              {filtroEstado.trim() !== ""
                ? `Nenhuma conversa com estado «${filtroEstado.trim()}».`
                : "O inbox está ligado e não há conversas nesta organização."}
            </p>
          ) : (
            <ul data-testid="inbox-conversas" className="space-y-1">
              {conversas.map((conversa) => (
                <li key={conversa.id} data-testid={`inbox-conversa-${conversa.id}`}>
                  <button
                    type="button"
                    data-testid={`inbox-abrir-${conversa.id}`}
                    onClick={() => void abrir(conversa.id)}
                    className="w-full rounded px-2 py-1 text-left text-xs text-neutral-200 hover:bg-neutral-800"
                  >
                    <span className="font-semibold">
                      {conversa.contact_name || conversa.contact_id}
                    </span>{" "}
                    <span className="text-neutral-400">
                      [{descreverEstadoConversa(conversa.status)}] · {conversa.channel} ·{" "}
                      {conversa.message_count} mensagem(ns)
                    </span>
                    <span className="block text-[11px] text-neutral-500">
                      {descreverPrevia(conversa)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}

          {detalheErro && (
            <p data-testid="inbox-detalhe-erro" role="alert" className="mt-2 text-xs text-red-400">
              {detalheErro}
            </p>
          )}

          {detalhe && (
            <div
              data-testid="inbox-detalhe"
              className="mt-3 rounded border border-neutral-700 bg-neutral-950/60 p-3 text-xs"
            >
              <h4 className="mb-1 font-semibold text-neutral-200">
                {detalhe.conversation.contact_name || detalhe.conversation.contact_id}
              </h4>
              {/* `reply_blocked` nao e' erro: a conversa continua legivel. */}
              <p
                data-testid="inbox-bloqueio"
                className={
                  detalhe.reply_blocked ? "text-amber-300" : "text-emerald-400"
                }
              >
                {descreverBloqueio(detalhe.reply_blocked, detalhe.reply_blocked_reason)}
              </p>
              {detalhe.reply_blocked && (
                <p data-testid="inbox-bloqueio-nota" className="mt-1 text-[11px] text-neutral-500">
                  A conversa continua legível. O que está bloqueado é responder neste canal, não ler.
                </p>
              )}

              {(detalhe.messages ?? []).length === 0 ? (
                <p data-testid="inbox-sem-mensagens" className="mt-2 text-neutral-400">
                  Não há mensagens nesta conversa.
                </p>
              ) : (
                <ul data-testid="inbox-mensagens" className="mt-2 space-y-1">
                  {(detalhe.messages ?? []).map((mensagem) => (
                    <li key={mensagem.id} className="text-neutral-300">
                      <span className="font-mono text-[10px] text-neutral-500">
                        {mensagem.direction} · {mensagem.author_type}
                      </span>{" "}
                      {mensagem.body}
                      {mensagem.deduplicated && (
                        <span
                          data-testid={`inbox-dedup-${mensagem.id}`}
                          className="ml-1 text-[10px] text-amber-400"
                        >
                          (reentrega — não é mensagem nova)
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </>
      )}

      <p data-testid="inbox-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê conversas, mensagens e métricas. Responder, atribuir, fazer handoff e fechar
        continuam a ser ações separadas — e os tempos vêm em segundos que aqui passam a minutos.
      </p>
    </section>
  );
}
