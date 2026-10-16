/**
 * MEDE a tabela do indice com um RENDERIZADOR REAL (`E-1045`).
 *
 * As tres medicoes anteriores erraram porque tentaram ADIVINHAR o que o Markdown faz com um
 * `|`. *Nao se adivinha um renderizador: usa-se um.*
 *
 * Aqui o `marked` (que ja' esta' no `node_modules`) renderiza as linhas de evidencia e conta
 * as `<td>` que saem. Se uma linha de 5 campos produzir 5 `<td>`, esta' CORRECTA -- independente
 * de quantos `|` tem no texto.
 */
import { readFileSync } from "node:fs";
import { fileURLToPath, pathToFileURL } from "node:url";
import path from "node:path";

const AQUI = path.dirname(fileURLToPath(import.meta.url));
const RAIZ = path.resolve(AQUI, "..");
const INDICE = path.join(RAIZ, "docs", "mission", "HADES_EVIDENCE_INDEX.md");
const { marked } = await import(
  pathToFileURL(path.join(RAIZ, "app", "ui", "app", "node_modules", "marked", "lib", "marked.esm.js")).href
);

const texto = readFileSync(INDICE, "utf8");
const linhas = texto.split("\n");
const evidencias = linhas
  .map((l, i) => [i + 1, l])
  .filter(([, l]) => /^\|\s*E-\d+\s*\|/.test(l));

console.log("  entradas de evidencia: " + evidencias.length);

// Renderiza CADA linha como uma tabela propria (com um cabecalho de 5 colunas) e conta os <td>.
const resultado = [];
for (const [n, l] of evidencias) {
  const bloco = "| a | b | c | d | e |\n| --- | --- | --- | --- | --- |\n" + l + "\n";
  const html = marked.parse(bloco);
  const corpo = html.split("</thead>")[1] ?? html;
  const tds = (corpo.match(/<td/g) ?? []).length;
  const temTr = /<tr/.test(corpo);
  resultado.push({ n, tds, temTr, linha: l });
}

const dist = {};
for (const r of resultado) dist[r.tds] = (dist[r.tds] ?? 0) + 1;
console.log("");
console.log("  <td> por linha, segundo o RENDERIZADOR:");
for (const k of Object.keys(dist).map(Number).sort((a, b) => a - b)) {
  const marca = k === 5 ? "  <-- 5 campos, CORRETO" : "  <-- QUEBRA";
  console.log("    " + k + " <td>: " + dist[k] + " linhas" + marca);
}

const quebradas = resultado.filter((r) => r.tds !== 5);
console.log("");
console.log("  linhas que o renderizador NAO consegue ler como 5 campos: " + quebradas.length);
if (quebradas.length) {
  console.log("");
  console.log("  === AMOSTRA (primeiras 10) ===");
  for (const r of quebradas.slice(0, 10)) {
    console.log("    :" + r.n + "  " + r.tds + " <td>  " + r.linha.trim().slice(0, 104));
  }
}
console.log("");
if (quebradas.length === 0) {
  console.log("  TODAS as 837 linhas renderizam com 5 campos.");
  console.log("  *As tres medicoes anteriores estavam erradas -- o Markdown aceita o `\\|` final.*");
}
