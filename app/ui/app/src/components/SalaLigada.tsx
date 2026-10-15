import { useEffect, useState } from "react";

import { SalaPainel, estadoCarregando, type EstadoDaSala } from "./SalaPainel";
import { carregarEstado } from "./salaEstado";
import type { Fetcher } from "../lib/salaApi";

/**
 * Complemento 48 / FASE 2 — A TELA LIGADA: efeito + cliente + painel.
 *
 * *O corpo desta função é deliberadamente pequeno. Toda a decisão que importa já está testada em três lugares*
 * (`salaApi` — falha não vira lista vazia; `SalaMembros` — a vista distingue os três casos; `SalaPainel` — o
 * estado é uma união discriminada). **Aqui só se liga os fios**, e ligar fios é o que menos deve pensar.
 *
 * DUAS REGRAS QUE ESTA TELA RESPEITA:
 *  1. **`catch` não engole**: uma falha vira `estadoDeErro` com o motivo — *nunca um `pronto` vazio*;
 *  2. **efeito com guarda de desmontagem**: *uma resposta que chega depois de a tela sair não pode escrever
 *     estado* (`vivo`), senão a última resposta ganha por acidente.
 *
 * LIMITE DECLARADO (medido): o arnês de testes deste pacote **não tem ambiente DOM** nem corre efeitos
 * (`renderToStaticMarkup`), por isso **esta unidade não tem teste local** — e não a declaro `TESTED`. *O que a
 * pode provar é um E2E de browser, que ainda não existe.*
 */

export type PropsDaTela = {
  projectID: string;
  roomID: string;
  /** Injectável: o E2E/uso real passa `fetch`; um teste de browser pode passar um duplo declarado. */
  fetcher: Fetcher;
};

export function SalaLigada({ projectID, roomID, fetcher }: PropsDaTela) {
  const [estado, setEstado] = useState<EstadoDaSala>(estadoCarregando());

  useEffect(() => {
    let vivo = true;
    setEstado(estadoCarregando());
    // A decisão vive em `carregarEstado` (pura e testada); *aqui só fica a fiação e a guarda de desmontagem.*
    void carregarEstado({ fetcher, projectID, roomID }).then((proximo) => {
      if (vivo) setEstado(proximo);
    });
    return () => {
      vivo = false;
    };
  }, [fetcher, projectID, roomID]);

  return <SalaPainel estado={estado} />;
}
