import { useCallback, useEffect, useState } from "react";

import {
  corDaAcao,
  fetchInstallCatalogo,
  fetchInstallPlano,
  rotuloDaAcao,
  type InstallCatalogo,
  type InstallPlano,
} from "@/lib/toolchainClient";

/**
 * FASE 28.6-A / 28.8: a seccao de PERFIS DE INSTALACAO dentro da pagina de Capacidades.
 *
 * Nao e' um painel paralelo. Vive na mesma area `Capacidades e Ferramentas`, ao lado do
 * preflight de toolchain, porque responde a mesma pergunta: *o que falta nesta maquina?*
 *
 * Tres decisoes que os testes travam:
 *
 *  1. O perfil e' INTENCAO, nao permissao. Um componente que exige elevacao/servico/reinicio
 *     aparece como "Requer aprovacao", nunca como "pronto a instalar".
 *  2. `PULAR` (UNKNOWN) usa ambar, nao vermelho. *Nao saber nao e' falhar.*
 *  3. NAO existe botao de instalar. A rota devolve `executa_algo: false` e a UI nao oferece o
 *     que nao existe. *Um botao "Instalar" na UI de diagnostico mentiria: instalar exige
 *     approval e superficie propria.*
 */

const PERFIS_ORDENADOS = [
  "BASIC",
  "RECOMMENDED",
  "DEVELOPER_COMPLETE",
  "MOBILE",
  "MEDIA_AND_VOICE",
  "BROWSER_AND_COMPUTER_USE",
  "CONTAINERS_AND_CLOUD",
  "CUSTOM",
  "FULL_OFFLINE",
] as const;

type Estado = "a-carregar" | "pronto" | "erro";

