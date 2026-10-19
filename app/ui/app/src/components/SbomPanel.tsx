/**
 * FASE 28.8: o painel do SBOM dentro da pagina de Capacidades.
 *
 * O QUE ESTE PAINEL FAZ, E O QUE ELE DEIXA CLARO
 *
 *  1. **Nao tem campo de raiz.** A raiz vem de `HADES_RAIZ_SBOM` no servidor -- o cliente nao a
 *     escolhe: *nenhuma das duas fronteiras e' escolhida pelo cliente.*
 *  2. **Diz que o SBOM e' minimo** antes de haver resultado. *Um SBOM que inclui o que o programador
 *     supoe e' pior do que nenhum: quem le' acredita.*
 *  3. **Mostra `nao_produzido` por extenso** -- o que o provider nao faz e' dito, nao omitido.
 *  4. **Nao existe botao de assinar nem de atestar.** A rota nao oferece.
 *  5. **Um inventario de zero componentes nao e' erro:** e' um resultado, e o painel di-lo.
 */

import { useCallback, useState } from "react";

import {
  descreverTipoDeComponente,
  gerarSBOM,
  provaDoSBOM,
  resumirInventario,
  type RelatorioDeSBOM,
} from "@/lib/sbomClient";
import { mensagemDaFalhaCom } from "../lib/mensagemDeFalha";

type Estado = "inativo" | "a-gerar" | "pronto" | "erro";

/** Extrai a mensagem de uma falha sem inventar uma. */
function mensagemDeFalha(falha: unknown): string {
  return mensagemDaFalhaCom(falha, "nao foi possivel gerar o inventario");
}

export function SbomPanel() {
  const [estado, setEstado] = useState<Estado>("inativo");
  const [erro, setErro] = useState("");
  const [relatorio, setRelatorio] = useState<RelatorioDeSBOM | null>(null);

  const gerar = useCallback(async () => {
    setEstado("a-gerar");
    setErro("");
    setRelatorio(null);
    try {
      setRelatorio(await gerarSBOM());
      setEstado("pronto");
    } catch (falha) {
      setErro(mensagemDeFalha(falha));
      setEstado("erro");
    }
  }, []);

  const componentes = relatorio?.inventario.componentes ?? [];

  return (
    <section
      aria-labelledby="sbom-titulo"
      data-testid="sbom"
      className="rounded-lg border border-neutral-700 p-4"
    >
      <h2 id="sbom-titulo" className="text-lg font-semibold">
        Inventario de software (SBOM)
      </h2>
      {/* Dizer o que e' ANTES do botao. */}
      <p className="mt-1 text-sm text-neutral-400">
        Este painel gera um SBOM <strong>minimo</strong>: le' os ficheiros de dependencia declarados
        na raiz autorizada do projecto e lista os componentes.{" "}
        <strong>Nao assina, nao emite provenance e nao inclui dependencias transitivas.</strong> A
        raiz vem do ambiente do servidor, nunca deste painel.
      </p>

      <div className="mt-3">
        <button
          type="button"
          className="w-fit rounded border border-neutral-600 px-3 py-1 text-sm disabled:opacity-50"
          onClick={() => void gerar()}
          disabled={estado === "a-gerar"}
        >
          Gerar inventario
        </button>
      </div>

      {estado === "a-gerar" ? (
        <p className="mt-3 text-sm" aria-live="polite" data-testid="sbom-a-gerar">
          A ler as fontes de dependencia...
        </p>
      ) : null}

      {estado === "erro" ? (
        <p className="mt-3 text-sm text-red-400" role="alert" data-testid="sbom-erro">
          {erro}
        </p>
      ) : null}

      {estado === "pronto" && relatorio !== null ? (
        <div className="mt-3 space-y-2" aria-live="polite" data-testid="sbom-relatorio">
          <p className="text-sm font-medium" data-testid="sbom-resumo">
            {resumirInventario(relatorio)}
          </p>
          <p className="text-xs text-neutral-400" data-testid="sbom-prova">
            {provaDoSBOM(relatorio)}
          </p>
          <dl className="grid grid-cols-2 gap-1 text-xs">
            <dt className="text-neutral-400">Total</dt>
            <dd data-testid="sbom-total">{relatorio.inventario.total}</dd>
            <dt className="text-neutral-400">Sem licenca</dt>
            <dd data-testid="sbom-sem-licenca">{relatorio.inventario.sem_licenca}</dd>
            <dt className="text-neutral-400">Assinado</dt>
            <dd data-testid="sbom-assinado">{relatorio.assinado ? "SIM" : "nao"}</dd>
            <dt className="text-neutral-400">Provenance</dt>
            <dd data-testid="sbom-provenance">{relatorio.provenance ? "SIM" : "nao"}</dd>
          </dl>

          {componentes.length > 0 ? (
            <ul className="max-h-64 space-y-1 overflow-y-auto text-xs" data-testid="sbom-componentes">
              {componentes.map((componente) => (
                <li
                  key={`${componente.tipo}:${componente.nome}`}
                  className="flex flex-wrap items-baseline gap-2"
                >
                  <span className="font-mono">{componente.nome}</span>
                  <span className="text-neutral-500">{componente.versao}</span>
                  <span className="text-neutral-500">
                    {descreverTipoDeComponente(componente.tipo)}
                  </span>
                  {componente.direto ? (
                    <span className="text-neutral-500">direto</span>
                  ) : (
                    <span className="text-neutral-600">indireto</span>
                  )}
                  {/* Um componente sem licenca declarada e' dito, nao escondido. */}
                  {componente.licenca === undefined || componente.licenca === "" ? (
                    <span className="text-amber-500">sem licenca declarada</span>
                  ) : (
                    <span className="text-neutral-500">{componente.licenca}</span>
                  )}
                </li>
              ))}
            </ul>
          ) : null}

          {/* O QUE NAO FOI PRODUZIDO, por extenso. */}
          {relatorio.inventario.nao_produzido !== null &&
          relatorio.inventario.nao_produzido.length > 0 ? (
            <div className="text-xs" data-testid="sbom-nao-produzido">
              <p className="text-neutral-400">Nao produzido por este provider:</p>
              <ul className="list-inside list-disc text-neutral-500">
                {relatorio.inventario.nao_produzido.map((lacuna) => (
                  <li key={lacuna}>{lacuna}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {/* O aviso vem do backend, por inteiro. */}
          <p className="text-xs text-neutral-400" data-testid="sbom-aviso-fronteira">
            {relatorio.aviso_de_fronteira}
          </p>
        </div>
      ) : null}
    </section>
  );
}
