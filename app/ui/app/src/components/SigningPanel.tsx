/**
 * FASE 28.8: o painel de ASSINATURA dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL FAZ, E O QUE ELE DEIXA CLARO
 *
 *  1. **Diz que so' avalia** antes de haver resultado -- *nao assina*.
 *  2. **So' aceita um CAMINHO de certificado publico.** Nao ha' campo para chave privada nem para
 *     senha: *um painel que tem um campo para o segredo acaba por o receber.*
 *  3. **Mostra o motivo por extenso** de cada negacao -- dizer "falta o EKU" a um certificado
 *     expirado manda quem le' corrigir a coisa errada; a ordem dos testes garante que o motivo e' o
 *     primeiro que falhou.
 *  4. **Nao existe botao de assinar.** A rota nao oferece, o cliente nao tem, o painel nao inventa.
 *  5. **Mostra os alvos suportados** que o backend declara, para nao prometer tipos que nao existem.
 */

import { useCallback, useState } from "react";

import {
  ambientesDeAssinatura,
  descreverAptidao,
  descreverTipoDeAssinatura,
  planearAssinatura,
  provaDoPlanoDeAssinatura,
  type PlanoDeAssinatura,
  type TipoDeAssinatura,
} from "@/lib/signingPlanClient";
import { mensagemDaFalhaCom } from "../lib/mensagemDeFalha";

type Estado = "inativo" | "a-avaliar" | "pronto" | "erro";

/** Extrai a mensagem de uma falha sem inventar uma. */
function mensagemDeFalha(falha: unknown): string {
  return mensagemDaFalhaCom(falha, "nao foi possivel avaliar a identidade");
}

export function SigningPanel() {
  const [tipo, setTipo] = useState<TipoDeAssinatura>("AUTHENTICODE");
  const [caminho, setCaminho] = useState("");
  const [rotulo, setRotulo] = useState("");
  const [ambiente, setAmbiente] = useState("desenvolvimento");
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState("");
  const [plano, setPlano] = useState<PlanoDeAssinatura | null>(null);

  const avaliar = useCallback(async () => {
    setEstado("a-avaliar");
    setErro("");
    setPlano(null);
    try {
      setPlano(
        await planearAssinatura(tipo, {
          caminho_do_certificado: caminho,
          rotulo,
          ambiente,
        }),
      );
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, [tipo, caminho, rotulo, ambiente]);

  return (
    <section
      aria-labelledby="assinatura-titulo"
      data-testid="assinatura"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="assinatura-titulo" className="text-lg font-semibold">
        Identidade de assinatura
      </h2>
      {/* Dizer o que faz ANTES do botao. */}
      <p className="mt-1 text-sm text-neutral-400">
        Este painel apenas <strong>avalia</strong> se uma identidade serve para um tipo de assinatura:
        le' os metadados X.509 do certificado <strong>publico</strong> e verifica validade, emissor e
        EKU. <strong>Nao assina, nao le' a chave privada e nao escreve.</strong>
      </p>

      <div className="mt-3 flex flex-col gap-2">
        <label className="flex flex-col gap-1 text-sm">
          <span>Tipo de assinatura</span>
          <select
            id="assinatura-tipo"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={tipo}
            onChange={(evento) => setTipo(evento.target.value as TipoDeAssinatura)}
          >
            <option value="AUTHENTICODE">Authenticode (Windows)</option>
            <option value="APPLE_DEVELOPER_ID">Apple Developer ID (macOS)</option>
            <option value="ANDROID_APK">Android APK</option>
            <option value="OPENPGP">OpenPGP</option>
          </select>
        </label>

        <label className="flex flex-col gap-1 text-sm">
          <span>Certificado publico (.cer/.pem/.crt)</span>
          <input
            id="assinatura-certificado"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={caminho}
            onChange={(evento) => setCaminho(evento.target.value)}
            placeholder="C:/caminho/para/certificado.cer"
            autoComplete="off"
            spellCheck={false}
          />
        </label>

        <label className="flex flex-col gap-1 text-sm">
          <span>Rotulo (opcional)</span>
          <input
            id="assinatura-rotulo"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={rotulo}
            onChange={(evento) => setRotulo(evento.target.value)}
            autoComplete="off"
          />
        </label>

        <label className="flex flex-col gap-1 text-sm">
          <span>Ambiente</span>
          <select
            id="assinatura-ambiente"
            className="rounded border border-neutral-700 bg-transparent px-2 py-1"
            value={ambiente}
            onChange={(evento) => setAmbiente(evento.target.value)}
          >
            {ambientesDeAssinatura().map((nome) => (
              <option key={nome} value={nome}>
                {nome}
              </option>
            ))}
          </select>
        </label>

        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void avaliar()}
          disabled={estado === "a-avaliar" || caminho.trim() === ""}
        >
          Avaliar identidade
        </button>
      </div>

      {estado === "a-avaliar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="assinatura-a-avaliar">
          A ler os metadados do certificado...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="assinatura-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && plano !== null ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="assinatura-plano">
          <p className="text-sm font-medium" data-testid="assinatura-aptidao">
            {descreverAptidao(plano.avaliacao.aptidao)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="assinatura-prova">
            {provaDoPlanoDeAssinatura(plano)}
          </p>
          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Apta</dt>
            <dd data-testid="assinatura-apta">{plano.avaliacao.apta ? "sim" : "nao"}</dd>
            <dt className="text-neutral-400">Assinado</dt>
            <dd data-testid="assinatura-nada-assinado">
              {plano.nada_assinado ? "nao" : "SIM (inesperado)"}
            </dd>
            <dt className="text-neutral-400">Sujeito</dt>
            <dd className="break-all">{plano.avaliacao.sujeito}</dd>
            <dt className="text-neutral-400">Emissor</dt>
            <dd className="break-all">{plano.avaliacao.emissor}</dd>
            <dt className="text-neutral-400">Valido ate'</dt>
            <dd data-testid="assinatura-valido-ate">{plano.avaliacao.valido_ate}</dd>
          </dl>
          {plano.avaliacao.motivo !== "" ? (
            <p className="text-xs text-amber-400" data-testid="assinatura-motivo">
              {plano.avaliacao.motivo}
            </p>
          ) : null}
          {plano.alvos_suportados !== null && plano.alvos_suportados.length > 0 ? (
            <p className="text-xs text-neutral-400" data-testid="assinatura-alvos">
              Alvos suportados:{" "}
              {plano.alvos_suportados.map(descreverTipoDeAssinatura).join(", ")}
            </p>
          ) : null}
          {/* O aviso vem do backend, por inteiro. */}
          <p className="text-xs text-neutral-400" data-testid="assinatura-aviso-fronteira">
            {plano.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
