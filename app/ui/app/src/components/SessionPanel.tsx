// Painel de sessão -- quem está autenticado, em que organização e com que MFA.
//
// O QUE ESTE PAINEL NAO FAZ, DE PROPOSITO:
//   - NAO mostra segredo de MFA nem codigo de recuperacao: a API nao os devolve
//     (`User.Public()` limpa os dois campos, e `omitempty` faz o resto).
//   - NAO le `mfa_enabled` ausente como «desconhecido»: ausente e' `false`.
//   - NAO trata `mode: "local"` como sessao invalida: e' o modo local.
//   - NAO faz logout, MFA, OAuth nem SAML: 12 rotas de escrita ficam de fora.
import { useCallback, useEffect, useState } from "react";
import {
  descreverMFA,
  descreverModo,
  descreverPapel,
  mfaLigado,
  nomeDoUtilizador,
  obterSessao,
  type SessaoDeAutenticacao,
} from "@/lib/authClient";

type EstadoDoPainel = "a-carregar" | "pronta" | "erro";

export function SessionPanel() {
  const [estado, setEstado] = useState<EstadoDoPainel>("a-carregar");
  const [sessao, setSessao] = useState<SessaoDeAutenticacao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setErro(null);
    setEstado("a-carregar");
    try {
      const s = await obterSessao();
      setSessao(s);
      setEstado("pronta");
    } catch (causa) {
      setSessao(null);
      setErro(causa instanceof Error ? causa.message : "Nao foi possivel ler a sessao.");
      setEstado("erro");
    }
  }, []);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  const utilizador = sessao?.user;
  const membro = sessao?.membership;

  return (
    <section
      data-testid="sessao"
      aria-labelledby="sessao-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex items-center justify-between gap-3">
        <h3 id="sessao-titulo" className="text-sm font-semibold text-neutral-100">
          Sessão
        </h3>
        <button
          type="button"
          data-testid="sessao-actualizar"
          onClick={() => void carregar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Atualizar
        </button>
      </header>

      {estado === "erro" && erro && (
        <p data-testid="sessao-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "a-carregar" && (
        <p data-testid="sessao-a-carregar" className="text-xs text-neutral-400">
          A ler a sessão...
        </p>
      )}

      {estado === "pronta" && sessao && (
        <>
          <p
            data-testid="sessao-modo"
            className={
              sessao.mode === "local"
                ? "text-[11px] text-neutral-300"
                : "text-[11px] text-emerald-300"
            }
          >
            {descreverModo(sessao)}
          </p>

          {sessao.authenticated ? (
            <>
              <p data-testid="sessao-utilizador" className="mt-2 text-xs font-semibold text-neutral-100">
                {nomeDoUtilizador(utilizador)}
              </p>

              {utilizador?.email && (
                <p data-testid="sessao-email" className="text-[11px] text-neutral-400">
                  {utilizador.email}
                </p>
              )}

              <p
                data-testid="sessao-mfa"
                className={
                  mfaLigado(utilizador)
                    ? "mt-1 text-[11px] text-emerald-400"
                    : "mt-1 text-[11px] text-amber-400"
                }
              >
                {descreverMFA(utilizador)}
              </p>

              <p data-testid="sessao-papel" className="mt-1 text-[11px] text-neutral-400">
                {descreverPapel(membro)}
              </p>

              {utilizador?.created_at && (
                <p data-testid="sessao-criado" className="mt-1 text-[11px] text-neutral-500">
                  conta criada em {utilizador.created_at}
                </p>
              )}
            </>
          ) : (
            <p data-testid="sessao-sem-utilizador" className="mt-2 text-xs text-neutral-400">
              Não há utilizador nesta sessão. Isto não é um erro: é o HADES a correr sem
              autenticação exigida.
            </p>
          )}
        </>
      )}

      <p data-testid="sessao-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        A API devolve a versão pública do utilizador: o segredo de MFA e os códigos de
        recuperação são removidos antes da resposta. Este painel só lê — entrar, sair, ligar
        MFA, OAuth e SAML continuam a ser ações separadas.
      </p>
    </section>
  );
}
