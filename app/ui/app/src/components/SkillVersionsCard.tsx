import { useEffect, useState } from "react";

import { getSkillVersions, restoreSkillVersion, type SkillVersionRestaurada } from "../lib/agenticClient";

type Props = { skillId: string };

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
export function SkillVersionsCard({ skillId }: Props) {
  const [versoes, setVersoes] = useState<string[] | null>(null);
  const [restaurada, setRestaurada] = useState<SkillVersionRestaurada | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [aRestaurar, setARestaurar] = useState<string | null>(null);

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
              <button
                type="button"
                onClick={() => void restaurar(versao)}
                disabled={aRestaurar !== null}
                className="min-h-9 rounded-lg bg-neutral-900 px-3 py-2 text-xs font-medium text-white disabled:opacity-40 dark:bg-neutral-100 dark:text-neutral-900"
              >
                {aRestaurar === versao ? "A restaurar…" : "Restaurar esta versão"}
              </button>
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
    </section>
  );
}
