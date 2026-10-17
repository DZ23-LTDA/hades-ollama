/**
 * Registo do Service Worker (`E-1061`).
 *
 * POR QUE O REGISTO E' SO' EM PRODUCAO.
 *
 * Em desenvolvimento o Vite serve os modulos um a um e faz HMR; um service worker a cachear
 * `/assets/*` **esconde as mudancas** e o sintoma passa a ser «o meu codigo nao aparece».
 * *Um service worker em dev e' um cache que mente sobre o que o programador acabou de escrever.*
 *
 * E POR QUE NAO E' SILENCIOSO.
 *
 * Se o registo falhar, o PWA nao fica instalavel -- e isso e' **informacao**, nao ruido. O
 * `warn` diz o motivo. Mas **nao** lanca: a aplicacao nao pode deixar de abrir porque o
 * service worker falhou. *Offline e' um extra; abrir e' o requisito.*
 */
export function registarServiceWorker(): void {
  if (!import.meta.env.PROD) return;
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) {
    console.warn("[hades-sw] serviceWorker indisponivel neste browser: PWA offline fica NAO_IMPLEMENTADO");
    return;
  }

  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/sw.js", { scope: "/" })
      .then((registo) => {
        // Uma actualizacao encontrada: deixa o SW novo assumir no proximo carregamento.
        registo.addEventListener("updatefound", () => {
          const novo = registo.installing;
          if (!novo) return;
          novo.addEventListener("statechange", () => {
            if (novo.state === "installed" && navigator.serviceWorker.controller) {
              console.info("[hades-sw] nova versao pronta; aplica-se no proximo carregamento");
            }
          });
        });
      })
      .catch((erro) => {
        console.warn("[hades-sw] registo falhou:", erro);
      });
  });
}
