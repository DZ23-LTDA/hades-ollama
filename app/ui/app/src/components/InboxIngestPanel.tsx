// Ingestão de mensagens no inbox.
//
// 201 nova e 200 reentrega são ESTADOS DISTINTOS. Um painel que os mostra como «sucesso» esconde a única
// coisa que o utilizador precisa de saber: se a mensagem chegou agora ou se já cá estava.
import { useCallback, useState } from "react";
import {
  descreverDeduplicacao,
  descreverEntrega,
  InboxAusenteError,
  InboxRecusadoError,
  receberMensagem,
  validarMensagem,
  type ResultadoDeInbox,
} from "@/lib/inboxIngestClient";

type Estado = "pronto" | "a-receber" | "nova" | "reentrega" | "ausente" | "recusada" | "erro";

export function InboxIngestPanel() {
  const [canal, setCanal] = useState("whatsapp");
  const [corpo, setCorpo] = useState("");
  const [contacto, setContacto] = useState("");
  const [externo, setExterno] = useState("");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [r, setR] = useState<ResultadoDeInbox | null>(null);

  const receber = useCallback(async () => {
    const problema = validarMensagem(canal, corpo);
    if (problema) {
      setErro(problema);
      setEstado("recusada");
      return;
    }

    setErro(null);
    setEstado("a-receber");
    try {
      const resposta = await receberMensagem({
        channel: canal,
        body: corpo,
        contactId: contacto,
        externalId: externo,
      });
      setR(resposta);
      // Dois estados distintos: 201 nova, 200 reentrega.
      setEstado(resposta.deduplicated === true ? "reentrega" : "nova");
      setCorpo("");
    } catch (causa) {
      setR(null);
      if (causa instanceof InboxAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      if (causa instanceof InboxRecusadoError) {
        setErro(causa.message);
        setEstado("recusada");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível receber.");
      setEstado("erro");
    }
  }, [canal, corpo, contacto, externo]);

  return (
    <section
      data-testid="inbox-ingest"
      aria-labelledby="inbox-ingest-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="inbox-ingest-titulo" className="text-sm font-semibold text-neutral-100">
          Receber mensagem no inbox
        </h3>
        <span className="text-[11px] text-neutral-500">201 nova · 200 reentrega</span>
      </header>

      <div className="mb-3 flex flex-wrap items-start gap-2">
        <select
          data-testid="inbox-ingest-canal"
          value={canal}
          onChange={(e) => setCanal(e.target.value)}
          aria-label="Canal"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        >
          <option value="whatsapp">whatsapp</option>
          <option value="telegram">telegram</option>
          <option value="email">email</option>
          <option value="sms">sms</option>
        </select>
        <input
          data-testid="inbox-ingest-contacto"
          value={contacto}
          onChange={(e) => setContacto(e.target.value)}
          placeholder="contact_id (opcional)"
          aria-label="Id do contacto"
          className="w-36 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="inbox-ingest-externo"
          value={externo}
          onChange={(e) => setExterno(e.target.value)}
          placeholder="external_id (chave da dedup.)"
          aria-label="Chave externa para deduplicação"
          className="w-48 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <textarea
          data-testid="inbox-ingest-corpo"
          value={corpo}
          onChange={(e) => setCorpo(e.target.value)}
          placeholder="corpo da mensagem"
          aria-label="Corpo da mensagem"
          rows={2}
          className="w-64 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="inbox-ingest-receber"
          onClick={() => void receber()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Receber
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="inbox-ingest-pronto" className="text-xs text-neutral-400">
          Nada recebido nesta sessão.
        </p>
      )}

      {estado === "a-receber" && (
        <p data-testid="inbox-ingest-a-receber" className="text-xs text-neutral-400">
          A receber...
        </p>
      )}

      {/* 201: mensagem NOVA. */}
      {estado === "nova" && (
        <p data-testid="inbox-ingest-nova" className="text-xs font-semibold text-emerald-300">
          {descreverEntrega(r)}
        </p>
      )}

      {/* 200: REENTREGA. Mesmo resultado, afirmacao diferente -- e nao pode parecer "nova". */}
      {estado === "reentrega" && (
        <p data-testid="inbox-ingest-reentrega" className="text-xs font-semibold text-amber-300">
          {descreverEntrega(r)} — não foi criada uma segunda conversa.
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="inbox-ingest-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusada" && (
        <p data-testid="inbox-ingest-recusada" className="text-xs text-amber-300">
          O servidor recusou: {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="inbox-ingest-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="inbox-ingest-dedup" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverDeduplicacao(externo)}
      </p>
    </section>
  );
}
