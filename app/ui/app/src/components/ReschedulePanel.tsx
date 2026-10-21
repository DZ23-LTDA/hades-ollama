// Horários para remarcar -- o motor responde ANTES de o cliente escolher.
//
// NAO faz:
//   - NAO deixa remarcar sem agendamento E data (valida os dois, diz qual falta);
//   - NAO esconde um slot que se declara ocupado numa lista de "livres";
//   - NAO apresenta "0 horários" como sucesso.
import { useCallback, useState } from "react";
import {
  contarNaoLivres,
  descreverAlvo,
  descreverDisponibilidade,
  descreverDuracoes,
  listarHorariosParaRemarcar,
  RemarcacaoAusenteError,
  RemarcacaoInvalidaError,
  validarPasso,
  validarPedido,
  type RespostaDeRemarcacao,
} from "@/lib/rescheduleClient";

type EstadoDoPainel = "pronto" | "a-carregar" | "pronta" | "invalido" | "ausente" | "erro";

export function ReschedulePanel() {
  const [appointmentId, setAppointmentId] = useState("");
  const [data, setData] = useState("");
  const [passo, setPasso] = useState("30");
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeRemarcacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const consultar = useCallback(async (id: string, dia: string, p: string) => {
    const erroPedido = validarPedido(id, dia);
    if (erroPedido) {
      setR(null);
      setErro(erroPedido);
      setEstado("invalido");
      return;
    }
    const erroPasso = validarPasso(p);
    if (erroPasso) {
      setR(null);
      setErro(erroPasso);
      setEstado("invalido");
      return;
    }

    setErro(null);
    setEstado("a-carregar");
    try {
      const resposta = await listarHorariosParaRemarcar(id, dia, p || undefined);
      setR(resposta);
      setEstado("pronta");
    } catch (causa) {
      setR(null);
      if (causa instanceof RemarcacaoInvalidaError) {
        setErro(causa.message);
        setEstado("invalido");
        return;
      }
      if (causa instanceof RemarcacaoAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler os horarios.");
      setEstado("erro");
    }
  }, []);

  const slots = r?.slots ?? [];
  const naoLivres = r ? contarNaoLivres(r) : 0;

  return (
    <section
      data-testid="remarcar"
      aria-labelledby="remarcar-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="remarcar-titulo" className="text-sm font-semibold text-neutral-100">
          Horários para remarcar
        </h3>
        <div className="flex items-center gap-2">
          <input
            data-testid="remarcar-id"
            value={appointmentId}
            onChange={(e) => setAppointmentId(e.target.value)}
            placeholder="agendamento"
            aria-label="Agendamento a remarcar"
            className="w-36 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <input
            data-testid="remarcar-data"
            value={data}
            onChange={(e) => setData(e.target.value)}
            placeholder="AAAA-MM-DD"
            aria-label="Data a remarcar"
            className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <input
            data-testid="remarcar-passo"
            value={passo}
            onChange={(e) => setPasso(e.target.value)}
            placeholder="passo (min)"
            aria-label="Passo em minutos"
            className="w-24 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <button
            type="button"
            data-testid="remarcar-consultar"
            onClick={() => void consultar(appointmentId, data, passo)}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Consultar
          </button>
        </div>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="remarcar-a-carregar" className="text-xs text-neutral-400">
          A consultar os horários livres...
        </p>
      )}

      {estado === "invalido" && (
        <p data-testid="remarcar-invalido" className="text-xs text-amber-300">
          {erro}
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

      {estado === "pronta" && r && (
        <>
          <p data-testid="remarcar-alvo" className="text-xs font-semibold text-neutral-200">
            {descreverAlvo(appointmentId, data)}
          </p>

          <p data-testid="remarcar-resumo" className="text-[11px] text-neutral-300">
            {descreverDisponibilidade(r)}
          </p>

          <p data-testid="remarcar-duracao" className="text-[11px] text-neutral-500">
            {descreverDuracoes(r)}
          </p>

          {naoLivres > 0 && (
            <p data-testid="remarcar-contradicao" className="mt-1 text-[11px] text-amber-300">
              {naoLivres} horário(s) aparecem como livres mas declaram-se ocupados
            </p>
          )}

          {slots.length === 0 ? (
            <p data-testid="remarcar-vazio" className="mt-1 text-[11px] text-neutral-400">
              O motor não devolveu nenhum horário livre para este dia.
            </p>
          ) : (
            <ul data-testid="remarcar-lista" className="mt-2 flex flex-wrap gap-1 text-[11px]">
              {slots.map((s, i) => (
                <li
                  key={`${s.starts_at}-${i}`}
                  data-testid={`remarcar-slot-${i}`}
                  className={
                    s.free
                      ? "rounded border border-emerald-800 px-2 py-0.5 text-emerald-400"
                      : "rounded border border-amber-800 px-2 py-0.5 text-amber-300"
                  }
                >
                  {s.starts_at}
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="remarcar-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Remarcar sem saber o que está livre obriga o cliente a adivinhar — e a adivinhação vira
        horário inexistente, que o motor recusaria <strong>depois</strong> de o cliente escolher. Esta
        lista é a resposta <strong>antes</strong>.
      </p>
    </section>
  );
}
