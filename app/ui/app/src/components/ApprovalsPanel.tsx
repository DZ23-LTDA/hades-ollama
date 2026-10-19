// Painel de Aprovacoes Humanas -- a fila que a FASE 4 exige.
//
// O QUE ESTE PAINEL NAO FAZ, DE PROPOSITO:
//   - NAO esconde `role_checked=false`: diz «sem lista de aprovadores».
//   - NAO mostra quorum como binario: diz «2 de 3».
//   - NAO trata «pendente» e «esquecido» como o mesmo: `overdue` e' mostrado em separado.
//   - NAO esconde que o canal de aviso pode ter falhado.
//   - NAO mostra o fingerprint inteiro: identifica, nao expoe.
//   - NAO decide aprovacao: so' le.
import { useCallback, useEffect, useState } from "react";
import {
  AprovacoesAusentesError,
  descreverEstadoDaAprovacao,
  descreverIdade,
  descreverNotificacao,
  descreverQuorum,
  descreverVerificacaoDePapel,
  estadoDaAuditoria,
  listarAprovacoes,
  truncarFingerprint,
  type AprovacaoDeServico,
  type EstadoDaAuditoria,
  type RespostaDeAprovacoes,
} from "@/lib/approvalsClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "sem-aprovacoes" | "erro";
const ESTADOS = ["", "pending", "approved", "denied", "consumed", "expired"];

