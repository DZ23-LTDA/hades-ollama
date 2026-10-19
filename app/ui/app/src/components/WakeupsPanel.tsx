// Despertar de agentes -- o painel que distingue "acordou" de "vai trabalhar".
//
// NAO faz:
//   - NAO mostra so' `enqueued` (esconderia as missoes que o sistema viu);
//   - NAO mostra so' `wakeups` (sugeriria que todas vao correr);
//   - NAO trata falha parcial como sucesso;
//   - NAO presume que o agregado e' global -- le' o `scope`.
import { useCallback, useState } from "react";
import {
  descreverEscopo,
  despertarAgentes,
  eInformativo,
  idDoDespertar,
  motivoDoDespertar,
  resumirDespertares,
  resumirInformativo,
  WakeupsSemEfeitoError,
  type RespostaDeWakeups,
} from "@/lib/wakeupsClient";

type EstadoDoPainel = "pronto" | "a-executar" | "feito" | "sem-efeito" | "erro";

export function WakeupsPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [r, setR] = useState<RespostaDeWakeups | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const executar = useCallback(async () => {
    setErro(null);
    setEstado("a-executar");
    try {
      const resposta = await despertarAgentes();
      setR(resposta);
      setEstado("feito");
    } catch (causa) {
      setR(null);
      if (causa instanceof WakeupsSemEfeitoError) {
        setErro(causa.message);
        setEstado("sem-efeito");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel despertar.");
      setEstado("erro");
    }
  }, []);

  const despertares = r?.wakeups ?? [];
  const enfileirados = r?.enqueued ?? [];
  const falhas = r?.failed ?? [];
  const informativo = r ? resumirInformativo(r) : null;

  return (
    <section
      data-testid="wakeups"
      aria-labelledby="wakeups-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-2">
        <h3 id="wakeups-titulo" className="text-sm font-semibold text-neutral-100">
          Despertar de agentes
        </h3>
        <button
          type="button"
          data-testid="wakeups-executar"
          onClick={() => void executar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Despertar
        </button>
      </header>

      {estado === "a-executar" && (
        <p data-testid="wakeups-a-executar" className="text-xs text-neutral-400">
          A calcular os despertares...
        </p>
      )}

      {/* Nada foi enfileirado E houve falhas: NAO e' sucesso parcial. */}
      {estado === "sem-efeito" && (
        <p data-testid="wakeups-sem-efeito" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="wakeups-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "feito" && r && (
        <>
          <p data-testid="wakeups-resumo" className="text-xs font-semibold text-neutral-200">
            {resumirDespertares(r)}
          </p>

          {/* O escopo e' declarado pelo servidor, nao presumido global. */}
          <p data-testid="wakeups-escopo" className="text-[11px] text-neutral-500">
            {descreverEscopo(r)}
          </p>

          {/* A diferenca que nao se pode colapsar. */}
          {informativo && (
            <p data-testid="wakeups-informativo" className="mt-1 text-[11px] text-amber-300">
              {informativo}
            </p>
          )}

          {falhas.length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-red-300">
                Falhas ({falhas.length}) — não escondem as restantes
              </h4>
              <ul data-testid="wakeups-falhas" className="space-y-0.5 text-[11px] text-red-300">
                {falhas.map((f, i) => (
                  <li key={`${f}-${i}`}>{f}</li>
                ))}
              </ul>
            </>
          )}

          {despertares.length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
                Despertares ({despertares.length})
              </h4>
              <ul data-testid="wakeups-lista" className="space-y-0.5 text-[11px]">
                {despertares.map((d, i) => (
                  <li
                    key={idDoDespertar(d, i)}
                    data-testid={`wakeups-item-${idDoDespertar(d, i)}`}
                    className={eInformativo(d) ? "text-amber-300" : "text-neutral-400"}
                  >
                    <span className="font-semibold text-neutral-200">
                      {idDoDespertar(d, i)}
                    </span>{" "}
                    · {motivoDoDespertar(d)}
                    {eInformativo(d) && <span className="text-amber-400"> — informativo</span>}
                  </li>
                ))}
              </ul>
            </>
          )}

          {enfileirados.length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-emerald-300">
                Enfileirados ({enfileirados.length}) — estes VÃO correr
              </h4>
              <ul
                data-testid="wakeups-enfileirados"
                className="space-y-0.5 text-[11px] text-emerald-300"
              >
                {enfileirados.map((e, i) => (
                  <li key={`${e}-${i}`}>{e}</li>
                ))}
              </ul>
            </>
          )}

          {despertares.length === 0 && enfileirados.length === 0 && falhas.length === 0 && (
            <p data-testid="wakeups-vazio" className="mt-1 text-[11px] text-neutral-400">
              O servidor não viu nenhuma missão a despertar.
            </p>
          )}
        </>
      )}

      <p data-testid="wakeups-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Esta rota é segura de repetir: enfileirar é <strong>idempotente por missão</strong>. Uma
        missão que espera aprovação produz despertar <strong>informativo</strong> — não é
        enfileirada, porque isso seria executá-la sem a decisão que ela própria pediu.
      </p>
    </section>
  );
}
