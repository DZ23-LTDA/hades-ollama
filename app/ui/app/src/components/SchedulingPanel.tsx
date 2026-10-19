// Painel de agendamento -- a superfície do maior domínio sem UI (14 rotas).
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - NÃO trata motor ausente como lista vazia. O servidor devolve 503 quando o
//     motor não está ligado, e diz porquê; o painel mostra os dois estados como
//     diferentes.
//   - NÃO esconde o buffer: `busy_until` é o intervalo REAL em que o recurso não
//     pode receber outro atendimento, e é diferente de `ends_at`.
//   - NÃO inventa rótulo para estado desconhecido: mostra o valor.
//   - NÃO cria nem cancela: este painel só lê (`services` e `appointments`).
import { useCallback, useEffect, useState } from "react";
import {
  contarAgendamentos,
  descreverBuffer,
  descreverEstadoDoAgendamento,
  descreverQuando,
  listarAgendamentos,
  listarServicosDeAgendamento,
  MotorDeAgendamentoAusenteError,
  type Agendamento,
  type ServicoDeAgendamento,
} from "@/lib/schedulingClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-motor" | "erro";

export function SchedulingPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [servicos, setServicos] = useState<ServicoDeAgendamento[]>([]);
  const [agendamentos, setAgendamentos] = useState<Agendamento[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const [respostaServicos, respostaAgendamentos] = await Promise.all([
        listarServicosDeAgendamento(),
        listarAgendamentos(),
      ]);
      setServicos(respostaServicos.services ?? []);
      setAgendamentos(respostaAgendamentos.appointments ?? []);
      setEstado("pronta");
    } catch (causa) {
      setServicos([]);
      setAgendamentos([]);
      if (causa instanceof MotorDeAgendamentoAusenteError) {
        setErro(causa.message);
        setEstado("sem-motor");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler o agendamento.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const contagem = contarAgendamentos(agendamentos);

  return (
    <section
      data-testid="scheduling"
      aria-labelledby="scheduling-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="scheduling-titulo" className="text-sm font-semibold text-neutral-100">
          Agendamento
        </h3>
        <button
          type="button"
          data-testid="scheduling-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="scheduling-a-carregar" className="text-xs text-neutral-400">
          A ler o agendamento...
        </p>
      )}

      {estado === "sem-motor" && (
        <div data-testid="scheduling-sem-motor" className="text-xs text-amber-300">
          <p className="font-semibold">O motor de agendamento não está ligado neste runtime.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há serviços»: é «não há motor». Uma lista vazia aqui mentiria.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="scheduling-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && (
        <>
          <h4 className="mb-1 text-xs font-semibold text-neutral-300">
            Serviços ({servicos.length})
          </h4>
          {servicos.length === 0 ? (
            <p data-testid="scheduling-sem-servicos" className="mb-3 text-xs text-neutral-400">
              O motor está ligado e não há serviços definidos nesta organização.
            </p>
          ) : (
            <table
              data-testid="scheduling-servicos"
              className="mb-4 w-full text-left text-xs text-neutral-200"
            >
              <thead className="text-neutral-400">
                <tr>
                  <th scope="col" className="py-1 pr-2">serviço</th>
                  <th scope="col" className="py-1 pr-2">duração</th>
                  <th scope="col" className="py-1 pr-2">buffer</th>
                </tr>
              </thead>
              <tbody>
                {servicos.map((servico) => (
                  <tr key={servico.id} className="border-t border-neutral-800">
                    <td className="py-1 pr-2">{servico.name}</td>
                    <td className="py-1 pr-2">{servico.duration_minutes} min</td>
                    <td className="py-1 pr-2 text-neutral-400">
                      {servico.buffer_minutes ? `${servico.buffer_minutes} min` : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          <h4 className="mb-1 text-xs font-semibold text-neutral-300">
            Agendamentos ({contagem.total})
          </h4>
          <p data-testid="scheduling-contagem" className="mb-2 text-xs text-neutral-300">
            {contagem.marcados} marcado(s) · {contagem.cancelados} cancelado(s) ·{" "}
            {contagem.faltas} falta(s)
            {contagem.desconhecidos > 0 && ` · ${contagem.desconhecidos} desconhecido(s)`}
          </p>

          {agendamentos.length === 0 ? (
            <p data-testid="scheduling-sem-agendamentos" className="text-xs text-neutral-400">
              O motor está ligado e não há agendamentos nesta organização.
            </p>
          ) : (
            <table
              data-testid="scheduling-agendamentos"
              className="w-full text-left text-xs text-neutral-200"
            >
              <thead className="text-neutral-400">
                <tr>
                  <th scope="col" className="py-1 pr-2">início</th>
                  <th scope="col" className="py-1 pr-2">fim</th>
                  <th scope="col" className="py-1 pr-2">estado</th>
                </tr>
              </thead>
              <tbody>
                {agendamentos.map((agendamento) => {
                  const buffer = descreverBuffer(agendamento);
                  return (
                    <tr
                      key={agendamento.id}
                      data-testid={`scheduling-agendamento-${agendamento.id}`}
                      className="border-t border-neutral-800"
                    >
                      <td className="py-1 pr-2">{descreverQuando(agendamento.starts_at)}</td>
                      <td className="py-1 pr-2 text-neutral-400">
                        {descreverQuando(agendamento.ends_at)}
                        {buffer && (
                          <span
                            data-testid={`scheduling-buffer-${agendamento.id}`}
                            className="ml-1 text-[10px] text-amber-300"
                          >
                            {buffer}
                          </span>
                        )}
                      </td>
                      <td className="py-1 pr-2">{descreverEstadoDoAgendamento(agendamento.status)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </>
      )}

      <p data-testid="scheduling-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê serviços e agendamentos. Marcar, cancelar, remarcar e gerir a lista de espera
        continuam a ser ações separadas — e o buffer mostrado é o intervalo real em que o recurso não
        pode receber outro atendimento.
      </p>
    </section>
  );
}
