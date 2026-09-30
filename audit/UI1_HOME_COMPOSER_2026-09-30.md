# UI-1 — Composer da Home: modelo e anexos

**Data:** 2026-09-30 16:47 -03  
**Branch:** `recovery/ollama-full-snapshot`

## Implementado

- A Home oferece modo **Automático — grátis-primeiro**, enviando `auto/coding` ao backend para o roteador real.
- O modo **Manual** usa o `ModelPicker` controlado, filtrado por modelos selecionáveis/PASS, e envia o provider/modelo escolhido à criação real da missão.
- O composer aceita múltiplos arquivos pelo seletor, drag-and-drop e paste via `FileUpload`, respeita limite local de 10 MB, mostra chips removíveis e transporta os nomes dos anexos para o contexto da missão.
- O ModelPicker controlado não altera a preferência global do usuário; o fluxo de chat existente permanece compatível.

## Evidência

- Desktop 1440x900: `docs/evidencias/screen-ui1-home-desktop.png`
- Mobile 390x844: `docs/evidencias/screen-ui1-home-mobile.png`
- Auditoria: `docs/evidencias/browser-console-audit-ui1.json`
- Hashes distintos:
  - `f68f03d276373a4b21fa77fe66e7bc13acf3d9e2a6362abceca3ddce74bbd9d9` desktop
  - `ec3e454ab8e6bf1ae2b62c950cf6bb4311e4e514a1ec9c18dac572b5df6409ed` mobile

A captura foi executada contra Vite novo (`VITE_API_URL=http://127.0.0.1:11434`) e backend real local. Resultado: `console_errors: []`, `http_errors: []`, ambos os modos exibiram os controles automático/manual, o seletor de modelo e o anexo.
