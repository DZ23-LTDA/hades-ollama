// Download de arquivo gerado no cliente.
//
// Fica em módulo próprio (em vez de repetir em cada painel) e com GUARDA explícita: o ambiente de
// testes roda sem DOM, e um `document.createElement` solto ali vira `ReferenceError` no meio de um
// fluxo — falha confusa que parece defeito do produto. Melhor falhar alto e explicado.
export function triggerDownload(blob: Blob, filename: string): void {
  if (typeof document === "undefined" || typeof URL?.createObjectURL !== "function") {
    throw new Error("download requer um navegador: o ambiente atual não expõe document/URL");
  }
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
