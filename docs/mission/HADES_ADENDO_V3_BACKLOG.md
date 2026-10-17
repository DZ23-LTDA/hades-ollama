# HADES — BACKLOG EXECUTÁVEL DO ADENDO V3

> **Origem:** adendo único de 2026-10-17 ao /goal HADES V3 (fases 0-30).
> **Estado:** `PARTIAL_IMPLEMENTED_NOT_E2E_VERIFIED` — o adendo **não** foi executado; foi **inventariado**.
> **Base medida:** HEAD `3f1df9aa7`, árvore limpa, 0 stashes, 966→967 evidências (`E-1174`).

## O achado que reordena o adendo

O adendo descreve 26 capacidades como se fossem novas. **Medido com `git grep -l -E <padrão> -- '*.go'`:
24 de 26 já têm código no HEAD.** O trabalho não é construir 26 módulos — é **provar 24** e **fechar 2**.

| capacidade | ficheiros `.go` | estado medido |
| --- | --- | --- |
| tenant | 178 | PRESENTE |
| design-studio / Builder | 161 | PRESENTE |
| sandbox | 78 | PRESENTE |
| artefactos | 74 | PRESENTE |
| auto-mode / multillm | 67 (120 no repo) | PRESENTE |
| voz | 33 | PRESENTE |
| pet / companion | 31 | PRESENTE |
| cli | 27 | PRESENTE |
| approvals | 24 | PRESENTE |
| worktree | 23 (69 no repo) | PRESENTE |
| eventos-sse | 23 | PRESENTE |
| mcp | 17 | PRESENTE |
| signing | 12 | PRESENTE |
| terminal | 10 | PRESENTE |
| skills | 9 | PRESENTE |
| orquestração (AgentOrchestrator) | 7 | PRESENTE |
| plugins | 5 | PRESENTE |
| browser-operator | 3 | PRESENTE |
| traces | 3 | PRESENTE |
| computer-use | 2 | PRESENTE |
| obsidian · memoria · protocolo | 1 · 1 · 1 | PRESENTE (mínimo) |
| **runtime-remoto (SSH)** | **0** | **AUSENTE** |
| **diff-review (ReviewComment)** | **0** | **AUSENTE** |

## Prioridade P0 — as duas lacunas reais

### P0-1 — `runtime-remoto` (Fase 6)
- **Medido:** 0 ficheiros `.go` com `RemoteRuntime`/`remoteRuntime`; 0 no repo inteiro.
- **O que o HADES já tem:** `sandbox` (78 ficheiros), `terminal` (10), `desktop.companion` (2).
- **Lacuna:** nenhuma interface comum de execução com provider `SSH` / `HADES Server persistente`.
- **Teste discriminante:** um provider SSH que responda a `capability probe` + `doctor` + `health` e
  falhe fechado quando o host não responde.
- **Critério de aceite:** `E2E` com dois processos reais, reconexão e sem duplicação de agente.

### P0-2 — `diff-review` (Fase 11)
- **Medido:** 0 ficheiros `.go` com `ReviewComment`; 0 no repo inteiro.
- **O que o HADES já tem:** `artefactos` (74), `approvals` (24), 97 testes de aprovação.
- **Lacuna:** comentários por linha/ficheiro devolvidos ao agente responsável.
- **Teste discriminante:** comentar num diff e provar que o comentário chega ao agente certo e que o
  próprio agente **não** pode marcar a sua revisão como aprovada.
- **Critério de aceite:** `E2E` com dois agentes e um comentário resolvido explicitamente.

## Prioridade P1 — provar o que já existe (as 24 presentes)

Cada uma exige o ciclo `código → teste discriminante → E2E real → evidência`, e **nenhuma** pode ser
marcada `COMPLETO_VERIFICADO` por existir. As mais próximas de prova, por já terem E2E medido:

| capacidade | prova que já existe | falta |
| --- | --- | --- |
| worktree | 69 ficheiros | E2E de 2 worktrees isolados com agentes paralelos |
| auto-mode | `E-1170` + 97 testes | E2E de missão real que escolhe topologia **e** modelo por passo |
| approvals | `E-1156`, `E-1170` | já provado incluindo entre processos |
| terminal | 10 ficheiros | splits, reconexão, reanexação a processo vivo |
| browser-operator | 3 ficheiros | E2E no navegador **pessoal** do utilizador (hoje é Playwright isolado) |
| obsidian | 1 ficheiro | contrato de sync, conflito, DLP, compactação (`Fase 17.1`) |

## Prioridade P2 — depende de infraestrutura externa

| fase | bloqueio real |
| --- | --- |
| 28 (Build/Signing/Publishing) | Expo/EAS revalidado, Android SDK, macOS/Xcode, service account Play |
| 27 (Computer Use total) | E2E físico por SO — só Windows disponível |
| 26 (Design Studio) | exportadores e renderers reais por tipo de artefacto |
| 14/mobile | APK + emulador; iOS exige macOS |

## O que este documento **não** é

Não é prova de implementação. É o **mapa** que impede duplicar 24 módulos e que fixa as 2 lacunas
reais. Cada linha só sai daqui com o seu teste e o seu ID de evidência.

*Um adendo que descreve 26 capacidades novas e encontra 24 já implementadas tem um problema de
inventário, não de engenharia. A FASE 0 existe exactamente para isso.*