export function ApprovalsPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [resposta, setResposta] = useState<RespostaDeAprovacoes | null>(null);
  const [auditoria, setAuditoria] = useState<EstadoDaAuditoria | null>(null);
  const [auditoriaErro, setAuditoriaErro] = useState<string | null>(null);
  const [filtro, setFiltro] = useState("pending");
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async (estadoFiltro: string) => {
    setErro(null);
    setEstado("a-carregar");
    setAuditoriaErro(null);
    try {
      const r = await listarAprovacoes(estadoFiltro);
      setResposta(r);
      setEstado("pronta");
    } catch (causa) {
      setResposta(null);
      if (causa instanceof AprovacoesAusentesError) {
        setErro(causa.message);
        setEstado("sem-aprovacoes");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler as aprovacoes.");
      setEstado("erro");
    }
    // A auditoria e' um pedido separado: a sua falha nao apaga a fila.
    try {
      setAuditoria(await estadoDaAuditoria());
    } catch (causa) {
      setAuditoria(null);
      setAuditoriaErro(
        causa instanceof Error ? causa.message : "Nao foi possivel ler a auditoria.",
      );
    }
  }, []);

  useEffect(() => {
    void carregar("pending");
  }, [carregar]);

  const fila = resposta?.queue;
  const notificacao = resposta?.notification;
  const aprovacoes: AprovacaoDeServico[] = resposta?.approvals ?? [];

  return (
    <section
      data-testid="aprovacoes"
      aria-labelledby="aprovacoes-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="aprovacoes-titulo" className="text-sm font-semibold text-neutral-100">
          Aprovações humanas
        </h3>
        <button
          type="button"
          data-testid="aprovacoes-actualizar"
          onClick={() => void carregar(filtro)}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      <div className="mb-3 flex flex-wrap items-end gap-2">
        <label className="flex flex-col gap-1 text-xs text-neutral-300">
          Estado
          <select
            data-testid="aprovacoes-filtro"
            value={filtro}
            onChange={(evento) => setFiltro(evento.target.value)}
            className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-neutral-100"
          >
            {ESTADOS.map((valor) => (
              <option key={valor} value={valor}>
                {valor === "" ? "todos" : descreverEstadoDaAprovacao(valor)}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          data-testid="aprovacoes-filtrar"
          onClick={() => void carregar(filtro)}
          className="rounded border border-neutral-700 px-3 py-1 text-xs text-neutral-100 hover:bg-neutral-800"
        >
          Filtrar
        </button>
      </div>

      {estado === "sem-aprovacoes" && (
        <div data-testid="aprovacoes-sem-store" className="text-xs text-amber-300">
          <p className="font-semibold">O store de aprovação humana não está configurado.</p>
          <p className="mt-1 text-neutral-400">{erro}</p>
          <p className="mt-1 text-neutral-500">
            Isto não é «não há aprovações pendentes»: é «não há aprovações». Uma fila vazia aqui
            diria que ninguém está à espera — e ninguém saberia que ninguém está à espera.
          </p>
        </div>
      )}

      {estado === "erro" && erro && (
        <p data-testid="aprovacoes-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="aprovacoes-a-carregar" className="text-xs text-neutral-400">
          A ler a fila...
        </p>
      )}

      {estado === "pronta" && fila && (
        <>
          <p data-testid="aprovacoes-fila" className="mb-1 text-xs text-neutral-300">
            {fila.pending} pendente(s) ·{" "}
            <span className={fila.overdue > 0 ? "text-red-400" : "text-emerald-400"}>
              {fila.overdue} em atraso
            </span>{" "}
            · {fila.expired} expirada(s) · mais antiga há {descreverIdade(fila.oldest_seconds)} · TTL{" "}
            {descreverIdade(fila.pending_ttl_seconds)}
          </p>

          <p
            data-testid="aprovacoes-notificacao"
            className={
              notificacao && !notificacao.configurado
                ? "mb-3 text-[11px] text-amber-400"
                : notificacao && notificacao.ultimo_erro !== ""
                  ? "mb-3 text-[11px] text-red-400"
                  : "mb-3 text-[11px] text-neutral-500"
            }
          >
            {descreverNotificacao(notificacao)}
          </p>

          {aprovacoes.length === 0 ? (
            <p data-testid="aprovacoes-lista-vazia" className="text-xs text-neutral-400">
              {filtro !== ""
                ? `Nenhuma aprovação com estado «${descreverEstadoDaAprovacao(filtro)}».`
                : "O store está ligado e não há aprovações nesta organização."}
            </p>
          ) : (
            <ul data-testid="aprovacoes-lista" className="space-y-1">
              {aprovacoes.map((aprovacao) => (
                <li
                  key={aprovacao.id}
                  data-testid={`aprovacao-${aprovacao.id}`}
                  className="rounded border border-neutral-800 px-2 py-1 text-xs text-neutral-200"
                >
                  <span className="font-semibold">{aprovacao.tool_name}</span>{" "}
                  <span className="text-neutral-400">
                    [{descreverEstadoDaAprovacao(aprovacao.status)}] ·{" "}
                    {descreverQuorum(
                      aprovacao.decisions?.length ?? 0,
                      aprovacao.required_approvals,
                    )}{" "}
                    · {truncarFingerprint(aprovacao.fingerprint)}
                  </span>
                  {/* `role_checked=false` e' declarado, nao escondido. */}
                  <span
                    data-testid={`aprovacao-papel-${aprovacao.id}`}
                    className={aprovacao.role_checked ? "block text-[11px] text-emerald-400" : "block text-[11px] text-amber-400"}
                  >
                    {descreverVerificacaoDePapel(aprovacao.role_checked)}
                  </span>
                </li>
              ))}
            </ul>
          )}

          {fila.items && fila.items.length > 0 && (
            <>
              <h4 className="mb-1 mt-3 text-xs font-semibold text-neutral-300">
                O que está à espera há mais tempo
              </h4>
              <ul data-testid="aprovacoes-fila-itens" className="space-y-0.5 text-[11px] text-neutral-300">
                {fila.items.map((item) => (
                  <li
                    key={item.id}
                    data-testid={`aprovacao-item-${item.id}`}
                    className={item.overdue ? "text-red-400" : "text-neutral-300"}
                  >
                    {item.tool_name} · {descreverQuorum(item.approvals_given, item.approvals_needed)}{" "}
                    · há {descreverIdade(item.age_seconds)}
                    {item.overdue && " · PASSOU DO LIMIAR"}
                  </li>
                ))}
              </ul>
            </>
          )}
        </>
      )}

      {auditoriaErro && (
        <p data-testid="aprovacoes-auditoria-erro" className="mt-2 text-[11px] text-amber-400">
          Trilha de auditoria: {auditoriaErro}
        </p>
      )}

      {auditoria && (
        <p
          data-testid="aprovacoes-auditoria"
          className={auditoria.persisted ? "mt-2 text-[11px] text-emerald-400" : "mt-2 text-[11px] text-amber-400"}
        >
          {auditoria.persisted
            ? "trilha de auditoria durável activa"
            : "sem trilha de auditoria durável — nada é guardado"}
        </p>
      )}

      <p data-testid="aprovacoes-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        Este painel só lê. Aprovar, negar e definir aprovadores continuam a ser ações separadas — e
        uma aprovação sem lista de aprovadores não é a mesma coisa que uma aprovação verificada.
      </p>
    </section>
  );
}
