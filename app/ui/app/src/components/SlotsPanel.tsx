// Horários oferecíveis -- o que está REALMENTE aberto, não apenas sem reunião marcada.
//
// NAO faz:
//   - NAO assume "hoje" quando a data está vazia;
//   - NAO esconde um slot com `free=false` numa lista de abertos;
//   - NAO trata "0 slots" como "dia livre".
import { useCallback, useState } from "react";
import {
  contarContradicoes,
  descreverDuracoes,
  descreverHorarios,
  HorariosAusenteError,
  HorariosInvalidosError,
  listarHorarios,
  validarData,
  validarPasso,
  type RespostaDeHorarios,
} from "@/lib/slotsClient";

type EstadoDoPainel = "pronto" | "a-carregar" | "pronta" | "invalido" | "ausente" | "erro";

export function SlotsPanel() {
  const [data, setData] = useState("");
  const [passo, setPasso] = useState("30");
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeHorarios | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const consultar = useCallback(async (dia: string, p: string) => {
    const erroData = validarData(dia);
    if (erroData) {
      setR(null);
      setErro(erroData);
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
      const resposta = await listarHorarios(dia, p || undefined);
      setR(resposta);
      setEstado("pronta");
    } catch (causa) {
      setR(null);
      if (causa instanceof HorariosInvalidosError) {
        setErro(causa.message);
        setEstado("invalido");
        return;
      }
      if (causa instanceof HorariosAusenteError) {
        setErro(causa.message);
        setEstado("ausente");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler os horarios.");
      setEstado("erro");
    }
  }, []);

  const slots = r?.slots ?? [];
  const contradicoes = r ? contarContradicoes(r) : 0;
  const passoNum = Number(passo) || 30;

  return (
    <section
      data-testid="horarios"
      aria-labelledby="horarios-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="horarios-titulo" className="text-sm font-semibold text-neutral-100">
          Horários oferecíveis
        </h3>
        <div className="flex items-center gap-2">
          <input
            data-testid="horarios-data"
            value={data}
            onChange={(e) => setData(e.target.value)}
            placeholder="AAAA-MM-DD"
            aria-label="Data a consultar"
            className="w-32 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <input
            data-testid="horarios-passo"
            value={passo}
            onChange={(e) => setPasso(e.target.value)}
            placeholder="passo (min)"
            aria-label="Passo em minutos"
            className="w-28 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
          />
          <button
            type="button"
            data-testid="horarios-consultar"
            onClick={() => void consultar(data, passo)}
            className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
          >
            Consultar
          </button>
        </div>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="horarios-a-carregar" className="text-xs text-neutral-400">
          A consultar os horários...
        </p>
      )}

      {estado === "invalido" && (
        <p data-testid="horarios-invalido" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="horarios-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="horarios-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && r && (
        <>
          <p data-testid="horarios-resumo" className="text-xs font-semibold text-neutral-200">
            {descreverHorarios(r)}
          </p>

          <p data-testid="horarios-duracao" className="text-[11px] text-neutral-500">
            {descreverDuracoes(r, passoNum)}
          </p>

          {/* Uma lista de ABERTOS com `free=false` e' uma contradicao: mostrar, nao esconder. */}
          {contradicoes > 0 && (
            <p data-testid="horarios-contradicao" className="mt-1 text-[11px] text-amber-300">
              {contradicoes} horário(s) aparecem nesta lista como abertos mas declaram-se ocupados
            </p>
          )}

          {slots.length === 0 ? (
            <p data-testid="horarios-vazio" className="mt-1 text-[11px] text-neutral-400">
              O motor não ofereceu nenhum horário para este dia.
            </p>
          ) : (
            <ul data-testid="horarios-lista" className="mt-2 flex flex-wrap gap-1 text-[11px]">
              {slots.map((s, i) => (
                <li
                  key={`${s.starts_at}-${i}`}
                  data-testid={`horario-${i}`}
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

      <p data-testid="horarios-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Estes horários estão <strong>dentro do turno, fora de fechamento e sem ocupação</strong>. «Sem
        reunião marcada» não é o mesmo que «empresa aberta»: é essa a diferença entre disponibilidade
        e horário oferecível.
      </p>
    </section>
  );
}
