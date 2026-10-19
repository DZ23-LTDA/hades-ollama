// Agent Center: catalogo registado, drift e instalacao.
//
// O QUE ESTE PAINEL NAO FAZ
//
//  1. Nao apresenta o ledger como catalogo -- o servidor diz `catalog_available: false` e o painel repete.
//     *Uma lista que so' mostra o que ja' se instalou nao e' um catalogo: e' um historico.*
//  2. Nao recalcula o drift: mostra o estado E a accao que o servidor devolveu.
//  3. Nao reimplementa a fronteira do destino: manda como o utilizador escreveu e deixa o servidor decidir.
import { useCallback, useState } from "react";
import {
  CentroAusenteError,
  coberturaDeAvaliacao,
  contarCasos,
  lerAvaliacao,
  descreverCatalogo,
  listarCatalogo,
  descreverDrift,
  instalarAgente,
  InstalacaoRecusadaError,
  ConflitoDeInstalacaoError,
  lerDrift,
  listarAgentes,
  resumoLegivel,
  validarDestino,
  type Avaliacao,
  type Catalogo,
  type ItemDeDrift,
  type ListaDeAgentes,
  type RelatorioDeDrift,
  type ResultadoDeInstalacao,
} from "@/lib/agentCenterClient";

type Estado =
  | "pronto"
  | "a-carregar"
  | "carregado"
  | "catalogo"
  | "avaliacao"
  | "drift"
  | "instalado"
  | "ausente"
  | "recusado"
  | "conflito"
  | "erro";

const PERSONA_MINIMA = {
  id: "revisor-exemplo",
  name: "Revisor de exemplo",
  version: "1.0.0",
  source: "hades-catalogo",
  license: "MIT",
  description: "Exemplo de persona para provar o registo.",
  prompt: "Leia o diff e aponte o defeito.",
};

