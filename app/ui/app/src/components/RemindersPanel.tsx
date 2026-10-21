// Lembretes devidos -- a metade do "quem deve ser avisado".
//
// NAO faz:
//   - NAO envia `now` (quem escolhe "agora" e' o servidor);
//   - NAO se chama "lembretes enviados" -- e' so' a metade pendente;
//   - NAO assume "todos os tipos" quando o `kind` esta' vazio.
import { useCallback, useState } from "react";
import {
  descreverJanela,
  descreverLembrete,
  descreverMetade,
  LembretesAusenteError,
  LembretesInvalidosError,
  listarLembretesDevidos,
  validarAntecedencia,
  validarTipo,
  type RespostaDeLembretes,
} from "@/lib/remindersClient";

type EstadoDoPainel = "pronto" | "a-carregar" | "pronta" | "invalido" | "ausente" | "erro";

export function RemindersPanel() {
  const [kind, setKind] = useState("24h");
  const [lead, setLead] = useState("");
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeLembretes | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const consultar = useCallback(async (tipo: string, antecedencia: string) => {
    const erroTipo = validarTipo(tipo);
    if (erroTipo) {
      setR(null);
      setErro(erroTipo);
      setEstado("invalido");
      return;
    }
    const erroLead = validarAntecedencia(antecedencia);
    if (erroLead) {
      setR(null);
      setErro(erroLead);
      setEstado("invalido");
      return;
    }

    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await listarLembretesDevidos(tipo, antecedencia || undefined);
      setR(resposta);
      setEstado("pronta");
    } catch (causa) {
      setR(null);
      if (causa instanceof LembretesInvalidosError) {
        setErro(causa.message);
        setEstado("invalido");
        return;
      }
      if (causa instanceof LembretesAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler os lembretes.");
      setEstado("erro");
    }
  }, []);

  const lista = r?.appointments ?? [];

  return (
    <section
      data-testid="lembretes"
      aria-labelledby="lembretes-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="lembretes-titulo" className="text-sm font-semibold text-neutral-100">
          Lembretes devidos
        </h3>
        <div className="flex items-center gap-2">
          <input
            data-testid="lembretes-kind"
            value={kind}
            onChange={(e) => setKind(e.target.value)}
            placeholder="tipo (ex.: 24h)"
            aria-label="Tipo de lembrete"
            className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <input
            data-testid="lembretes-lead"
            value={lead}
            onChange={(e) => setLead(e.target.value)}
            placeholder="antecedência (min)"
            aria-label="Antecedência em minutos"
            className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <button
            type="button"
            data-testid="lembretes-consultar"
            onClick={() => void consultar(kind, lead)}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Consultar
          </button>
        </div>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="lembretes-a-carregar" className="text-xs text-neutral-400">
          A consultar os lembretes...
        </p>
      )}

      {estado === "invalido" && (
        <p data-testid="lembretes-invalido" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="lembretes-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="lembretes-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && r && (
        <>
          {/* A metade que esta rota da'. */}
          <p data-testid="lembretes-metade" className="text-xs font-semibold text-neutral-200">
            {descreverMetade(r)}
          </p>

          <p data-testid="lembretes-janela" className="text-[11px] text-neutral-500">
            {descreverJanela(r)}
          </p>

          {lista.length > 0 && (
            <ul data-testid="lembretes-lista" className="mt-2 space-y-0.5 text-[11px] text-neutral-300">
              {lista.map((l, i) => (
                <li key={`${l.appointment_id}-${i}`} data-testid={`lembrete-${l.appointment_id}`}>
                  {descreverLembrete(l)}
                </li>
              ))}
            </ul>
          )}

          {lista.length === 0 && (
            <p data-testid="lembretes-vazio" className="mt-1 text-[11px] text-neutral-400">
              O servidor não devolveu nenhum agendamento para esta janela.
            </p>
          )}
        </>
      )}

      <p data-testid="lembretes-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Esta lista é <strong>quem deve ser avisado agora</strong> — não é o registo do que já foi
        enviado. O instante «agora» é do servidor: este painel não o escolhe.
      </p>
    </section>
  );
}
