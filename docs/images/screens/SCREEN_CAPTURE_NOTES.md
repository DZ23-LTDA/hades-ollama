# Notas das capturas

## Captura atual — 2026-10-21

Recapturadas nesta rodada as 10 telas da série `class-a-plus-*.png` com Chromium em viewport de
1440x900, através de `app/ui/app/scripts/capture-parity-screens.mjs` (Vite dev em `127.0.0.1:4175`,
proxy `/api` para o servidor Hades local em `127.0.0.1:11435`, com `OLLAMA_AGENT_STORE` e
`OLLAMA_AGENT_ROOT` dentro de `_tmp/`). `npm run screens:verify` passa com **PASS (10 files)**.

Para esta captura foram corrigidos dois defeitos reais encontrados pelo próprio procedimento:

1. **`SkillSelection` serializava `"selected": null`** quando a lista estava vazia (slice nil em Go).
   O painel fazia `.map()` sobre `null` e a página `/skills` registava
   `TypeError: Cannot read properties of null (reading 'map')`. Corrigido com `MarshalJSON` no tipo
   (contrato: lista, mesmo vazia) e travado por `TestSkillSelectionVaziaSerializaComoLista`.
2. **O manifesto e o validador tinham contratos diferentes**: o script escrevia `captures[]` com
   caminho absoluto e o validador lê `files[]` com caminho relativo. O script passou a emitir os dois
   (`files` = contrato verificado).

`POST /api/v1/settings` devolve **501 por decisão** neste servidor local (sem persistência), travado
por `server/desktop_local_settings_ratchet_test.go`; a ferramenta de captura passou a classificar esse
501 (e a mensagem correspondente na consola) como **esperado**, em vez de tratar o servidor local
como página avariada.

Os arquivos `class-a-plus-home.png`, `class-a-plus-projects.png`, `class-a-plus-library.png`, `class-a-plus-scheduled.png`, `class-a-plus-skills.png`, `class-a-plus-plugins.png`, `class-a-plus-tasks.png`, `class-a-plus-agentic.png`, `class-a-plus-settings.png` e `class-a-plus-company.png` foram recapturados nesta rodada com Chromium em viewport de 1440×900. O arquivo compatível `settings.png` aponta para a mesma captura funcional de Settings.

A captura foi feita com `app/ui/app/scripts/capture-parity-screens.mjs`, usando o Vite dev em `127.0.0.1:4173` e o servidor Ollama local em `127.0.0.1:3001`. O uso do Vite dev é intencional: o cliente agentic usa a URL de desenvolvimento para consultar a API local; o preview estático sem proxy pode mostrar uma tela de erro e não deve ser usado como fonte dessas imagens.

O manifesto versionado [`class-a-plus-capture-manifest.json`](class-a-plus-capture-manifest.json) registra o checkpoint de desenvolvimento, viewport, tamanho e SHA-256 de cada PNG. `npm run screens:verify` falha quando um arquivo é alterado sem atualizar o manifesto; esse gate verifica proveniência do arquivo, não transforma screenshot em evidência de produção.

As telas mostram navegação, Agentic Console, projetos, biblioteca, schedules, skills, plugins, tarefas, Settings e Company OS com estado local/sandbox. Alguns dados demonstrativos foram criados no runtime local; aprovações permanecem visíveis como pendentes quando aplicável. Nenhuma captura representa provider externo conectado, credencial válida, login enterprise, dispositivo físico, deploy, marketplace, loja ou harness remoto.

## Arquivos históricos

`agentic-console.png` permanece uma captura histórica; `settings.png` é mantido como alias compatível da captura atual `class-a-plus-settings.png`. O README e o guia usam a série `class-a-plus-*.png` atualizada acima. Capturas históricas não devem ser interpretadas como prova de provider externo, conta conectada ou release homologado.

## Regra de proveniência

Screenshots reais demonstram a implementação e o estado observado no ambiente em que foram capturados. Eles não substituem testes de integração, smoke com contas externas, staging distribuído, validação em dispositivos ou evidência de release. Mockups conceituais continuam documentados separadamente em `docs/images/mockups/MOCKUP_NOTES.md`.