export function AgentCenterPanel() {
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [lista, setLista] = useState<ListaDeAgentes | null>(null);
  const [relatorio, setRelatorio] = useState<RelatorioDeDrift | null>(null);
  const [catalogo, setCatalogo] = useState<Catalogo | null>(null);
  const [avaliacao, setAvaliacao] = useState<Avaliacao | null>(null);
  const [instalacao, setInstalacao] = useState<ResultadoDeInstalacao | null>(null);
  const [alvo, setAlvo] = useState("hades");
  const [destino, setDestino] = useState(".hades/agents/revisor-exemplo.md");

  const correr = useCallback(
    async (op: "list" | "catalog" | "evaluation" | "drift" | "install") => {
      setErro(null);
      setEstado("a-carregar");
      try {
        if (op === "list") {
          setLista(await listarAgentes());
          setEstado("carregado");
        } else if (op === "catalog") {
          setCatalogo(await listarCatalogo());
          setEstado("catalogo");
        } else if (op === "evaluation") {
          setAvaliacao(await lerAvaliacao());
          setEstado("avaliacao");
        } else if (op === "drift") {
          setRelatorio(await lerDrift());
          setEstado("drift");
        } else {
          const problema = validarDestino(destino);
          if (problema) {
            setErro(problema);
            setEstado("recusado");
            return;
          }
          setInstalacao(await instalarAgente(PERSONA_MINIMA, alvo, destino));
          setEstado("instalado");
        }
      } catch (causa) {
        if (causa instanceof CentroAusenteError) {
          setErro("O Agent Center não está ligado neste runtime (503): não é «nenhum agente», é «sem armazenamento».");
          setEstado("ausente");
          return;
        }
        if (causa instanceof ConflitoDeInstalacaoError) {
          setErro("Já existe um registo com esta chave e hashes diferentes: resolva o conflito, não reinstale por cima.");
          setEstado("conflito");
          return;
        }
        if (causa instanceof InstalacaoRecusadaError) {
          setErro(causa.message);
          setEstado("recusado");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível correr.");
        setEstado("erro");
      }
    },
    [alvo, destino],
  );

  const itens: ItemDeDrift[] = relatorio?.items ?? [];
  const agentes = lista?.agents ?? [];

  return (
    <section
      data-testid="agent-center"
      aria-labelledby="agent-center-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="agent-center-titulo" className="text-sm font-semibold text-neutral-100">
          Agentes e Equipas
        </h3>
        <span className="text-[11px] text-neutral-500">ledger determinístico · drift por hash</span>
      </header>

      {/* A nota do catalogo vem do SERVIDOR, nao do painel. */}
      <p data-testid="agent-center-catalogo" className="mb-3 text-[11px] leading-relaxed text-amber-300">
        {descreverCatalogo(lista)}
      </p>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="agent-center-listar"
          onClick={() => void correr("list")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Listar agentes
        </button>
        <button
          type="button"
          data-testid="agent-center-catalogo-btn"
          onClick={() => void correr("catalog")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Ver catálogo
        </button>
        <button
          type="button"
          data-testid="agent-center-avaliacao-btn"
          onClick={() => void correr("evaluation")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Ver critérios de avaliação
        </button>
        <button
          type="button"
          data-testid="agent-center-drift"
          onClick={() => void correr("drift")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Verificar integridade
        </button>
        <select
          data-testid="agent-center-alvo"
          value={alvo}
          onChange={(e) => setAlvo(e.target.value)}
          aria-label="Formato de destino"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        >
          <option value="hades">hades</option>
          <option value="codex">codex</option>
          <option value="claude-code">claude-code</option>
        </select>
        <input
          data-testid="agent-center-destino"
          value={destino}
          onChange={(e) => setDestino(e.target.value)}
          placeholder="destino relativo"
          aria-label="Destino relativo"
          className="w-64 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="agent-center-instalar"
          onClick={() => void correr("install")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Registar instalação
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="agent-center-pronto" className="text-xs text-neutral-400">
          Nada consultado nesta sessão.
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="agent-center-a-carregar" className="text-xs text-neutral-400">
          A consultar...
        </p>
      )}

      {estado === "carregado" && (
        <div className="space-y-1">
          <p data-testid="agent-center-contagem" className="text-xs font-semibold text-emerald-300">
            {agentes.length} instalação(ões) registada(s)
          </p>
          <ul data-testid="agent-center-lista" className="font-mono text-[11px] text-neutral-300">
            {agentes.map((a) => (
              <li key={`${a.persona_id}-${a.target}-${a.destination}`}>
                {a.persona_id} v{a.version} · {a.target} · {a.destination}
              </li>
            ))}
          </ul>
        </div>
      )}

      {estado === "catalogo" && (
        <div className="space-y-1">
          <p data-testid="agent-center-cat-versao" className="text-[11px] text-neutral-300">
            versão {catalogo?.version ?? "?"} · {coberturaDeAvaliacao(catalogo).total} persona(s) ·{" "}
            {coberturaDeAvaliacao(catalogo).avaliadas} avaliada(s)
          </p>
          {/* *Um numero sem a sua qualidade ao lado e' um numero que se cita mal.* */}
          {coberturaDeAvaliacao(catalogo).avaliadas === 0 && (
            <p data-testid="agent-center-cat-sem-avaliacao" className="text-[11px] text-amber-300">
              Nenhuma persona do catálogo foi avaliada. Isto não é «aprovado»: é «não medido».
            </p>
          )}
          <ul data-testid="agent-center-cat-lista" className="space-y-1 font-mono text-[11px] text-neutral-300">
            {(catalogo?.personas ?? []).map((p) => (
              <li key={p.id}>
                {p.id} v{p.version} · {p.division ?? "?"} · {p.license ?? "?"} ·{" "}
                {(p.evaluation ?? "").trim() === "" ? "NÃO AVALIADO" : p.evaluation}
              </li>
            ))}
          </ul>
        </div>
      )}

      {estado === "avaliacao" && (
        <div className="space-y-1">
          <p data-testid="agent-center-aval-nota" className="text-[11px] text-amber-300">
            {avaliacao?.note ?? "sem nota do servidor"}
          </p>
          <p data-testid="agent-center-aval-inferencia" className="text-[11px] text-neutral-400">
            {avaliacao?.inference_available === true
              ? "execução ligada a um motor"
              : "execução contra modelo NÃO ligada neste runtime"}
          </p>
          <ul data-testid="agent-center-aval-suits" className="space-y-1 font-mono text-[11px] text-neutral-300">
            {(avaliacao?.divisions ?? []).map((d) => {
              const s = avaliacao?.suites?.[d];
              const n = contarCasos(s);
              return (
                <li key={d}>
                  {d} · v{s?.version ?? "?"} · {n.total} caso(s) · {n.negativos} de recusa/injeção
                </li>
              );
            })}
          </ul>
          <p data-testid="agent-center-aval-estados" className="text-[11px] text-neutral-500">
            {(avaliacao?.states ?? []).length} estados declarados ·{" "}
            {(avaliacao?.states ?? []).filter((s) => s.terminal).length} terminal(is)
          </p>
        </div>
      )}

      {estado === "drift" && (
        <div className="space-y-1">
          <p data-testid="agent-center-resumo" className="text-[11px] text-neutral-300">
            {resumoLegivel(relatorio?.summary)}
          </p>
          <ul data-testid="agent-center-itens" className="space-y-1 text-[11px] text-neutral-300">
            {itens.map((item) => (
              <li key={`${item.entry.persona_id}-${item.entry.destination}`}>
                {item.entry.persona_id} · {descreverDrift(item)}
              </li>
            ))}
          </ul>
        </div>
      )}

      {estado === "instalado" && (
        <p data-testid="agent-center-instalado" className="text-xs font-semibold text-emerald-300">
          {instalacao?.created === true ? "201 criado" : "200 já registado"} ·{" "}
          {instalacao?.entry?.persona_id ?? "sem id"} · {instalacao?.bytes ?? 0} bytes renderizados
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="agent-center-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "conflito" && (
        <p data-testid="agent-center-conflito" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="agent-center-recusado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="agent-center-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="agent-center-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Registar uma instalação não escreve no destino: calcula os hashes e grava no ledger. O destino tem de
        ser relativo e o servidor recusa `..` — um destino que o cliente escreve é um destino que o cliente
        escolhe.
      </p>
    </section>
  );
}
