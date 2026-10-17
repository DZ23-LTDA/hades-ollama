import { useEffect, useState } from "react";
import { CompanionMascote } from "./CompanionMascote";
import {
  PREFERENCIAS_PADRAO,
  carregarPreferencias,
  guardarPreferencias,
  aplicarAccao,
  type AccaoDePreferencia,
  restaurarPreferencias,
  sistemaPedeMenosMovimento as lerSistemaPedeMenosMovimento,
  type CompanionPreferencias,
} from "@/lib/companionPreferencias";
import {
  carregarEstadoDoCompanion,
  estadoIndisponivel,
  type EstadoDoCompanion,
} from "@/lib/companionApi";

/**
 * A LIGACAO do Companion (item 46). *Este ficheiro NAO decide nada:* le' a rota
 * testada e entrega o resultado ao componente. Ate' aqui, o `CompanionMascote`
 * estava construido e testado e **nao era renderizado em lado nenhum** - a classe
 * de defeito que esta missao persegue: existe, e' testado, e ninguem o ve.
 *
 * ***`E-1086` (§FASE-3-A, item 4) — AS PREFERÊNCIAS CHEGAM AQUI.*** O `E-1085` ligou-as ao
 * `CompanionMascote`, mas **o componente que a rota monta é ESTE**, e ele não as passava. ⇒ *as
 * preferências existiam, o componente sabia usá-las, e **ninguém as entregava** — a mesma classe de
 * defeito, um nível acima: «existe, é testado, e não chega ao utilizador».*
 */

/** O resultado de tentar saber o estado. *Explicito, para que a falha nao vire estado bonito.* */
export type ResultadoDoEstado =
  | { tipo: "ok"; estado: EstadoDoCompanion }
  | { tipo: "falhou"; motivo: string };

/**
 * A unica decisao deste ficheiro - e por isso esta' extraida e testada:
 * **em caso de falha, mostra-se a AUSENCIA declarada, nunca um estado inventado.**
 */
export function estadoParaMostrar(
  resultado: ResultadoDoEstado,
): EstadoDoCompanion {
  if (resultado.tipo === "ok") {
    return resultado.estado;
  }
  return estadoIndisponivel(resultado.motivo);
}

export function CompanionLigado({
  missionID,
  nome,
}: {
  missionID: string;
  nome?: string;
}) {
  const [resultado, setResultado] = useState<ResultadoDoEstado | null>(null);
  // ***`E-1086`: as preferências começam no DEFAULT e são lidas no mount.*** *A leitura é síncrona
  // (`localStorage`), logo não há um segundo render a piscar o pet.*
  const [preferencias, setPreferencias] = useState<CompanionPreferencias>(PREFERENCIAS_PADRAO);
  const [naoPersistiu, setNaoPersistiu] = useState(false);
  const [sistemaReduzido] = useState<boolean>(() => lerSistemaPedeMenosMovimento());

  useEffect(() => {
    // *Lida uma vez, no mount: a preferência é deste dispositivo, e não depende da missão.*
    setPreferencias(carregarPreferencias());
  }, []);

  useEffect(() => {
    let vivo = true;
    carregarEstadoDoCompanion(missionID)
      .then((estado) => {
        if (vivo) setResultado({ tipo: "ok", estado });
      })
      .catch((erro: unknown) => {
        if (!vivo) return;
        const motivo = erro instanceof Error ? erro.message : String(erro);
        setResultado({ tipo: "falhou", motivo });
      });
    return () => {
      vivo = false;
    };
  }, [missionID]);

  // ***A GRAVAÇÃO É UMA ACÇÃO, E A FALHA DELA TAMBÉM SE MOSTRA.*** *O `guardarPreferencias` devolve
  // `false` quando não conseguiu; aqui isso vira um aviso, e não um silêncio.*
  const agir = (accao: AccaoDePreferencia) => {
    const novas = aplicarAccao(preferencias, accao);
    setPreferencias(novas);
    const persistiu = accao === "restaurar" ? restaurarPreferencias() : guardarPreferencias(novas);
    setNaoPersistiu(!persistiu);
  };

  // Enquanto nao se sabe, nao se afirma: *um mascote que comeca a "trabalhar" antes de
  // o servidor o dizer estaria a inventar.*
  if (resultado === null) {
    return (
      <div data-companion-a-carregar className="text-xs text-neutral-500">
        A carregar o estado do Companion…
      </div>
    );
  }

  const estado = estadoParaMostrar(resultado);
  return (
    <div className="flex flex-col gap-2" data-companion-ligado>
      <CompanionMascote
        estado={estado.estado}
        razao={estado.razao}
        nome={nome}
        preferencias={preferencias}
        sistemaPedeMenosMovimento={sistemaReduzido}
      />
      <div className="flex flex-wrap items-center gap-2" aria-label="Preferências do Companion">
        <button
          type="button"
          className="rounded-lg border border-neutral-300 px-2 py-1 text-xs dark:border-neutral-700"
          onClick={() => agir("alternarVisivel")}
        >
          {preferencias.visivel ? "Ocultar Companion" : "Mostrar Companion"}
        </button>
        <button
          type="button"
          className="rounded-lg border border-neutral-300 px-2 py-1 text-xs dark:border-neutral-700"
          onClick={() => agir("alternarSilenciado")}
        >
          {preferencias.silenciado ? "Ouvir avisos" : "Silenciar avisos"}
        </button>
        <button
          type="button"
          className="rounded-lg border border-neutral-300 px-2 py-1 text-xs dark:border-neutral-700"
          onClick={() => agir("restaurar")}
        >
          Restaurar
        </button>
        {naoPersistiu ? (
          // ***A AUSÊNCIA DECLARADA.*** *A escolha foi aplicada nesta sessão e **não** persistiu —
          // e o utilizador tem de saber isso, em vez de descobrir no próximo arranque.*
          <span className="text-xs text-amber-600 dark:text-amber-400" data-companion-nao-persistiu>
            A escolha não foi guardada (armazenamento indisponível). Vale só nesta sessão.
          </span>
        ) : null}
      </div>
    </div>
  );
}
