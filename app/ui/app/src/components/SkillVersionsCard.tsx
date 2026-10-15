import { useEffect, useState } from "react";

import {
  getSkillDifference,
  getSkillVersions,
  restoreSkillVersion,
  type SkillDiferenca,
  type SkillVersionRestaurada,
} from "../lib/agenticClient";

type Props = { skillId: string; activeVersion?: string };

/**
 * SkillVersionsCard lista o HISTÓRICO de uma skill e permite restaurar uma versão antiga (E-644).
 *
 * O QUE ESTA SUPERFÍCIE DIZ, E É O PONTO: ***restaurar recupera o CONTEÚDO, não a confiança***.
 * Depois de restaurar, o cartão afixa o estado devolvido pelo servidor (`trusted`), sem o
 * arredondar: uma versão restaurada que aparecesse como confiável seria uma escalada silenciosa
 * apresentada como normalidade.
 *
 * E não inventa: se a lista vier vazia, diz que não há versões anteriores — não supõe que a skill
 * não existe (o servidor responde vazio também para skill de outra organização, de propósito).
 */
export function SkillVersionsCard({ skillId, activeVersion }: Props) {
  const [versoes, setVersoes] = useState<string[] | null>(null);
  const [restaurada, setRestaurada] = useState<SkillVersionRestaurada | null>(null);
  const [diferenca, setDiferenca] = useState<SkillDiferenca | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [aRestaurar, setARestaurar] = useState<string | null>(null);

  const comparar = async (versao: string) => {
    if (!activeVersion) return;
    setErro(null);
    try {
      const resposta = await getSkillDifference(skillId, versao, activeVersion);
      setDiferenca(resposta.difference);
    } catch (falha: unknown) {
      setErro(falha instanceof Error ? falha.message : "falha ao comparar");
    }
  };

  useEffect(() => {
    let cancelado = false;
    void getSkillVersions(skillId)
      .then((resposta) => {
        if (!cancelado) setVersoes(resposta.versions ?? []);
      })
      .catch((falha: unknown) => {
        if (!cancelado) setErro(falha instanceof Error ? falha.message : "indisponível");
      });
    return () => {
      cancelado = true;
    };
  }, [skillId]);

  const restaurar = async (versao: string) => {
    setARestaurar(versao);
    setErro(null);
    try {
      const resposta = await restoreSkillVersion(skillId, versao);
      setRestaurada(resposta.skill);
    } catch (falha: unknown) {
      setErro(falha instanceof Error ? falha.message : "falha ao restaurar");
    } finally {
      setARestaurar(null);
    }
  };

  return (
    <section aria-labelledby="skill-versions-heading" className="mt-5 rounded-xl border border-neutral-200 bg-neutral-50 p-4 dark:border-neutral-800 dark:bg-neutral-900/40">
      <h3 id="skill-versions-heading" className="text-sm font-semibold">Histórico de versões</h3>
      <p className="mt-1 text-xs text-neutral-500">{skillId}</p>

      {erro ? <p className="mt-2 text-xs text-neutral-500" role="status">Não foi possível: {erro}</p> : null}

      {versoes === null && !erro ? (
        <p className="mt-2 text-xs text-neutral-500" role="status">A carregar versões…</p>
      ) : null}

      {versoes !== null && versoes.length === 0 ? (
        <p className="mt-2 text-xs text-neutral-500" role="status">Nenhuma versão anterior arquivada.</p>
      ) : null}

      {versoes && versoes.length > 0 ? (
        <ul className="mt-3 space-y-2">
          {versoes.map((versao) => (
            <li key={versao} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-neutral-200 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-950">
              <span className="text-xs font-medium">{versao}</span>
              <div className="flex flex-wrap gap-2">
                {activeVersion ? (
                  <button
                    type="button"
                    onClick={() => void comparar(versao)}
                    className="min-h-9 rounded-lg border border-neutral-300 px-3 py-2 text-xs font-medium dark:border-neutral-700"
                  >
                    Comparar com a activa
                  </button>
                ) : null}
                <button
                  type="button"
                  onClick={() => void restaurar(versao)}
                  disabled={aRestaurar !== null}
                  className="min-h-9 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900"
                >
                  {aRestaurar === versao ? "A restaurar…" : "Restaurar esta versão"}
                </button>
              </div>
            </li>
          ))}
        </ul>
      ) : null}

      {restaurada ? (
        <p className="mt-3 text-xs" role="status">
          <span className="text-neutral-500">Versão activa agora: </span>
          <span className="font-medium">{restaurada.version}</span>
          {/* A CONFIANÇA É MOSTRADA COMO O SERVIDOR A DEVOLVEU. Restaurar não a promove. */}
          <span className="text-neutral-500"> · confiança: {restaurada.trusted ? "confiável" : "não confiável (exige promoção com política)"}</span>
        </p>
      ) : null}

      {diferenca ? (
        <div className="mt-3 rounded-lg border border-neutral-200 bg-white p-3 dark:border-neutral-800 dark:bg-neutral-950">
          <p className="text-[11px] font-medium">
            Diferença de {diferenca.from} para {diferenca.to}
          </p>
          {diferenca.changes.length === 0 ? (
            <p className="mt-1 text-[11px] text-neutral-500">Sem diferenças de conteúdo.</p>
          ) : (
            <ul className="mt-1 space-y-1 text-[11px] text-neutral-600 dark:text-neutral-300">
              {diferenca.changes.map((mudanca) => (
                <li key={mudanca.field}>
                  <span className="font-medium">{mudanca.field}</span>: {mudanca.from || "—"} → {mudanca.to || "—"}
                </li>
              ))}
            </ul>
          )}
          {/* INTEGRIDADE DITA À PARTE: não é «conteúdo de trabalho», é a prova de que o conteúdo é o
              que diz ser. Misturá-la com a descrição faria a diferença mais importante perder-se. */}
          <p className="mt-2 text-[11px] text-neutral-500">
            {diferenca.content_changed ? "Conteúdo (hash) mudou." : "Conteúdo (hash) igual."}{" "}
            {diferenca.signature_changed ? "Assinatura mudou." : "Assinatura igual."}
          </p>
        </div>
      ) : null}
    </section>
  );
}
