// Mídia -- OCR e tom. Duas escritas que criam um Artifact, e o caminho é contido pelo servidor.
//
// NAO faz:
//   - NAO corre ao montar (são duas ESCRITAS que criam Artifacts);
//   - NAO reimplementa `containedPath` -- o painel não é a segunda fronteira;
//   - NAO trata 502 como erro do pedido (é a dependência que falhou);
//   - NAO diz "correu bem": diz que artefato ficou.
import { useCallback, useState } from "react";
import {
  descreverFronteiraDoCaminho,
  descreverResultado,
  executarOcr,
  gerarTom,
  MidiaAusenteError,
  MidiaIndisponivelError,
  MidiaRecusadaError,
  validarCaminhoDeEntrada,
  validarMissao,
  validarTom,
  type ResultadoDeMidia,
} from "@/lib/mediaClient";

type EstadoDoPainel = "pronto" | "a-correr" | "criado" | "recusado" | "indisponivel" | "ausente" | "erro";

export function MediaPanel() {
  const [missao, setMissao] = useState("");
  const [caminho, setCaminho] = useState("");
  const [idioma, setIdioma] = useState("");
  const [frequencia, setFrequencia] = useState("");
  const [duracao, setDuracao] = useState("");
  const [confirmado, setConfirmado] = useState(false);
  const [estado, setEstado] = useState<EstadoDoPainel>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [r, setR] = useState<ResultadoDeMidia | null>(null);
  const [op, setOp] = useState<"ocr" | "tone">("ocr");

  const correr = useCallback(
    async (operacao: "ocr" | "tone") => {
      setOp(operacao);

      if (!confirmado) {
        setErro("Confirme antes de correr: isto cria um artefacto na missão.");
        setEstado("erro");
        return;
      }

      if (operacao === "ocr") {
        const e = validarMissao(missao) ?? validarCaminhoDeEntrada(caminho);
        if (e) {
          setErro(e);
          setEstado("recusado");
          return;
        }
      } else {
        const freq = Number(frequencia);
        const dur = Number(duracao);
        const e =
          validarMissao(missao) ??
          validarTom(Number.isFinite(freq) ? freq : undefined, Number.isFinite(dur) ? dur : undefined);
        if (e) {
          setErro(e);
          setEstado("recusado");
          return;
        }
      }

      setErro(null);
      setEstado("a-correr");
      try {
        const resposta =
          operacao === "ocr"
            ? await executarOcr({ missionId: missao, inputPath: caminho, language: idioma })
            : await gerarTom({ missionId: missao, frequency: Number(frequencia), durationMs: Number(duracao) });
        setR(resposta);
        setEstado("criado");
      } catch (causa) {
        setR(null);
        if (causa instanceof MidiaAusenteError) {
          setErro(causa.message);
          setEstado("ausente");
          return;
        }
        if (causa instanceof MidiaIndisponivelError) {
          setErro(causa.message);
          setEstado("indisponivel");
          return;
        }
        if (causa instanceof MidiaRecusadaError) {
          setErro(causa.message);
          setEstado("recusado");
          return;
        }
        setErro(causa instanceof Error ? causa.message : "Não foi possível correr.");
        setEstado("erro");
      }
    },
    [missao, caminho, idioma, frequencia, duracao, confirmado],
  );

  return (
    <section
      data-testid="midia"
      aria-labelledby="midia-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="midia-titulo" className="text-sm font-semibold text-neutral-100">
          Mídia: OCR e tom
        </h3>
        <label className="flex items-center gap-1 text-[11px] text-neutral-400">
          <input
            data-testid="midia-confirmar"
            type="checkbox"
            checked={confirmado}
            onChange={(e) => setConfirmado(e.target.checked)}
          />
          confirmo que isto cria um artefacto
        </label>
      </header>

      <div className="mb-3 flex flex-wrap items-center gap-2">
        <input
          data-testid="midia-missao"
          value={missao}
          onChange={(e) => setMissao(e.target.value)}
          placeholder="mission_id"
          aria-label="Identificador da missão"
          className="w-36 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="midia-caminho"
          value={caminho}
          onChange={(e) => setCaminho(e.target.value)}
          placeholder="caminho relativo"
          aria-label="Caminho de entrada"
          className="w-44 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="midia-idioma"
          value={idioma}
          onChange={(e) => setIdioma(e.target.value)}
          placeholder="idioma"
          aria-label="Idioma"
          className="w-20 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="midia-frequencia"
          value={frequencia}
          onChange={(e) => setFrequencia(e.target.value)}
          placeholder="Hz"
          aria-label="Frequência"
          inputMode="numeric"
          className="w-20 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
        <input
          data-testid="midia-duracao"
          value={duracao}
          onChange={(e) => setDuracao(e.target.value)}
          placeholder="ms"
          aria-label="Duração em milissegundos"
          inputMode="numeric"
          className="w-20 rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        />
      </div>

      <div className="mb-3 flex flex-wrap gap-2">
        <button
          type="button"
          data-testid="midia-ocr"
          onClick={() => void correr("ocr")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Executar OCR
        </button>
        <button
          type="button"
          data-testid="midia-tom"
          onClick={() => void correr("tone")}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Gerar tom
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="midia-pronto" className="text-xs text-neutral-400">
          Nada criado nesta sessão. São duas escritas que criam um artefacto — confirme para correr.
        </p>
      )}

      {estado === "a-correr" && (
        <p data-testid="midia-a-correr" className="text-xs text-neutral-400">
          A correr...
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="midia-recusado" className="text-xs text-amber-300">
          O servidor recusou: {erro}
        </p>
      )}

      {/* 502: o pedido estava certo, a dependencia falhou. Nao e' erro do cliente. */}
      {estado === "indisponivel" && (
        <p data-testid="midia-indisponivel" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "ausente" && (
        <p data-testid="midia-ausente" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="midia-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      {estado === "criado" && (
        <p data-testid="midia-resultado" className="text-xs font-semibold text-emerald-300">
          201 criado · {op === "ocr" ? "OCR" : "tom"} · {descreverResultado(r)}
        </p>
      )}

      <p data-testid="midia-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        {descreverFronteiraDoCaminho()} Os dois devolvem <strong>201</strong> e não 200: criam um Artifact —
        o produto não é «correu bem», é a coisa que ficou.
      </p>
    </section>
  );
}
