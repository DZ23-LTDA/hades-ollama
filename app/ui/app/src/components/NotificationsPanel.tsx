// Notificações -- registo do dispositivo para push.
//
// NAO faz:
//   - NAO mostra o token: o servidor nao o pode devolver (`json:"-"`);
//   - NAO trata 501 como falha temporaria (e' configuracao em falta);
//   - NAO guarda o token em estado do React mais do que o necessario para o enviar.
import { useCallback, useState } from "react";
import {
  descreverAusenciaDoToken,
  descreverSubscricao,
  plataformasConhecidas,
  PushNaoImplementadoError,
  PushRecusadoError,
  registarDispositivo,
  validarRegisto,
  type SubscricaoDePush,
} from "@/lib/pushClient";

type Estado = "pronto" | "a-registar" | "registado" | "nao-implementado" | "recusado" | "erro";

export function NotificationsPanel() {
  const [token, setToken] = useState("");
  const [plataforma, setPlataforma] = useState("web");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [subscricao, setSubscricao] = useState<SubscricaoDePush | null>(null);

  const registar = useCallback(async () => {
    const problema = validarRegisto(token, plataforma);
    if (problema) {
      setErro(problema);
      setEstado("recusado");
      return;
    }

    setErro(null);
    setEstado("a-registar");
    try {
      const r = await registarDispositivo({ token, platform: plataforma });
      setSubscricao(r);
      // O token nao fica no estado depois de enviado.
      setToken("");
      setEstado("registado");
    } catch (causa) {
      setSubscricao(null);
      if (causa instanceof PushNaoImplementadoError) {
        setErro(causa.message);
        setEstado("nao-implementado");
        return;
      }
      if (causa instanceof PushRecusadoError) {
        setErro(causa.message);
        setEstado("recusado");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível registar.");
      setEstado("erro");
    }
  }, [token, plataforma]);

  return (
    <section
      data-testid="notificacoes"
      aria-labelledby="notificacoes-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="notificacoes-titulo" className="text-sm font-semibold text-neutral-100">
          Notificações push
        </h3>
        <span className="text-[11px] text-neutral-500">o token nunca volta do servidor</span>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="notificacoes-token"
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="token do dispositivo"
          aria-label="Token do dispositivo"
          type="password"
          autoComplete="off"
          className="w-56 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <select
          data-testid="notificacoes-plataforma"
          value={plataforma}
          onChange={(e) => setPlataforma(e.target.value)}
          aria-label="Plataforma"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        >
          {plataformasConhecidas().map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
        <button
          type="button"
          data-testid="notificacoes-registar"
          onClick={() => void registar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Registar dispositivo
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="notificacoes-pronto" className="text-xs text-neutral-400">
          Nenhum dispositivo registado nesta sessão.
        </p>
      )}

      {estado === "a-registar" && (
        <p data-testid="notificacoes-a-registar" className="text-xs text-neutral-400">
          A registar...
        </p>
      )}

      {estado === "registado" && (
        <p data-testid="notificacoes-registado" className="text-xs font-semibold text-emerald-300">
          201 criado · {descreverSubscricao(subscricao)}
        </p>
      )}

      {/* 501: nao implementado. Nao e' o mesmo que 503. */}
      {estado === "nao-implementado" && (
        <p data-testid="notificacoes-nao-implementado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="notificacoes-recusado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="notificacoes-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="notificacoes-token-ausente" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverAusenciaDoToken()}
      </p>
    </section>
  );
}
