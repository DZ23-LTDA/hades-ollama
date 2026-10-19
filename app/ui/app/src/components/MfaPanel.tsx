// MFA: ligar, desligar e gerar códigos de recuperação.
//
// O painel NÃO pede senha para fingir uma reautenticação que o servidor não faz. Pede uma CONFIRMAÇÃO
// (trava de clique) e DECLARA que é só isso. A diferença entre as duas coisas é a diferença entre um
// painel e um selo.
import { useCallback, useState } from "react";
import {
  descreverAchadoDeReautenticacao,
  descreverCodigos,
  descreverEstado,
  descreverLimiteDaConfirmacao,
  desligarMfa,
  gerarCodigosDeRecuperacao,
  ligarMfa,
  MfaRecusadoError,
  MfaSemSessaoError,
  validarSegredo,
  type ResultadoDeMfa,
  type ResultadoDeRecuperacao,
} from "@/lib/authMfaClient";

type Estado = "pronto" | "a-correr" | "ligado" | "desligado" | "codigos" | "sem-sessao" | "recusado" | "erro";

export function MfaPanel() {
  const [segredo, setSegredo] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [mfa, setMfa] = useState<ResultadoDeMfa | null>(null);
  const [recuperacao, setRecuperacao] = useState<ResultadoDeRecuperacao | null>(null);

  const correr = useCallback(
    async (op: "enable" | "disable" | "codes") => {
      if (op !== "enable" && !confirmado) {
        setErro(
          "Confirme antes: esta operação mexe no segundo factor. (E note: o servidor não reautentica — a confirmação é uma trava de clique.)",
        );
        setEstado("recusado");
        return;
      }
      if (op === "enable") {
        const problema = validarSegredo(segredo);
        if (problema) {
          setErro(problema);
          setEstado("recusado");
          return;
        }
      }

      setErro(null);
      setEstado("a-correr");
      try {
        if (op === "enable") {
          const r = await ligarMfa(segredo);
          setMfa(r);
          setRecuperacao(null);
          setEstado("ligado");
          setSegredo("");
        } else if (op === "disable") {
          const r = await desligarMfa();
          setMfa(r);
          setRecuperacao(null);
          setEstado("desligado");
        } else {
          const r = await gerarCodigosDeRecuperacao();
          setRecuperacao(r);
          setMfa(null);
          setEstado("codigos");
        }
      } catch (causa) {
        if (causa instanceof MfaSemSessaoError) {
          setErro(causa.message);
          setEstado("sem-sessao");
          return;
        }
        if (causa instanceof MfaRecusadoError) {
          setErro(causa.message);
          setEstado("recusado");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível correr.");
        setEstado("erro");
      }
    },
    [segredo, confirmado],
  );

  const codigos = recuperacao?.recovery_codes ?? [];

  return (
    <section
      data-testid="mfa"
      aria-labelledby="mfa-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="mfa-titulo" className="text-sm font-semibold text-neutral-100">
          Segundo factor (MFA)
        </h3>
        <span className="text-[11px] text-amber-300">o servidor não reautentica</span>
      </header>

      {/* O achado, sempre visível -- nao escondido num rodape. */}
      <p data-testid="mfa-achado" className="mb-3 text-[11px] leading-relaxed text-amber-300">
        {descreverAchadoDeReautenticacao()}
      </p>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="mfa-segredo"
          value={segredo}
          onChange={(e) => setSegredo(e.target.value)}
          placeholder="segredo TOTP (base32)"
          aria-label="Segredo TOTP"
          className="w-56 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="mfa-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo mexer no segundo factor
        </label>
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="mfa-ligar"
          onClick={() => void correr("enable")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Ligar MFA
        </button>
        <button
          type="button"
          data-testid="mfa-desligar"
          onClick={() => void correr("disable")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Desligar MFA
        </button>
        <button
          type="button"
          data-testid="mfa-codigos"
          onClick={() => void correr("codes")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Gerar códigos de recuperação
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="mfa-pronto" className="text-xs text-neutral-400">
          Nenhuma operação nesta sessão.
        </p>
      )}

      {estado === "a-correr" && (
        <p data-testid="mfa-a-correr" className="text-xs text-neutral-400">
          A correr...
        </p>
      )}

      {estado === "ligado" && (
        <p data-testid="mfa-ligado" className="text-xs font-semibold text-emerald-300">
          200 · {descreverEstado(mfa)}
        </p>
      )}

      {estado === "desligado" && (
        <p data-testid="mfa-desligado" className="text-xs font-semibold text-amber-300">
          200 · {descreverEstado(mfa)}
        </p>
      )}

      {estado === "codigos" && (
        <div className="space-y-2">
          <p data-testid="mfa-codigos-estado" className="text-xs font-semibold text-emerald-300">
            201 criado · {descreverCodigos(recuperacao)}
          </p>
          {/* Os codigos aparecem UMA vez. O painel nao os guarda nem reenvia. */}
          <ul data-testid="mfa-lista-codigos" className="font-mono text-[11px] text-neutral-200">
            {codigos.map((c, i) => (
              <li key={`${i}-${c.slice(0, 2)}`}>{c}</li>
            ))}
          </ul>
          <p data-testid="mfa-aviso" className="text-[11px] text-amber-300">
            {recuperacao?.warning ?? "Guarde estes códigos: são mostrados uma única vez."}
          </p>
        </div>
      )}

      {estado === "sem-sessao" && (
        <p data-testid="mfa-sem-sessao" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="mfa-recusado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="mfa-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="mfa-limite" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverLimiteDaConfirmacao()}
      </p>
    </section>
  );
}
