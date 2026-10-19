// Comparacao de solucoes -- o painel que recomenda sem integrar.
//
// NAO faz:
//   - NAO integra nada (e diz isso, em vez de deixar inferir do metodo POST);
//   - NAO trata empate como falha;
//   - NAO esconde os criterios que ninguem mediu nem os medidos parcialmente;
//   - NAO inventa candidatos: recebe-os do utilizador, porque as medicoes sao ENTRADAS.
import { useCallback, useState } from "react";
import {
  compararSolucoes,
  ComparacaoInvalidaError,
  descreverIntegracao,
  descreverRanking,
  descreverTestes,
  descreverVencedor,
  motivoDaRejeicao,
  resumirMedidosParcialmente,
  resumirNaoMedidos,
  type CandidatoDaSolucao,
  type RespostaDaComparacao,
} from "@/lib/solutionsClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "invalida" | "erro";

const EXEMPLO = `[
  {
    "id": "wt-a",
    "branch": "solucao/a",
    "commit": "abc123",
    "agente": "qwen2.5-coder",
    "testes_passaram": 42,
    "testes_falharam": 0,
    "medicoes": [{ "criterio": "diff_linhas", "medido": true, "valor": 120 }]
  },
  {
    "id": "wt-b",
    "branch": "solucao/b",
    "commit": "def456",
    "agente": "llama3.1",
    "testes_passaram": 41,
    "testes_falharam": 1
  }
]`;

export function SolutionsPanel() {
  const [texto, setTexto] = useState(EXEMPLO);
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [resultado, setResultado] = useState<RespostaDaComparacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const comparar = useCallback(async (bruto: string) => {
    setErro(null);
    setEstado("a-carregar");

    let candidatos: CandidatoDaSolucao[];
    try {
      const parsed = JSON.parse(bruto);
      candidatos = Array.isArray(parsed) ? parsed : [];
    } catch (causa) {
      setResultado(null);
      setErro(causa instanceof Error ? causa.message : "JSON invalido.");
      setEstado("invalida");
      return;
    }

    try {
      const r = await compararSolucoes(candidatos);
      setResultado(r);
      setEstado("pronta");
    } catch (causa) {
      setResultado(null);
      if (causa instanceof ComparacaoInvalidaError) {
        setErro(causa.message);
        setEstado("invalida");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel comparar.");
      setEstado("erro");
    }
  }, []);

  const rejeitados = resultado?.rejeitados ?? [];
  const naoMedidos = resultado ? resumirNaoMedidos(resultado) : null;
  const parciais = resultado ? resumirMedidosParcialmente(resultado) : null;

  return (
    <section
      data-testid="solucoes"
      aria-labelledby="solucoes-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-2">
        <h3 id="solucoes-titulo" className="text-sm font-semibold text-neutral-100">
          Comparação de soluções
        </h3>
        <button
          type="button"
          data-testid="solucoes-comparar"
          onClick={() => void comparar(texto)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Comparar
        </button>
      </header>

      <label htmlFor="solucoes-candidatos" className="mb-1 block text-[11px] text-neutral-500">
        Candidatos com as suas medições (as medições são entradas — quem mede é quem chama)
      </label>
      <textarea
        id="solucoes-candidatos"
        data-testid="solucoes-candidatos"
        value={texto}
        onChange={(e) => setTexto(e.target.value)}
        rows={6}
        spellCheck={false}
        className="w-full rounded border border-neutral-700 bg-neutral-950 p-2 font-mono text-[11px] text-neutral-300"
      />

      {estado === "a-carregar" && (
        <p data-testid="solucoes-a-carregar" className="mt-2 text-xs text-neutral-400">
          A comparar...
        </p>
      )}

      {estado === "invalida" && (
        <p data-testid="solucoes-invalida" className="mt-2 text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="solucoes-erro" role="alert" className="mt-2 text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "pronta" && resultado && (
        <>
          {/* O vencedor, dito com o que significa. */}
          <p data-testid="solucoes-vencedor" className="mt-2 text-xs font-semibold text-neutral-200">
            {descreverVencedor(resultado)}
          </p>

          {/* POST, efeito nenhum -- dito pelo servidor. */}
          <p data-testid="solucoes-somente-leitura" className="text-[11px] text-neutral-400">
            {resultado.somente_leitura
              ? "somente leitura: nada foi alterado"
              : "o servidor NÃO declarou somente leitura — tratar como incomum"}
          </p>

          {/* SEMPRE falso. Integrar e' decisao humana separada. */}
          <p data-testid="solucoes-integracao" className="text-[11px] text-emerald-400">
            {descreverIntegracao(resultado)}
          </p>

          {resultado.explicacao && (
            <p className="mt-1 text-[11px] text-neutral-400">{resultado.explicacao}</p>
          )}

          <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">Ranking</h4>
          <p data-testid="solucoes-ranking" className="text-[11px] text-neutral-300">
            {descreverRanking(resultado)}
          </p>

          {/* Criterios sem medicao em ninguem. */}
          {naoMedidos && (
            <p data-testid="solucoes-nao-medidos" className="mt-1 text-[11px] text-amber-300">
              {naoMedidos}
            </p>
          )}

          {/* Criterios medidos em alguns e nao noutros: informam, nao comparam. */}
          {parciais && (
            <p data-testid="solucoes-parciais" className="mt-1 text-[11px] text-amber-300">
              {parciais}
            </p>
          )}

          {naoMedidos === null && parciais === null && (
            <p data-testid="solucoes-tudo-medido" className="mt-1 text-[11px] text-neutral-400">
              Todos os critérios foram medidos em todos os candidatos aceites.
            </p>
          )}

          {rejeitados.length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
                Rejeitados ({rejeitados.length}) — não entram no ranking
              </h4>
              <ul data-testid="solucoes-rejeitados" className="space-y-0.5 text-[11px] text-red-300">
                {rejeitados.map((m, i) => (
                  <li key={`${m.id}-${i}`}>
                    {m.id}: {motivoDaRejeicao(m)}
                  </li>
                ))}
              </ul>
            </>
          )}

          {(resultado.candidatos ?? []).length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">Candidatos</h4>
              <ul data-testid="solucoes-candidatos-lista" className="space-y-0.5 text-[11px]">
                {(resultado.candidatos ?? []).map((c) => (
                  <li key={c.id} data-testid={`solucoes-cand-${c.id}`} className="text-neutral-400">
                    <span className="font-semibold text-neutral-200">{c.id}</span>
                    {c.branch ? ` · ${c.branch}` : ""}
                    {c.agente ? ` · ${c.agente}` : ""} · {descreverTestes(c)}
                  </li>
                ))}
              </ul>
            </>
          )}

          {(resultado.avisos ?? []).length > 0 && (
            <ul data-testid="solucoes-avisos" className="mt-3 space-y-0.5 text-[11px] text-neutral-500">
              {(resultado.avisos ?? []).map((a, i) => (
                <li key={i}>{a}</li>
              ))}
            </ul>
          )}
        </>
      )}

      <p data-testid="solucoes-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Esta rota é POST mas <strong>não altera nada</strong> — o corpo não cabe num URL. Ela
        recomenda, e <strong>nunca integra</strong>: escolher o vencedor e assinar o merge continua a
        ser uma decisão humana.
      </p>
    </section>
  );
}
