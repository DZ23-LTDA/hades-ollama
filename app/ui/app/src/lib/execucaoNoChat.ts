// FATIA 4 / T4 — a superficie de conversa NAO pode fingir que executa.
//
// DEFEITO MEDIDO pelo dono: no chat, um pedido de execucao ("cria a pasta teste-hades com
// index.html") devolveu instrucoes de terminal com comandos INVENTADOS -- hades create-dir,
// hades create-file, hades list-files, hades open-browser -- e nenhum existe no binario.
//
// A conversa nao tem ferramentas ligadas. Em vez de deixar o modelo preencher o vazio a inventar
// passos, o produto responde o que e' verdade e aponta para a superficie que EXECUTA.

/** Intencoes que exigem ferramentas (criar/escrever/executar algo), nao apenas conversa. */
const VERBOS_DE_EXECUCAO = [
  "cria",
  "crie",
  "criar",
  "cria uma",
  "cria um",
  "faz",
  "faca",
  "faça",
  "gera",
  "gere",
  "gerar",
  "escreve",
  "escreva",
  "executa",
  "execute",
  "roda",
  "rode",
  "instala",
  "instale",
  "apaga",
  "delete",
  "move",
  "renomeia",
  "mkdir",
  "touch",
  "npm install",
  "git clone",
];


/**
 * Detecta pedido de execucao (nao conversa). Heuristica simples e explicita: verbo de execucao
 * OU alvo material no pedido. Falso positivo e' aceitavel (mostra o aviso); falso negativo deixa o
 * utilizador a falar com um agente que nao age.
 */
export function pedidoDeExecucao(texto: string): boolean {
  const t = (texto ?? "").toLowerCase();
  if (t.trim() === "") return false;
  // So' o VERBO dispara: um substantivo material sozinho ("o que e' um repositorio git?") e'
  // pergunta, nao pedido de execucao -- foi um teste que apanhou isso.
  return VERBOS_DE_EXECUCAO.some((v) => t.includes(v));
}

/**
 * Texto de recusa honesta. Regras:
 *  - NAO cita nenhum comando (nem real nem inventado) -- citar ja' seria ensinar um caminho falso;
 *  - diz o que e' verdade: esta superficie conversa, nao executa;
 *  - aponta a superficie que executa, sem inventar nomes de comandos.
 */
export function respostaSemFerramentas(): string {
  return (
    "Aqui eu só converso — esta caixa não tem ferramentas ligadas, por isso não consigo criar " +
    "pastas nem ficheiros por ti. Para eu executar, abre **Nova Tarefa** (Agent Center): é lá que " +
    "o agente corre com as ferramentas reais, e o resultado fica registado na missão."
  );
}

/** Comandos que o modelo inventou no chat do dono. Nenhum existe: nunca devem ser sugeridos. */
export const COMANDOS_INVENTADOS = [
  "create-dir",
  "create-file",
  "list-files",
  "open-browser",
];

/** true se o texto sugerir algum comando inventado (deve ser sempre false na resposta). */
export function citaComandoInventado(texto: string): boolean {
  const t = (texto ?? "").toLowerCase();
  return COMANDOS_INVENTADOS.some((c) => t.includes(c));
}
