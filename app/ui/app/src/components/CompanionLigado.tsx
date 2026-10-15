import { useEffect, useState } from "react";
import { CompanionMascote } from "./CompanionMascote";
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
    <CompanionMascote estado={estado.estado} razao={estado.razao} nome={nome} />
  );
}
