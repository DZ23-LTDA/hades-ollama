import { createElement } from "react";

/**
 * Complemento 48 / FASE 2 — A SUPERFÍCIE DA SALA: quem está nela.
 *
 * **O QUE ESTE COMPONENTE RECUSA FAZER** (é o que o torna útil, não o que o torna bonito):
 *  1. **NÃO confunde «não consegui carregar» com «não há ninguém».** São coisas diferentes: a primeira é uma
 *     falha, a segunda é um facto. *Uma lista vazia silenciosa depois de um erro é a mentira mais fácil de
 *     escrever numa tela* — e é exactamente a classe de defeito que a matriz deste complemento persegue.
 *  2. **NÃO faz um agente passar por pessoa.** O tipo (`humano`/`agente`) vai **no texto**, para quem vê e para
 *     quem ouve — *um leitor de tela não recebe a cor*.
 *  3. **NÃO inventa botões.** Não há acção nenhuma aqui enquanto a acção não existir no backend; *botão morto é
 *     pior do que botão ausente*.
 *
 * TIPO ESPELHADO DO BACKEND: `MembroDaSala` corresponde ao `Participant` de `internal/agent/participantes.go`
 * (`id`, `tipo`, `nome`) — a superfície não inventa um vocabulário paralelo.
 */

export type TipoDeMembro = "humano" | "agente";

export type MembroDaSala = {
  id: string;
  tipo: TipoDeMembro;
  nome?: string;
};

export type PropsDaSala = {
  membros: MembroDaSala[];
  /** `true` enquanto a leitura está em curso: **não** é o mesmo que lista vazia. */
  carregando?: boolean;
  /** Mensagem de falha. Quando existe, ela tem precedência sobre tudo — menos sobre nada. */
  erro?: string;
};

/** Os textos vivem juntos para que o teste os possa citar sem os reescrever. */
export const TEXTOS_DA_SALA = {
  vazia: "Ainda não há membros nesta sala.",
  carregando: "A carregar os membros da sala…",
  etiquetaHumano: "pessoa",
  etiquetaAgente: "agente",
  titulo: "Membros da sala",
} as const;

/** `rotuloDoMembro` é o texto que IDENTIFICA o membro — nunca só um id cru quando há nome. */
export function rotuloDoMembro(membro: MembroDaSala): string {
  const nome = (membro.nome ?? "").trim();
  return nome === "" ? membro.id : `${nome} (${membro.id})`;
}

/** `etiquetaDoTipo` diz, em palavras, o que o membro é. *A cor não chega.* */
export function etiquetaDoTipo(tipo: TipoDeMembro): string {
  return tipo === "agente" ? TEXTOS_DA_SALA.etiquetaAgente : TEXTOS_DA_SALA.etiquetaHumano;
}

export function SalaMembros({ membros, carregando = false, erro = "" }: PropsDaSala) {
  const falha = erro.trim();

  // A ORDEM DAS DECISÕES É A GARANTIA: falha primeiro, depois carregamento, depois ausência, depois lista.
  if (falha !== "") {
    return createElement(
      "p",
      { className: "sala-membros-erro", role: "alert" },
      `Não foi possível carregar os membros: ${falha}`,
    );
  }

  if (carregando) {
    return createElement(
      "p",
      { className: "sala-membros-carregando", "aria-live": "polite" },
      TEXTOS_DA_SALA.carregando,
    );
  }

  if (membros.length === 0) {
    return createElement("p", { className: "sala-membros-vazia" }, TEXTOS_DA_SALA.vazia);
  }

  return createElement(
    "ul",
    { className: "sala-membros", "aria-label": TEXTOS_DA_SALA.titulo },
    membros.map((membro) =>
      createElement(
        "li",
        { key: membro.id, className: `sala-membro sala-membro-${membro.tipo}` },
        createElement("span", { className: "sala-membro-tipo" }, etiquetaDoTipo(membro.tipo)),
        " ",
        createElement("span", { className: "sala-membro-nome" }, rotuloDoMembro(membro)),
      ),
    ),
  );
}
