// Painel de roteamento multiobjectivo -- a superfície que faltava para `/routing`.
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - não escolhe modelo: mostra a escolha do servidor e o porquê;
//   - não inventa pesos nem recalcula o score no frontend;
//   - não trata o 422 como avaria: "nenhum modelo satisfaz" é uma decisão
//     honesta do servidor e o painel mostra-a com as restrições que a produziram;
//   - não esconde os factores por trás do score total;
//   - não trata a fronteira de Pareto como ranking: são alternativas, não uma
//     ordem.
//
//	*Um painel que confia cegamente no servidor não é um painel: é um espelho.*
import { useCallback, useEffect, useState } from "react";
import {
  consultarRota,
  descreverCusto,
  descreverFactores,
  PREFERENCIAS,
  SemRotaError,
  type ChaveDePreferencia,
  type RespostaDeRota,
} from "@/lib/routingClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-rota" | "erro";

export function RoutingPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [resposta, setResposta] = useState<RespostaDeRota | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const [capacidades, setCapacidades] = useState("");
  const [preferencia, setPreferencia] = useState<ChaveDePreferencia>("auto");
  const [apenasLocal, setApenasLocal] = useState(false);
  const [custoMaximo, setCustoMaximo] = useState("");
  const [caminho, setCaminho] = useState("");

  const consultar = useCallback(
    async (parametros: {
      capacidades: string;
      preferencia: ChaveDePreferencia;
      apenasLocal: boolean;
      custoMaximo: string;
      caminho: string;
    }) => {
      setErro(null);
      setEstado("a-carregar");
      try {
        const resultado = await consultarRota({
          capabilities: parametros.capacidades,
          preference: parametros.preferencia,
          localOnly: parametros.apenasLocal,
          maxCostCents: parametros.custoMaximo,
          path: parametros.caminho,
        });
        setResposta(resultado);
        setEstado("pronta");
      } catch (causa) {
        if (causa instanceof SemRotaError) {
          setResposta(null);
          setErro(causa.message);
          setEstado("sem-rota");
          return;
        }
        setResposta(null);
        setErro(causa instanceof Error ? causa.message : "Não foi possível consultar a rota.");
        setEstado("erro");
      }
    },
    [],
  );

  useEffect(() => {
    void consultar({
      capacidades: "",
      preferencia: "auto",
      apenasLocal: false,
      custoMaximo: "",
      caminho: "",
    });
  }, [consultar]);

  const aplicar = useCallback(() => {
    void consultar({ capacidades, preferencia, apenasLocal, custoMaximo, caminho });
  }, [apenasLocal, capacidades, caminho, consultar, custoMaximo, preferencia]);

  return (
    <section
      data-testid="routing"
      aria-labelledby="routing-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="routing-titulo" className="text-sm font-semibold text-neutral-100">
          Porquê esta IA
        </h3>
        <button
          type="button"
          data-testid="routing-consultar"
          onClick={aplicar}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Consultar
        </button>
      </header>

      <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label className="flex flex-col gap-1 text-xs text-neutral-300">
          Preferência
          <select
            data-testid="routing-preferencia"
            value={preferencia}
            onChange={(evento) => setPreferencia(evento.target.value as ChaveDePreferencia)}
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          >
            {PREFERENCIAS.map((opcao) => (
              <option key={opcao.chave} value={opcao.chave}>
                {opcao.rotulo}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-1 text-xs text-neutral-300">
          Capacidades exigidas (separadas por vírgula)
          <input
            type="text"
            data-testid="routing-capacidades"
            value={capacidades}
            onChange={(evento) => setCapacidades(evento.target.value)}
            placeholder="coding, vision"
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>

        <label className="flex flex-col gap-1 text-xs text-neutral-300">
          Custo máximo por 1k (cêntimos)
          <input
            type="text"
            data-testid="routing-custo"
            value={custoMaximo}
            onChange={(evento) => setCustoMaximo(evento.target.value)}
            placeholder="deixe vazio para sem limite"
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>

        <label className="flex flex-col gap-1 text-xs text-neutral-300">
          Path do pedido
          <input
            type="text"
            data-testid="routing-path"
            value={caminho}
            onChange={(evento) => setCaminho(evento.target.value)}
            placeholder="/api/chat (omissão do servidor)"
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          />
        </label>

        <label className="flex items-center gap-2 text-xs text-neutral-300">
          <input
            type="checkbox"
            data-testid="routing-local"
            checked={apenasLocal}
            onChange={(evento) => setApenasLocal(evento.target.checked)}
          />
          Somente local
        </label>
      </div>

      {estado === "a-carregar" && (
        <p data-testid="routing-a-carregar" className="text-xs text-neutral-400">
          A consultar o roteador...
        </p>
      )}

      {estado === "erro" && erro && (
        <p data-testid="routing-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "sem-rota" && (
        <div data-testid="routing-sem-rota" className="text-xs text-amber-300">
          <p className="font-semibold">Nenhum modelo satisfaz estas restrições.</p>
          <p className="mt-1 text-neutral-300">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto é uma resposta do roteador, não uma avaria: ele recusou em vez de escolher um modelo
            que não cumpre o que foi pedido.
          </p>
        </div>
      )}

      {estado === "pronta" && resposta && (
        <>
          <p data-testid="routing-vencedor" className="mb-2 text-xs text-neutral-200">
            <span className="text-neutral-400">Escolhido:</span>{" "}
            <strong className="font-mono">{resposta.winner.model}</strong>{" "}
            <span className="text-neutral-500">({resposta.winner.provider})</span> — score{" "}
            {resposta.winner.score.toFixed(2)}
          </p>
          <p data-testid="routing-motivo" className="mb-3 text-xs text-neutral-400">
            {resposta.winner.reason}
          </p>

          <table data-testid="routing-candidatos" className="mb-4 w-full text-left text-xs text-neutral-200">
            <thead className="text-neutral-400">
              <tr>
                <th scope="col" className="py-1 pr-2">#</th>
                <th scope="col" className="py-1 pr-2">modelo</th>
                <th scope="col" className="py-1 pr-2">score</th>
                <th scope="col" className="py-1 pr-2">custo</th>
                <th scope="col" className="py-1 pr-2">factores</th>
              </tr>
            </thead>
            <tbody>
              {resposta.candidates.map((candidato) => (
                <tr
                  key={`${candidato.rank}-${candidato.model}`}
                  data-testid={`routing-candidato-${candidato.rank}`}
                  className="border-t border-neutral-800"
                >
                  <td className="py-1 pr-2 text-neutral-500">{candidato.rank}</td>
                  <td className="py-1 pr-2 font-mono text-[11px]">{candidato.model}</td>
                  <td className="py-1 pr-2">{candidato.score.toFixed(2)}</td>
                  <td className="py-1 pr-2">{descreverCusto(candidato.cost_cents_per_1k)}</td>
                  <td className="py-1 pr-2 font-mono text-[10px] text-neutral-500">
                    {descreverFactores(candidato.factors)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {resposta.frontier.length > 0 && (
            <>
              <h4 className="mb-1 text-xs font-semibold text-neutral-300">
                Fronteira de Pareto
              </h4>
              <p className="mb-2 text-[11px] text-neutral-500">
                Alternativas em que nenhuma é melhor em todos os eixos. Não é um ranking — é uma
                escolha.
              </p>
              <table data-testid="routing-fronteira" className="w-full text-left text-xs text-neutral-200">
                <thead className="text-neutral-400">
                  <tr>
                    <th scope="col" className="py-1 pr-2">modelo</th>
                    <th scope="col" className="py-1 pr-2">qualidade</th>
                    <th scope="col" className="py-1 pr-2">custo</th>
                    <th scope="col" className="py-1 pr-2">latência</th>
                    <th scope="col" className="py-1 pr-2">privacidade</th>
                  </tr>
                </thead>
                <tbody>
                  {resposta.frontier.map((alternativa) => (
                    <tr key={alternativa.model} className="border-t border-neutral-800">
                      <td className="py-1 pr-2 font-mono text-[11px]">{alternativa.model}</td>
                      <td className="py-1 pr-2">{alternativa.quality.toFixed(2)}</td>
                      <td className="py-1 pr-2">{alternativa.cost.toFixed(2)}</td>
                      <td className="py-1 pr-2">{alternativa.latency.toFixed(2)}</td>
                      <td className="py-1 pr-2">{alternativa.privacy.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </>
          )}
        </>
      )}

      <p data-testid="routing-fronteira-nota" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        O score e os factores vêm do roteador do servidor; este painel não os recalcula. Um pedido
        sem modelo que o satisfaça devolve recusa com as restrições, e o painel mostra a recusa em
        vez de escolher por conta própria.
      </p>
    </section>
  );
}