export function InstallProfilesPanel() {
  const [catalogo, setCatalogo] = useState<InstallCatalogo | null>(null);
  const [plano, setPlano] = useState<InstallPlano | null>(null);
  const [perfilActivo, setPerfilActivo] = useState<string>("BASIC");
  const [estado, setEstado] = useState<Estado>("a-carregar");
  const [erro, setErro] = useState<string>("");

  useEffect(() => {
    let vivo = true;
    setEstado("a-carregar");
    fetchInstallCatalogo()
      .then((dados) => {
        if (!vivo) return;
        setCatalogo(dados);
        setEstado("pronto");
      })
      .catch((causa: unknown) => {
        if (!vivo) return;
        setErro(causa instanceof Error ? causa.message : String(causa));
        setEstado("erro");
      });
    return () => {
      vivo = false;
    };
  }, []);

  const carregarPlano = useCallback((perfil: string) => {
    setPerfilActivo(perfil);
    setPlano(null);
    fetchInstallPlano(perfil)
      .then((dados) => setPlano(dados))
      .catch((causa: unknown) => {
        setErro(causa instanceof Error ? causa.message : String(causa));
        setEstado("erro");
      });
  }, []);

  if (estado === "a-carregar") {
    return (
      <section aria-labelledby="perfis-instalacao" className="mt-10">
        <h3 id="perfis-instalacao" className="text-lg font-semibold text-neutral-100">
          Perfis de instalacao
        </h3>
        <p className="mt-2 text-sm text-neutral-400">A ler o catalogo de pacotes...</p>
      </section>
    );
  }

  if (estado === "erro") {
    return (
      <section aria-labelledby="perfis-instalacao" className="mt-10">
        <h3 id="perfis-instalacao" className="text-lg font-semibold text-neutral-100">
          Perfis de instalacao
        </h3>
        <p role="alert" className="mt-2 text-sm text-red-400">
          Nao foi possivel ler o catalogo de pacotes: {erro}
        </p>
      </section>
    );
  }

  const avisos = catalogo?.avisos ?? [];
  const perfis = catalogo?.perfis ?? [];
  // Ordem canonica primeiro; um perfil desconhecido no fim, para nada desaparecer em silencio.
  const ordenados = [...perfis].sort((a, b) => {
    const ia = PERFIS_ORDENADOS.indexOf(a.id as (typeof PERFIS_ORDENADOS)[number]);
    const ib = PERFIS_ORDENADOS.indexOf(b.id as (typeof PERFIS_ORDENADOS)[number]);
    const pa = ia === -1 ? PERFIS_ORDENADOS.length : ia;
    const pb = ib === -1 ? PERFIS_ORDENADOS.length : ib;
    return pa - pb;
  });

  return (
    <section aria-labelledby="perfis-instalacao" className="mt-10">
      <h3 id="perfis-instalacao" className="text-lg font-semibold text-neutral-100">
        Perfis de instalacao
      </h3>
      <p className="mt-1 text-sm text-neutral-400">
        O perfil diz o que <em>quer</em> instalar. Nao autoriza nada: componente que exige
        elevacao, servico ou reinicio continua a precisar de aprovacao.
      </p>

      {avisos.length > 0 && (
        <ul role="alert" className="mt-3 space-y-1 rounded border border-amber-700/40 p-3">
          {avisos.map((aviso) => (
            <li key={aviso} className="text-sm text-amber-400">
              {aviso}
            </li>
          ))}
        </ul>
      )}

      <ul className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {ordenados.map((perfil) => {
          const activo = perfil.id === perfilActivo;
          return (
            <li key={perfil.id}>
              <button
                type="button"
                aria-expanded={activo}
                onClick={() => carregarPlano(perfil.id)}
                className={`w-full rounded border p-3 text-left transition ${
                  activo
                    ? "border-sky-500 bg-sky-500/10"
                    : "border-neutral-700 hover:border-neutral-500"
                }`}
              >
                <span className="block text-sm font-semibold text-neutral-100">{perfil.id}</span>
                <span className="mt-1 block text-xs text-neutral-400">{perfil.descricao}</span>
                <span className="mt-1 block text-xs text-neutral-500">
                  {perfil.escolhe_pacotes
                    ? "escolha componente a componente"
                    : `${perfil.pacotes.length} pacotes · ${perfil.tamanho_mb} MB`}
                </span>
              </button>
            </li>
          );
        })}
      </ul>

      {plano && (
        <div className="mt-6 rounded border border-neutral-700 p-4">
          <p className="text-sm text-neutral-300">{plano.resumo}</p>

          {plano.bloqueios && plano.bloqueios.length > 0 && (
            <div className="mt-3">
              <h4 className="text-sm font-semibold text-amber-400">O que falta saber</h4>
              <ul className="mt-1 space-y-1">
                {plano.bloqueios.map((bloqueio) => (
                  <li key={bloqueio} className="text-xs text-neutral-400">
                    {bloqueio}
                  </li>
                ))}
              </ul>
            </div>
          )}

          <table className="mt-3 w-full text-left text-xs">
            <thead>
              <tr className="text-neutral-500">
                <th scope="col">Pacote</th>
                <th scope="col">Acao</th>
                <th scope="col">Motivo</th>
              </tr>
            </thead>
            <tbody>
              {plano.itens.map((item) => (
                <tr key={item.pacote_id} className="border-t border-neutral-800">
                  <td className="py-1 text-neutral-200">{item.nome}</td>
                  <td className={`py-1 ${corDaAcao(item.acao)}`}>{rotuloDaAcao(item.acao)}</td>
                  <td className="py-1 text-neutral-400">{item.motivo}</td>
                </tr>
              ))}
            </tbody>
          </table>

          {plano.itens.length === 0 && (
            <p className="mt-2 text-xs text-neutral-400">
              Este perfil nao inclui pacotes por omissao: a escolha e' do utilizador.
            </p>
          )}

          {plano.aprovacoes_necessarias > 0 && (
            <p className="mt-3 text-xs text-violet-400">
              {plano.aprovacoes_necessarias} componente(s) exigem aprovacao humana antes de
              instalar.
            </p>
          )}
        </div>
      )}
    </section>
  );
}
