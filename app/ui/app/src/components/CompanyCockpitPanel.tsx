// Cockpit da empresa: lançar e certificar. Dois atos do mesmo ciclo de vida.
//
// O painel NÃO diz «empresa criada»: diz o que o servidor entregou — artefatos e checklist. A fábrica não
// cria cliente nem emite registro legal, e o painel repete as palavras do servidor, não as do marketing.
import { useCallback, useState } from "react";
import {
  certificarJornada,
  descreverArtefactos,
  descreverFronteiraDaFabrica,
  descreverIsolamento,
  EmpresaAusenteError,
  FabricaRecusadaError,
  JornadaQuebradaError,
  lancarEmpresa,
  resumirJornada,
  validarEmpresa,
  type RelatorioDaFabrica,
  type RelatorioDeJornada,
} from "@/lib/companyCockpitClient";

type Estado = "pronto" | "a-correr" | "lancada" | "certificada" | "quebrada" | "ausente" | "recusada" | "erro";

export function CompanyCockpitPanel() {
  const [nome, setNome] = useState("");
  const [moeda, setMoeda] = useState("BRL");
  const [tagline, setTagline] = useState("");
  const [outraOrg, setOutraOrg] = useState("");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [fabrica, setFabrica] = useState<RelatorioDaFabrica | null>(null);
  const [jornada, setJornada] = useState<RelatorioDeJornada | null>(null);
  const [op, setOp] = useState<"launch" | "certify">("launch");

  const correr = useCallback(
    async (operacao: "launch" | "certify") => {
      setOp(operacao);

      if (operacao === "launch") {
        const problema = validarEmpresa(nome, moeda);
        if (problema) {
          setErro(problema);
          setEstado("recusada");
          return;
        }
      }

      setErro(null);
      setEstado("a-correr");
      try {
        if (operacao === "launch") {
          const r = await lancarEmpresa({ name: nome, currency: moeda, tagline });
          setFabrica(r);
          setJornada(null);
          setEstado("lancada");
        } else {
          const r = await certificarJornada({ otherOrganizationId: outraOrg });
          setJornada(r);
          setFabrica(null);
          setEstado("certificada");
        }
      } catch (causa) {
        if (causa instanceof EmpresaAusenteError) {
          setErro(causa.message);
          setEstado("ausente");
          return;
        }
        // 422: a jornada quebrou E o relatório veio junto. Mostramos os dois.
        if (causa instanceof JornadaQuebradaError) {
          setJornada(causa.relatorio);
          setErro(causa.message);
          setEstado("quebrada");
          return;
        }
        if (causa instanceof FabricaRecusadaError) {
          setErro(causa.message);
          setEstado("recusada");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível correr.");
        setEstado("erro");
      }
    },
    [nome, moeda, tagline, outraOrg],
  );

  return (
    <section
      data-testid="empresa-cockpit"
      aria-labelledby="empresa-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <h3 id="empresa-titulo" className="mb-3 text-sm font-semibold text-neutral-100">
        Empresa: lançar e certificar
      </h3>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="empresa-nome"
          value={nome}
          onChange={(e) => setNome(e.target.value)}
          placeholder="nome da empresa"
          aria-label="Nome da empresa"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="empresa-moeda"
          value={moeda}
          onChange={(e) => setMoeda(e.target.value)}
          placeholder="moeda"
          aria-label="Moeda"
          className="w-20 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="empresa-tagline"
          value={tagline}
          onChange={(e) => setTagline(e.target.value)}
          placeholder="tagline (opcional)"
          aria-label="Tagline"
          className="w-44 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="empresa-outra-org"
          value={outraOrg}
          onChange={(e) => setOutraOrg(e.target.value)}
          placeholder="2.ª organização (opcional)"
          aria-label="Segunda organização para a prova de isolamento"
          className="w-48 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="empresa-lancar"
          onClick={() => void correr("launch")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Lançar empresa
        </button>
        <button
          type="button"
          data-testid="empresa-certificar"
          onClick={() => void correr("certify")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Certificar jornada
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="empresa-pronto" className="text-xs text-neutral-400">
          Nada executado nesta sessão. Lançar cria artefactos; certificar corre a jornada inteira.
        </p>
      )}

      {estado === "a-correr" && (
        <p data-testid="empresa-a-correr" className="text-xs text-neutral-400">
          A correr...
        </p>
      )}

      {estado === "lancada" && (
        <div className="space-y-1">
          <p data-testid="empresa-lancada" className="text-xs font-semibold text-emerald-300">
            201 criado · {op === "launch" ? "lançamento" : "certificação"} · {descreverArtefactos(fabrica)}
          </p>
          {/* A fronteira: a fabrica NAO abre a empresa. */}
          <p data-testid="empresa-fronteira" className="text-[11px] text-neutral-500">
            {descreverFronteiraDaFabrica()}
          </p>
        </div>
      )}

      {estado === "certificada" && (
        <div className="space-y-1">
          <p data-testid="empresa-certificada" className="text-xs font-semibold text-emerald-300">
            200 · {resumirJornada(jornada)}
          </p>
          <p data-testid="empresa-isolamento" className="text-[11px] text-neutral-500">
            {descreverIsolamento()}
          </p>
        </div>
      )}

      {/* 422: a jornada quebrou E o relatorio veio junto. Os dois sao mostrados. */}
      {estado === "quebrada" && (
        <div className="space-y-1">
          <p data-testid="empresa-quebrada" className="text-xs font-semibold text-amber-300">
            422 · a jornada quebrou · {resumirJornada(jornada)}
          </p>
          <p data-testid="empresa-quebrada-relatorio" className="text-[11px] text-neutral-400">
            O relatório veio COM a falha: é ele que diz qual etapa quebrou. Um relatório que só aparece
            quando tudo corre bem é um prémio, não um diagnóstico.
          </p>
          {erro && <p className="text-[11px] text-neutral-500">{erro}</p>}
        </div>
      )}

      {estado === "ausente" && (
        <p data-testid="empresa-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusada" && (
        <p data-testid="empresa-recusada" className="text-xs text-amber-300">
          O servidor recusou: {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="empresa-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="empresa-fronteira-estatica" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverFronteiraDaFabrica()}
      </p>
    </section>
  );
}
