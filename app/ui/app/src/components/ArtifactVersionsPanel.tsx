import { useCallback, useEffect, useState } from "react";
import { ArrowUturnLeftIcon } from "@heroicons/react/24/outline";
import {
  carregarVersoesDoArtefacto,
  restaurarVersaoDoArtefacto,
  versoesIndisponiveis,
  type VersoesDoArtefacto,
} from "@/lib/artifactVersionsApi";

/**
 * Painel de VERSOES de um artefacto (Estagio 47 / item 47).
 *
 * *Ate' aqui o versionamento existia no dominio e nao tinha ecra: o backend versiona, compara e
 * restaura, e o utilizador nao via nada.* Este painel e' a ultima milha - e, como no Companion,
 * **nao inventa**: quando o servidor recusa (400 sem tenant, 404 de outra missao, 409 de CAS)
 * mostra a razao que veio, em vez de esconder ou fingir sucesso.
 */

type Estado =
  | { tipo: "a_carregar" }
  | { tipo: "pronta"; dados: VersoesDoArtefacto }
  | { tipo: "falhou"; motivo: string };

/** A decisao pura do painel - *extraida para ser testavel sem DOM (`vitest` nao tem jsdom).* */
export function estadoParaMostrar(
  resultado:
    { ok: true; dados: VersoesDoArtefacto } | { ok: false; motivo: string },
): Estado {
  if (resultado.ok) {
    return { tipo: "pronta", dados: resultado.dados };
  }
  // *Falha nao vira lista vazia:* uma lista vazia diz «nao ha versoes», e isso seria falso.
  return { tipo: "falhou", motivo: resultado.motivo };
}

/** Texto mostrado ao utilizador; nenhum caso devolve string vazia. */
export function textoDoEstado(estado: Estado): string {
  if (estado.tipo === "a_carregar") return "A carregar o historico de versoes…";
  if (estado.tipo === "falhou")
    return `Nao foi possivel ler as versoes: ${estado.motivo}`;
  if (estado.dados.versoes.length === 0)
    return "Sem versoes registadas para este artefacto.";
  return `${estado.dados.versoes.length} versao(oes) · ${estado.dados.scope}`;
}

export function ArtifactVersionsPanel({
  missionId,
  artifactId,
}: {
  missionId: string;
  artifactId: string;
}) {
  const [estado, setEstado] = useState<Estado>({ tipo: "a_carregar" });
  const [aRestaurar, setARestaurar] = useState<number | null>(null);
  const [recado, setRecado] = useState<string | null>(null);

  const carregar = useCallback(async () => {
    setEstado({ tipo: "a_carregar" });
    try {
      const dados = await carregarVersoesDoArtefacto(missionId, artifactId);
      setEstado(estadoParaMostrar({ ok: true, dados }));
    } catch (erro) {
      const motivo = erro instanceof Error ? erro.message : String(erro);
      setEstado(estadoParaMostrar({ ok: false, motivo }));
      // O painel tambem declara a ausencia no `scope` para quem so' olha para os dados.
      void versoesIndisponiveis(motivo);
    }
  }, [missionId, artifactId]);

  useEffect(() => {
    void carregar();
  }, [carregar]);

  if (!missionId || !artifactId) {
    return (
      <p
        className="text-xs text-neutral-500"
        data-versions-panel="sem-artefacto"
      >
        Escolha um artefacto para ver o historico de versoes.
      </p>
    );
  }

  const versoes = estado.tipo === "pronta" ? estado.dados.versoes : [];
  const corrente = versoes.reduce(
    (maior, v) => (v.version > maior ? v.version : maior),
    0,
  );

  const restaurar = async (versao: number) => {
    setARestaurar(versao);
    setRecado(null);
    try {
      // `expected_version` = versao CORRENTE (o CAS do artefacto): *sem ele o servidor recusa
      // com 409, e e' isso que impede restaurar sobre uma versao que o cliente nao viu.*
      const restaurado = await restaurarVersaoDoArtefacto(
        missionId,
        artifactId,
        versao,
        corrente,
      );
      setRecado(
        restaurado
          ? `Versao ${versao} restaurada como versao ${restaurado.version}.`
          : `O servidor aceitou o pedido mas nao devolveu o artefacto restaurado.`,
      );
      await carregar();
    } catch (erro) {
      // A mensagem do SERVIDOR (400 sem tenant / 404 / 409) vai para o ecra tal como veio.
      setRecado(
        `O servidor recusou o restauro: ${erro instanceof Error ? erro.message : String(erro)}`,
      );
    } finally {
      setARestaurar(null);
    }
  };

  return (
    <section className="flex flex-col gap-2 rounded-lg border border-neutral-200 p-3 text-xs dark:border-neutral-800">
      <header className="flex items-center justify-between">
        <h3 className="font-semibold">Versoes</h3>
        <span className="text-neutral-500">{textoDoEstado(estado)}</span>
      </header>

      {recado ? (
        <p
          className="rounded bg-neutral-100 p-2 dark:bg-neutral-900"
          data-versions-recado
        >
          {recado}
        </p>
      ) : null}

      {versoes.length > 0 ? (
        <ul className="flex flex-col gap-1">
          {versoes.map((versao) => (
            <li
              key={`${versao.id}-${versao.version}`}
              className="flex items-center justify-between gap-3"
            >
              <span className="truncate">
                v{versao.version}
                {versao.sha256 ? ` · ${versao.sha256.slice(0, 12)}` : ""}
                {versao.version === corrente ? " · actual" : ""}
              </span>
              {versao.version !== corrente ? (
                <button
                  type="button"
                  className="inline-flex items-center gap-1 rounded border border-neutral-300 px-2 py-1 hover:bg-neutral-100 disabled:opacity-50 dark:border-neutral-700 dark:hover:bg-neutral-900"
                  disabled={aRestaurar !== null}
                  onClick={() => void restaurar(versao.version)}
                >
                  <ArrowUturnLeftIcon className="h-3 w-3" />
                  {aRestaurar === versao.version ? "A restaurar…" : "Restaurar"}
                </button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}
