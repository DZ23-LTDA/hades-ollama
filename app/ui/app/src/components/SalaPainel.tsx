import { createElement } from "react";

import { SalaMembros, type MembroDaSala } from "./SalaMembros";

/**
 * Complemento 48 / FASE 2 — O PAINEL DA SALA: o estado da leitura, já resolvido.
 *
 * **PORQUE É QUE ISTO EXISTE SEPARADO DA TELA:** o arnês de testes deste projeto (`renderToStaticMarkup`) **não
 * corre efeitos** — logo uma tela que busca dados sozinha **não é testável** com ele. O que se pode provar é a
 * **vista pura**: dado um estado, que ecrã aparece. Fica aqui, e a tela real limita-se a produzir este estado.
 *
 * **A REGRA, OUTRA VEZ, AGORA NO ESTADO:** os três estados são **distintos por construção** — `carregando`,
 * `erro` e `pronta`. *Não existe um estado «vazio» separado de `pronta`: uma sala vazia é `pronta` com zero
 * membros, e uma falha é `erro`.* **É esta separação de tipos que impede o erro de virar «ainda não há
 * membros»** — o mesmo defeito que o cliente e o componente já recusam, um nível acima.
 */

export type EstadoDaSala =
  | { tipo: "carregando" }
  | { tipo: "erro"; mensagem: string }
  | { tipo: "pronta"; membros: MembroDaSala[] };

export const estadoCarregando = (): EstadoDaSala => ({ tipo: "carregando" });

/** `estadoDeErro` exige uma mensagem **não vazia**: *um erro sem motivo não se vê na tela, e um erro invisível
 * é indistinguível de «vazio».* */
export function estadoDeErro(mensagem: string): EstadoDaSala {
  const limpa = mensagem.trim();
  return { tipo: "erro", mensagem: limpa === "" ? "motivo não informado" : limpa };
}

export function estadoPronto(membros: MembroDaSala[]): EstadoDaSala {
  return { tipo: "pronta", membros };
}

/**
 * `estadoDoResultado` traduz o resultado do cliente para o estado da vista.
 *
 * *Recebe o resultado JÁ resolvido (ou o erro) — e é aqui que a decisão fica escrita em um só lugar:*
 * **sucesso ⇒ `pronta` (mesmo com zero membros); falha ⇒ `erro` (nunca `pronta` vazia).**
 */
export function estadoDoResultado(resultado: { membros: MembroDaSala[] } | { erro: unknown }): EstadoDaSala {
  if ("erro" in resultado) {
    const causa = resultado.erro;
    const mensagem = causa instanceof Error ? causa.message : String(causa);
    return estadoDeErro(mensagem);
  }
  return estadoPronto(resultado.membros);
}

export function SalaPainel({ estado }: { estado: EstadoDaSala }) {
  if (estado.tipo === "erro") {
    // Reaproveita a vista que já recusa confundir falha com ausência - *uma decisão, um lugar.*
    return createElement(SalaMembros, { membros: [], erro: estado.mensagem });
  }
  if (estado.tipo === "carregando") {
    return createElement(SalaMembros, { membros: [], carregando: true });
  }
  return createElement(SalaMembros, { membros: estado.membros });
}
