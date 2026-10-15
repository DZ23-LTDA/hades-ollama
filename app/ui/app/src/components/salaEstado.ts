import { buscarMembrosDaSala, type Fetcher } from "../lib/salaApi";
import { estadoDeErro, estadoPronto, type EstadoDaSala } from "./SalaPainel";

/**
 * Complemento 48 / FASE 2 — A LÓGICA DO EFEITO, SEM REACT.
 *
 * **PORQUE É QUE ISTO EXISTE:** a `SalaLigada` tem efeito (`useEffect`) e o arnês de testes deste pacote **não tem
 * DOM** nem corre efeitos ⇒ *a parte que decide o que fazer com o resultado ficaria sem prova*. Aqui essa decisão
 * é uma **função pura e testável**; na tela sobra só a fiação (`useState`/`useEffect`/guarda de desmontagem).
 *
 * **A REGRA, UMA VEZ MAIS:** *`carregarEstado` NUNCA lança e NUNCA transforma falha em «pronta-vazia».*
 *   - sucesso (mesmo com zero membros) ⇒ `pronta`;
 *   - falha ⇒ `erro` **com o motivo**.
 * É esta função que impede o erro de chegar à tela disfarçado de «ainda não há membros».
 */

export type PedidoDaSala = {
  fetcher: Fetcher;
  projectID: string;
  roomID: string;
};

export async function carregarEstado(
  pedido: PedidoDaSala,
): Promise<EstadoDaSala> {
  try {
    const membros = await buscarMembrosDaSala(pedido);
    return estadoPronto(membros);
  } catch (causa) {
    const mensagem = causa instanceof Error ? causa.message : String(causa);
    return estadoDeErro(mensagem);
  }
}
