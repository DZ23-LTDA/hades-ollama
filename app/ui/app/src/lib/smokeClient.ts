/**
 * FASE 28.7/28.8: cliente da rota `build.smoke`.
 *
 * O QUE ESTE FICHEIRO LIGA
 *
 * A E-1273 criou a rota que **executa** um binario e provou-a com 6 testes. Mas nenhum painel a
 * consumia -- e o defeito recorrente desta missao ja' se reabriu onze vezes assim:
 *
 *	*Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala
 *	com o outro.*
 *
 * O QUE ESTE CLIENTE NAO TEM, POR DESENHO
 *
 * **Nao tem parametro para a raiz.** A raiz vem de `HADES_ARTEFACTOS` no ambiente do servidor. Um
 * cliente que escolhesse a raiz escolheria o que executar:
 *
 *	*Um cliente que escolhe a raiz dos binarios e' um cliente que decide o que corre.*
 *
 * O QUE A RESPOSTA TRAZ, E QUE O PAINEL TEM DE MOSTRAR
 *
 * `sem_sandbox: true` e o `aviso_de_sandbox`. Executar um binario sem isolamento e' uma capacidade
 * real com um risco real, e o resultado tem de o dizer -- nao a documentacao:
 *
 *	*Um aviso que fica na documentacao nao chega a quem correu o binario.*
 */

import { agentFetch } from "@/lib/agenticClient";

/** Pedido de `build.smoke`. */
export interface PedidoDeSmoke {
  caminho: string;
  argumentos: string[];
  saida_esperada?: string;
  codigo_esperado?: number;
  timeout_ms?: number;
}

/** Resultado de `build.smoke`. */
export interface ResultadoDeSmoke {
  executou: boolean;
  aprovado: boolean;
  codigo_de_saida: number;
  duracao_ms: number;
  saida: string;
  saida_truncada: boolean;
  timeout: boolean;
  explicacao: string;
  sem_sandbox: boolean;
  aviso_de_sandbox: string;
  raiz_autorizada: boolean;
}

/**
 * Executa o artefacto no servidor.
 *
 * O caminho tem de estar dentro da raiz autorizada do servidor; este cliente nao sabe (nem pode
 * saber) qual e'.
 */
export async function executarSmoke(pedido: PedidoDeSmoke): Promise<ResultadoDeSmoke> {
  return agentFetch<ResultadoDeSmoke>("/api/agent/v1/build/smoke", {
    method: "POST",
    body: JSON.stringify(pedido),
  });
}

/**
 * Separa os argumentos de uma linha de texto.
 *
 * Aceita espacos e aspas duplas simples: `--version` e' um argumento, `"a b"` e' outro. **Nao**
 * interpreta escapes de shell, porque nao ha' shell do outro lado:
 *
 *	*Um cliente que imita o shell e' um cliente que ensina o utilizador a escrever shell.*
 */
export function separarArgumentos(linha: string): string[] {
  const partes: string[] = [];
  let atual = "";
  let dentroDeAspas = false;
  for (const caracter of linha) {
    if (caracter === '"') {
      dentroDeAspas = !dentroDeAspas;
      continue;
    }
    if (caracter === " " && !dentroDeAspas) {
      if (atual !== "") {
        partes.push(atual);
        atual = "";
      }
      continue;
    }
    atual += caracter;
  }
  if (atual !== "") {
    partes.push(atual);
  }
  return partes;
}

/**
 * Resume o resultado numa frase.
 *
 * Nunca diz "seguro": diz o que foi observado. Um smoke aprovado num binario sem sandbox nao e' um
 * binario seguro -- e' um binario que arrancou.
 */
export function resumirSmoke(resultado: ResultadoDeSmoke): string {
  if (resultado.timeout) {
    return `O artefacto NAO terminou em ${resultado.duracao_ms} ms: um processo que nao termina nao ` +
      `e' um processo que passou.`;
  }
  if (!resultado.executou) {
    return `O artefacto nao chegou a arrancar: ${resultado.explicacao}`;
  }
  const veredicto = resultado.aprovado ? "aprovado" : "reprovado";
  return `Smoke ${veredicto}: exit code ${resultado.codigo_de_saida}, ${resultado.duracao_ms} ms. ` +
    resultado.explicacao;
}

/**
 * Diz se o resultado permite afirmar que o artefacto esta' *funcional*.
 *
 * Um smoke aprovado prova que o binario arrancou e respondeu -- nao prova que faz o que promete.
 *
 *	*Um binario que arranca nao e' um binario que funciona: e' um binario que arranca.*
 */
export function provaDeFuncionamento(resultado: ResultadoDeSmoke): string {
  if (!resultado.aprovado) {
    return "nao provado: o smoke reprovou";
  }
  return "provado que arranca e responde; NAO provado que faz o que promete";
}
