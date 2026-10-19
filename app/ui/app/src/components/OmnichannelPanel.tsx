// Painel de Omnicanal -- conectores, eventos e fila de mortos.
//
// O QUE ESTE PAINEL NAO FAZ, DE PROPOSITO:
//   - NAO mostra o valor de nenhum segredo: mostra o NOME da variavel.
//   - NAO presume durabilidade: se `persisted=false`, diz «fila em memoria».
//   - NAO esconde o erro de persistencia quando existe.
//   - NAO trata omnicanal ausente (503) como lista vazia.
//   - NAO registra conector, nao reprocessa DLQ: so' le.
import { useCallback, useEffect, useState } from "react";
import {
  descreverDurabilidade,
  descreverEntrega,
  descreverFalha,
  descreverLimite,
  descreverSegredo,
  listarConectores,
  listarEventos,
  listarLetrasMortas,
  OmnicanalAusenteError,
  type EstadoDaDLQ,
  type LetraMorta,
  type ConectorDeOmnicanal,
  type EventoDeOmnicanal,
} from "@/lib/omnichannelClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-omnicanal" | "erro";

export function OmnichannelPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [conectores, setConectores] = useState<ConectorDeOmnicanal[]>([]);
  const [eventos, setEventos] = useState<EventoDeOmnicanal[]>([]);
  const [letras, setLetras] = useState<LetraMorta[]>([]);
  const [dlq, setDlq] = useState<EstadoDaDLQ | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [rConectores, rEventos, rDlq] = await Promise.all([
        listarConectores(),
        listarEventos(),
        listarLetrasMortas(),
      ]);
      setConectores(rConectores.connectors ?? []);
      setEventos(rEventos.events ?? []);
      setLetras(rDlq.dead_letters ?? []);
      setDlq(rDlq.storage);
      setEstado("pronta");
    } catch (causa) {
      setConectores([]);
      setEventos([]);
      setLetras([]);
      setDlq(null);
      if (causa instanceof OmnicanalAusenteError) {
        setErro(causa.message);
        setEstado("sem-omnicanal");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler o omnicanal.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  return (
    <section
      data-testid="omnicanal"
      aria-labelledby="omnicanal-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="omnicanal-titulo" className="text-sm font-semibold text-neutral-100">
          Omnicanal
        </h3>
        <button
          type="button"
          data-testid="omnicanal-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "sem-omnicanal" && (
        <div data-testid="omnicanal-sem-omnicanal" className="text-xs text-amber-300">
          <p className="font-semibold">O runtime omnicanal não está configurado.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há conectores»: é «não há omnicanal». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="omnicanal-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="omnicanal-a-carregar" className="text-xs text-neutral-400">
          A ler o omnicanal...
        </p>
      )}

      {estado === "pronta" && (
        <>
          <h4 className="mb-2 text-xs font-semibold text-neutral-300">
            Conectores ({conectores.length})
          </h4>

          {conectores.length === 0 ? (
            <p data-testid="omnicanal-sem-conectores" className="text-xs text-neutral-400">
              O omnicanal está ligado e não há conectores nesta organização.
            </p>
          ) : (
            <ul data-testid="omnicanal-conectores" className="space-y-1">
              {conectores.map((conector) => (
                <li
                  key={conector.id}
                  data-testid={`omnicanal-conector-${conector.id}`}
                  className="rounded border border-neutral-800 px-2 py-1 text-xs text-neutral-200"
                >
                  <span className="font-semibold">{conector.name || conector.channel}</span>{" "}
                  <span className="text-neutral-400">
                    [{conector.enabled ? "activo" : "inactivo"}]
                  </span>
                  {/* O NOME da variavel, nunca o valor. */}
                  <span data-testid={`omnicanal-segredo-${conector.id}`} className="block text-[11px] text-neutral-500">
                    {descreverSegredo(conector.secret_env)}
                  </span>
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-2 mt-4 text-xs font-semibold text-neutral-300">
            Eventos ({eventos.length})
          </h4>

          {eventos.length === 0 ? (
            <p data-testid="omnicanal-sem-eventos" className="text-xs text-neutral-400">
              Não há eventos recebidos nesta organização.
            </p>
          ) : (
            <ul data-testid="omnicanal-eventos" className="space-y-0.5 text-[11px] text-neutral-300">
              {eventos.map((evento) => (
                <li key={evento.id}>
                  {evento.channel_id} · {evento.message_type} ·{" "}
                  {descreverEntrega(evento.delivery_state)}
                </li>
              ))}
            </ul>
          )}

          <h4 className="mb-2 mt-4 text-xs font-semibold text-neutral-300">
            Fila de mortos ({letras.length})
          </h4>

          {dlq && (
            <>
              <p
                data-testid="omnicanal-durabilidade"
                className={dlq.persisted ? "text-[11px] text-emerald-400" : "text-[11px] text-amber-400"}
              >
                {descreverDurabilidade(dlq)}
              </p>
              <p data-testid="omnicanal-limite" className="mb-2 text-[11px] text-neutral-500">
                {descreverLimite(dlq)}
              </p>
            </>
          )}

          {letras.length === 0 ? (
            <p data-testid="omnicanal-sem-letras" className="text-xs text-neutral-400">
              Nenhuma entrega falhou. Isto é diferente de «não há fila»: a fila existe e está vazia.
            </p>
          ) : (
            <ul data-testid="omnicanal-letras" className="space-y-0.5 text-[11px] text-neutral-300">
              {letras.map((letra) => (
                <li key={letra.id} data-testid={`omnicanal-letra-${letra.id}`} className="text-amber-300">
                  {letra.channel} · {descreverFalha(letra.reason, letra.attempts)}
                </li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="omnicanal-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê conectores, eventos e a fila de mortos. Registar conector e reprocessar uma
        entrega continuam a ser ações separadas — e o segredo aparece sempre pelo nome da variável,
        nunca pelo valor.
      </p>
    </section>
  );
}
