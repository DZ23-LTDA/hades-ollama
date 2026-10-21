// Painel de retenção de memória -- a superfície que faltava para `/retention`.
//
// O QUE ESTE PAINEL NÃO FAZ, DE PROPÓSITO:
//   - não calcula a política sozinho: mostra o que o servidor devolveu;
//   - não trata "não configurada" como erro: é um estado legítimo;
//   - não apaga nada sem um clique explícito em "Aplicar agora", e diz os
//     números reais do que removeu em vez de "sucesso";
//   - não esconde o 400 do servidor: mostra a mensagem.
//
//	*Um painel que confia cegamente no servidor não é um painel: é um espelho.
import { useCallback, useEffect, useState } from "react";
import {
  aplicarPoliticaDeRetencao,
  definirPoliticaDeRetencao,
  descreverPolitica,
  descreverResultadoDeAplicacao,
  IDADE_MAXIMA_DIAS,
  IDADE_MINIMA_DIAS,
  MEMORIAS_MAXIMAS_POR_PROJECTO,
  obterPoliticaDeRetencao,
  validarPolitica,
  type PoliticaDeRetencao,
  type ResultadoDeAplicacao,
} from "@/lib/retentionClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "por-configurar" | "erro";

export function RetentionPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [politica, setPolitica] = useState<PoliticaDeRetencao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const [idade, setIdade] = useState(90);
  const [memorias, setMemorias] = useState(0);
  const [aGravar, setAGravar] = useState(false);
  const [avisoDeGravacao, setAvisoDeGravacao] = useState<string | null>(null);

  const [aAplicar, setAAplicar] = useState(false);
  const [resultado, setResultado] = useState<ResultadoDeAplicacao | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    try {
      const resposta = await obterPoliticaDeRetencao();
      if (resposta.configured && resposta.policy) {
        setPolitica(resposta.policy);
        setIdade(resposta.policy.max_age_days);
        setMemorias(resposta.policy.max_memories_per_project);
        setEstado("pronta");
      } else {
        setPolitica(null);
        setEstado("por-configurar");
      }
    } catch (causa) {
      setErro(causa instanceof Error ? causa.message : "Não foi possível ler a política.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const gravar = useCallback(async () => {
    const problema = validarPolitica(idade, memorias);
    if (problema) {
      setErro(problema);
      setAvisoDeGravacao(null);
      return;
    }
    setErro(null);
    setAGravar(true);
    try {
      const gravada = await definirPoliticaDeRetencao(idade, memorias);
      setPolitica(gravada);
      setEstado("pronta");
      setAvisoDeGravacao("Política gravada. Nada foi removido ainda.");
    } catch (causa) {
      setErro(causa instanceof Error ? causa.message : "Não foi possível gravar a política.");
      setAvisoDeGravacao(null);
    } finally {
      setAGravar(false);
    }
  }, [idade, memorias]);

  const aplicar = useCallback(async () => {
    setErro(null);
    setAAplicar(true);
    try {
      const aplicado = await aplicarPoliticaDeRetencao();
      setResultado(aplicado);
    } catch (causa) {
      setErro(causa instanceof Error ? causa.message : "Não foi possível aplicar a política.");
      setResultado(null);
    } finally {
      setAAplicar(false);
    }
  }, []);

  return (
    <section
      data-testid="retention"
      aria-labelledby="retention-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="retention-titulo" className="text-sm font-semibold text-neutral-100">
          Retenção de memória
        </h3>
        <button
          type="button"
          data-testid="retention-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "a-carregar" && (
        <p data-testid="retention-a-carregar" className="text-xs text-neutral-400">
          A ler a política...
        </p>
      )}

      {estado === "erro" && erro && (
        <p data-testid="retention-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "por-configurar" && (
        <p data-testid="retention-por-configurar" className="mb-3 text-xs text-amber-300">
          Esta organização ainda não tem política de retenção declarada. Sem política, nada é removido
          automaticamente — a memória fica guardada até alguém a apagar.
        </p>
      )}

      {(estado === "pronta" || estado === "por-configurar") && (
        <>
          {politica && (
            <p data-testid="retention-resumo" className="mb-3 text-xs text-neutral-300">
              {descreverPolitica(politica)}
              {politica.updated_at && (
                <span className="ml-2 text-neutral-500">
                  Atualizada em {new Date(politica.updated_at).toLocaleString("pt-BR")}.
                </span>
              )}
            </p>
          )}

          <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-xs text-neutral-300">
              Idade máxima (dias)
              <input
                type="number"
                data-testid="retention-idade"
                min={IDADE_MINIMA_DIAS}
                max={IDADE_MAXIMA_DIAS}
                value={idade}
                onChange={(evento) => setIdade(Number(evento.target.value))}
                className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
              />
              <span className="text-neutral-500">
                {IDADE_MINIMA_DIAS} a {IDADE_MAXIMA_DIAS} dias.
              </span>
            </label>

            <label className="flex flex-col gap-1 text-xs text-neutral-300">
              Tecto por projecto
              <input
                type="number"
                data-testid="retention-memorias"
                min={0}
                max={MEMORIAS_MAXIMAS_POR_PROJECTO}
                value={memorias}
                onChange={(evento) => setMemorias(Number(evento.target.value))}
                className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
              />
              <span className="text-neutral-500">
                0 significa <strong>sem tecto</strong> de contagem.
              </span>
            </label>
          </div>

          <div className="mb-3 flex flex-wrap gap-2">
            <button
              type="button"
              data-testid="retention-gravar"
              onClick={() => void gravar()}
              disabled={aGravar}
              className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800 disabled:opacity-50"
            >
              {aGravar ? "A gravar..." : "Gravar política"}
            </button>
            <button
              type="button"
              data-testid="retention-aplicar"
              onClick={() => void aplicar()}
              disabled={aAplicar || !politica}
              title={!politica ? "Grave uma política primeiro." : undefined}
              className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800 disabled:opacity-50"
            >
              {aAplicar ? "A aplicar..." : "Aplicar agora"}
            </button>
          </div>

          <div aria-live="polite" className="min-h-[1rem]">
            {avisoDeGravacao && (
              <p data-testid="retention-aviso-gravacao" className="text-xs text-emerald-400">
                {avisoDeGravacao}
              </p>
            )}
            {erro && (
              <p data-testid="retention-erro" role="alert" className="text-xs text-red-400">
                {erro}
              </p>
            )}
            {resultado && (
              <p data-testid="retention-resultado" className="text-xs text-neutral-200">
                {descreverResultadoDeAplicacao(resultado)}
              </p>
            )}
          </div>

          <p data-testid="retention-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
            A aplicação remove memórias de projetos desta organização e é idempotente: aplicar duas
            vezes não remove duas vezes. A política vive no servidor — este painel só a declara e
            pede a aplicação.
          </p>
        </>
      )}
    </section>
  );
}
