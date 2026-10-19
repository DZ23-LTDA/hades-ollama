// Aprovadores de aprovação: mover a lista, e dizer o que ela significa.
//
// Painel de ESCRITA num ato de POLÍTICA: a lista vazia não é «sem aprovadores» — é «sem checagem de
// papel». Um painel que mostre as duas iguais ensina o utilizador a confundir um portão aberto com uma
// lista por preencher.
import { useCallback, useState } from "react";
import {
  AprovadoresProibidosError,
  AprovadoresRecusadosError,
  definirAprovadores,
  descreverDuasIdentidades,
  descreverPortao,
  descreverResultado,
  validarAtor,
  type ResultadoDeAprovadores,
} from "@/lib/approversClient";

type Estado = "pronto" | "a-gravar" | "gravado" | "proibido" | "recusado" | "erro";

export function ApproversPanel() {
  const [ator, setAtor] = useState("");
  const [lista, setLista] = useState("");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [resultado, setResultado] = useState<ResultadoDeAprovadores | null>(null);

  const gravar = useCallback(async () => {
    const problema = validarAtor(ator);
    if (problema) {
      setErro(problema);
      setEstado("recusado");
      return;
    }

    setErro(null);
    setEstado("a-gravar");
    const approvers = lista
      .split(/[\n,]/)
      .map((a) => a.trim())
      .filter((a) => a !== "");

    try {
      const r = await definirAprovadores({ actorId: ator, approvers });
      setResultado(r);
      setEstado("gravado");
    } catch (causa) {
      setResultado(null);
      if (causa instanceof AprovadoresProibidosError) {
        setErro(causa.message);
        setEstado("proibido");
        return;
      }
      if (causa instanceof AprovadoresRecusadosError) {
        setErro(causa.message);
        setEstado("recusado");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível gravar.");
      setEstado("erro");
    }
  }, [ator, lista]);

  const portao = resultado?.role_checked;
  const vazio = (resultado?.approvers ?? []).length === 0;

  return (
    <section
      data-testid="aprovadores"
      aria-labelledby="aprovadores-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="aprovadores-titulo" className="text-sm font-semibold text-neutral-100">
          Aprovadores de aprovação
        </h3>
        <span className="text-[11px] text-neutral-500">ato de política · exige owner ou admin</span>
      </header>

      <div className="mb-3 flex flex-wrap items-start gap-2">
        <input
          data-testid="aprovadores-ator"
          value={ator}
          onChange={(e) => setAtor(e.target.value)}
          placeholder="actor_id"
          aria-label="Actor que declara a mudança"
          className="w-40 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <textarea
          data-testid="aprovadores-lista"
          value={lista}
          onChange={(e) => setLista(e.target.value)}
          placeholder="um id por linha (vazio = sem portão de papel)"
          aria-label="Ids dos aprovadores"
          rows={2}
          className="w-64 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="aprovadores-gravar"
          onClick={() => void gravar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Gravar lista
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="aprovadores-pronto" className="text-xs text-neutral-400">
          A lista actual não foi lida nesta sessão. Gravar substitui-a.
        </p>
      )}

      {estado === "a-gravar" && (
        <p data-testid="aprovadores-a-gravar" className="text-xs text-neutral-400">
          A gravar...
        </p>
      )}

      {estado === "gravado" && (
        <div className="space-y-1">
          <p data-testid="aprovadores-gravado" className="text-xs font-semibold text-emerald-300">
            200 · {descreverResultado(resultado)}
          </p>
          {/* Lista vazia: o portao abriu. Nao e' um estado neutro. */}
          {vazio && portao === false && (
            <p data-testid="aprovadores-portao-aberto" className="text-xs text-amber-300">
              {descreverPortao(resultado?.approvers)}
            </p>
          )}
          {/* A trilha pode falhar e o servidor declara isso. Nao escondemos. */}
          {resultado?.audit_recorded === false && (
            <p data-testid="aprovadores-trilha-falhou" className="text-xs text-red-400">
              A mudança foi gravada mas a TRILHA NÃO REGISTOU
              {resultado.audit_reason ? `: ${resultado.audit_reason}` : ""}. Um ato de política sem
              registo é invisível para quem audita.
            </p>
          )}
          {portao === true && (
            <p data-testid="aprovadores-portao" className="text-xs text-neutral-400">
              {descreverPortao(resultado?.approvers)}
            </p>
          )}
        </div>
      )}

      {/* 403: o portao de papel a funcionar, nao uma falha do pedido. */}
      {estado === "proibido" && (
        <p data-testid="aprovadores-proibido" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="aprovadores-recusado" className="text-xs text-amber-300">
          O servidor recusou: {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="aprovadores-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="aprovadores-duas-identidades" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverDuasIdentidades()}
      </p>
    </section>
  );
}
