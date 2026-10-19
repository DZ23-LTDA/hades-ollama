// Instalação de pacote de plugin.
//
// A fronteira do caminho é do SERVIDOR. O painel mostra o caminho como o utilizador o escreveu e deixa o
// servidor decidir — uma segunda verdade sobre o que está «dentro» seria uma segunda fronteira.
//
// E o 403 não é uma coisa só: subsistema DESLIGADO (falta a env var) e caminho RECUSADO respondem o mesmo
// status e são coisas diferentes. O painel distingue-as.
import { useCallback, useState } from "react";
import {
  AssinaturaRecusadaError,
  CaminhoRecusadoError,
  descreverDigest,
  descreverInstalacao,
  descreverRecusaDoPacote,
  instalarPacote,
  PacoteRecusadoError,
  PacotesDesligadosError,
  validarAssinatura,
  validarCaminho,
  validarManifesto,
  VersaoEmConflitoError,
  type ManifestoDePlugin,
  type ResultadoDeInstalacao,
} from "@/lib/pluginBundleClient";

type Estado = "pronto" | "a-instalar" | "instalado" | "desligado" | "recusado" | "conflito" | "erro";

export function PluginBundlePanel() {
  const [id, setId] = useState("");
  const [versao, setVersao] = useState("");
  const [kind, setKind] = useState("tool");
  const [caminho, setCaminho] = useState("");
  const [sha256, setSha256] = useState("");
  const [assinatura, setAssinatura] = useState("");
  const [chave, setChave] = useState("");
  const [estado, setEstado] = useState<Estado>("pronto");
  const [erro, setErro] = useState<string | null>(null);
  const [r, setR] = useState<ResultadoDeInstalacao | null>(null);

  const instalar = useCallback(async () => {
    const manifesto: ManifestoDePlugin = { id, version: versao, kind };
    const artefacto = { artifact: id, sha256, signing_key_id: chave, signature: assinatura };

    const problemas = [
      validarManifesto(manifesto),
      validarCaminho(caminho),
      validarAssinatura(artefacto),
    ].filter((p): p is string => p !== null);
    if (problemas.length > 0) {
      setErro(problemas.join(" · "));
      setEstado("recusado");
      return;
    }

    setErro(null);
    setEstado("a-instalar");
    try {
      const resposta = await instalarPacote(manifesto, caminho, artefacto);
      setR(resposta);
      setEstado("instalado");
    } catch (causa) {
      setR(null);
      if (causa instanceof PacotesDesligadosError) {
        setErro(descreverRecusaDoPacote(causa));
        setEstado("desligado");
        return;
      }
      if (causa instanceof VersaoEmConflitoError) {
        setErro(descreverRecusaDoPacote(causa));
        setEstado("conflito");
        return;
      }
      if (
        causa instanceof CaminhoRecusadoError ||
        causa instanceof AssinaturaRecusadaError ||
        causa instanceof PacoteRecusadoError
      ) {
        setErro(descreverRecusaDoPacote(causa));
        setEstado("recusado");
        return;
      }
      setErro(causa instanceof Error ? causa.message : "Não foi possível instalar.");
      setEstado("erro");
    }
  }, [id, versao, kind, caminho, sha256, assinatura, chave]);

  const digest = descreverDigest({ artifact: id, sha256, signing_key_id: chave, signature: assinatura });

  return (
    <section
      data-testid="plugin-bundle"
      aria-labelledby="plugin-bundle-titulo"
      className="rounded-lg border border-neutral-800 bg-neutral-900/60 p-4"
    >
      <header className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 id="plugin-bundle-titulo" className="text-sm font-semibold text-neutral-100">
          Instalar pacote de plugin
        </h3>
        <span className="text-[11px] text-neutral-500">201 CREATED · fronteira resolvida no servidor</span>
      </header>

      <div className="mb-3 grid grid-cols-2 gap-2">
        <input
          data-testid="plugin-bundle-id"
          value={id}
          onChange={(e) => setId(e.target.value)}
          placeholder="id do plugin"
          aria-label="Id do plugin"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <input
          data-testid="plugin-bundle-versao"
          value={versao}
          onChange={(e) => setVersao(e.target.value)}
          placeholder="versão (ex.: 1.2.0)"
          aria-label="Versão"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <select
          data-testid="plugin-bundle-kind"
          value={kind}
          onChange={(e) => setKind(e.target.value)}
          aria-label="Tipo de plugin"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 text-xs text-neutral-200"
        >
          <option value="tool">tool</option>
          <option value="panel">panel</option>
          <option value="provider">provider</option>
          <option value="agent">agent</option>
        </select>
        <input
          data-testid="plugin-bundle-caminho"
          value={caminho}
          onChange={(e) => setCaminho(e.target.value)}
          placeholder="bundle_path (resolvido pelo servidor)"
          aria-label="Caminho do pacote"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <input
          data-testid="plugin-bundle-sha256"
          value={sha256}
          onChange={(e) => setSha256(e.target.value)}
          placeholder="sha256 do artefacto (64 hex)"
          aria-label="Digest SHA-256"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <input
          data-testid="plugin-bundle-assinatura"
          value={assinatura}
          onChange={(e) => setAssinatura(e.target.value)}
          placeholder="signature"
          aria-label="Assinatura"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <input
          data-testid="plugin-bundle-chave"
          value={chave}
          onChange={(e) => setChave(e.target.value)}
          placeholder="signing_key_id"
          aria-label="Id da chave de assinatura"
          className="rounded border border-neutral-700 bg-neutral-950 px-2 py-1 font-mono text-xs text-neutral-200"
        />
        <button
          type="button"
          data-testid="plugin-bundle-instalar"
          onClick={() => void instalar()}
          className="rounded border border-neutral-700 px-2 py-1 text-xs text-neutral-300 hover:bg-neutral-800"
        >
          Instalar pacote
        </button>
      </div>

      {estado === "pronto" && (
        <p data-testid="plugin-bundle-pronto" className="text-xs text-neutral-400">
          Nada instalado nesta sessão.
        </p>
      )}

      {estado === "a-instalar" && (
        <p data-testid="plugin-bundle-a-instalar" className="text-xs text-neutral-400">
          A instalar...
        </p>
      )}

      {estado === "instalado" && (
        <div className="space-y-1">
          <p data-testid="plugin-bundle-instalado" className="text-xs font-semibold text-emerald-300">
            {descreverInstalacao(r)}
          </p>
          <p data-testid="plugin-bundle-digest" className="font-mono text-[11px] text-neutral-400">
            {digest}
          </p>
        </div>
      )}

      {/* 403 com causa "desligado": NAO e' um erro do pedido. */}
      {estado === "desligado" && (
        <p data-testid="plugin-bundle-desligado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "conflito" && (
        <p data-testid="plugin-bundle-conflito" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "recusado" && (
        <p data-testid="plugin-bundle-recusado" className="text-xs text-amber-300">
          {erro}
        </p>
      )}

      {estado === "erro" && (
        <p data-testid="plugin-bundle-erro" role="alert" className="text-xs text-red-400">
          {erro}
        </p>
      )}

      <p data-testid="plugin-bundle-fronteira" className="mt-3 text-[11px] leading-relaxed text-neutral-500">
        O servidor resolve o caminho com `filepath.Abs` e <strong>EvalSymlinks antes de comparar</strong>, e
        recusa o que sai do diretório permitido. O painel não repete essa conta: um caminho que passa por um
        symlink não é o caminho que se vê, e uma segunda opinião sobre o que está «dentro» seria a mais
        fraca das duas.
      </p>
    </section>
  );
}
