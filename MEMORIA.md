---
projeto: ollama-mobile
status: em andamento
atualizado: 2026-09-29 10:18 -03:00
ultima_ia: Codex (Rover)
tags: [mobile, expo, ollama-full]
---

# Ollama mobile

## Objetivo
Entregar a faixa MOBILE do Ollama Full no worktree `D:\IA\Trabalhos\DZ23-LTDA\ollama-mobile`, branch `feat/mobile-parity`, sem integrar recovery. Código restrito a `apps/mobile-agentic`.

## Estado atual
- A inbox Expo lista missões de `GET /api/agent/v1/missions`, abre detalhe/timeline e lê lista e até 20 detalhes recentes em cache quando a rede falha. O formato antigo de cache com uma missão é migrado na próxima gravação.
- `npm install` em `apps/mobile-agentic` concluiu com zero vulnerabilidades reportadas. React e React DOM foram fixados em 19.0.0 após erro real de versões divergentes no navegador.
- Testes finais `npm run test:inbox`, `npm run test:policy` e `npm run typecheck` passaram; `git diff --check` sem erros após a correção de múltiplos detalhes.
- UI aberta por HTTP em `http://localhost:19007` com API local simulada no Playwright. Lista, detalhe e leitura offline observados. Erros de console durante a simulação offline são abortos `ERR_INTERNET_DISCONNECTED`; no fluxo online mockado houve zero erros, duas advertências de bibliotecas web.
- Evidências: `docs/evidencias/inbox-desktop.png`, `docs/evidencias/inbox-mobile.png`, `docs/evidencias/inbox-offline-mobile.png`, `docs/evidencias/detalhe-offline-mobile.png`.
- Regressão visual de cache: `docs/evidencias/cache-multiplas-missoes-mobile.png`. Em Expo Web (390×844), `m1` e `m2` foram abertos online; após abortar a rede e recarregar, `m1` reabriu offline com timeline. Online: zero erros de console; offline: três abortos de requisição esperados.
- Commit `3bd97fa2b11b02f8ae0bdfcdddcd033878e008af` enviado a `origin/feat/mobile-parity`; ref remota conferida igual. Revisão Codex+Qwen pendente. Não houve integração com recovery.

## Próximo passo
Codex+Qwen devem revisar o diff de `feat/mobile-parity` e as quatro evidências visuais. Não integrar recovery nesta faixa.

## Decisões
| Data | Decisão | Motivo | IA |
|---|---|---|---|
| 2026-09-29 | Cache da inbox com chave por servidor/organização; reutilizar cache de missão existente para o detalhe. | Isolar escopos e preservar leitura offline. | Codex (Rover) |
| 2026-09-29 | Respostas HTTP não usam cache como fallback. | Mostrar falhas reais do servidor. | Codex (Rover) |
| 2026-09-29 | Token web somente em memória; SecureStore apenas em Android/iOS. | SecureStore não funciona na web e token não deve ir ao AsyncStorage. | Codex (Rover) |
| 2026-09-29 | Cache de até 20 detalhes no mesmo escopo, com leitura do formato legado. | Evitar que consultar outra missão apague a leitura offline da primeira. | Codex (Rover) |

## Como rodar e testar
Em `D:\IA\Trabalhos\DZ23-LTDA\ollama-mobile\apps\mobile-agentic`: `npm install`, `npm run typecheck`, `npm run test:inbox`, `npm run test:policy`, `npm run start -- --web --host localhost --port 19007`.

## Arquivos importantes
- `apps/mobile-agentic/App.tsx` — interface e integração com API/cache.
- `apps/mobile-agentic/inbox.ts` — carregamento e fallback da inbox/detalhe.
- `apps/mobile-agentic/inbox.test.ts` — testes automatizados.

## Pendências e ideias
- [ ] Revisão Codex+Qwen da faixa mobile após push.
- [ ] Teste físico Android/iOS e push remoto dependem de dispositivo e contas.

## Histórico de sessões
### 2026-09-29 10:18 -03:00 — Codex (Rover)
- Feito: revisado o handoff; como a revisão Codex+Qwen ainda não estava registrada, corrigido o cache de detalhe para preservar até 20 missões no mesmo escopo e migrar o formato antigo.
- Evidência: `npm run typecheck`, `npm run test:inbox`, `npm run test:policy`, `git diff --check` e `docs/evidencias/cache-multiplas-missoes-mobile.png`.
- Parou em: publicar o ajuste somente em `feat/mobile-parity` e aguardar revisão Codex+Qwen, sem integrar recovery.

### 2026-09-29 10:06 -03:00 — Codex (Rover)
- Feito: implementada inbox com lista, detalhe e leitura offline, testes e execução visual desktop/celular.
- Evidência: comandos acima, `git diff --check` e quatro PNGs em `docs/evidencias`.
- Parou em: commit `3bd97fa2` publicado na branch mobile, aguardando revisão Codex+Qwen.
