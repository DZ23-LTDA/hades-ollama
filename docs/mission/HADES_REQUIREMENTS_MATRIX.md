# HADES — MATRIZ ÚNICA DE REQUISITOS (rastreável por call site)

**Medido em**: 2026-10-15T01:45Z (2026-10-14T22:45 BRT) · **branch**: `fix/checker-relogio` ·
**head medido**: `6ef2aed65` · **PR**: #438 (aberto, `mergeable_state=blocked` por revisão humana) ·
**`origin/main`**: `4b3e2185422c24d5a05fd46a343953ad6f41d1e5`.

## 0. O que este documento é — e o que NÃO é

**Não é uma segunda fonte canônica.** As fontes canônicas continuam a ser
`HADES_MASTER_STATE.json` (estado), `HADES_STAGE_MAP.md` (pareamento dos 38 estágios com o prompt
master, gerado por `scripts/hades_stage_map.py` com `--check`) e `HADES_STAGE_REPORT.md`
(evidência por estágio). Este documento **acrescenta a coluna que faltava**: *o caminho completo,
com linha, do componente de UI até o efeito real* — porque **uma matriz de nomes de arquivos não é
auditoria** (§2 da meta).

**Regra de honestidade aplicada linha a linha**: só entra `TESTED`/`E2E_VERIFIED` com o comando e o
resultado medidos. Onde não há prova, o estado é o que a medição diz — `IMPLEMENTED_UNTESTED`,
`ADAPTER_ONLY` ou `EXTERNALLY_BLOCKED`.

### Enum de estado (§2 da meta)
`NOT_STARTED` · `ADAPTER_ONLY` · `IMPLEMENTED_UNTESTED` · `TESTED` · `E2E_VERIFIED` ·
`EXTERNALLY_BLOCKED`

*Distinção que se usa aqui*: **TESTED** = há teste que falha sem a correcção e passa com ela
(unitário ou de rota). **E2E_VERIFIED** = o efeito foi observado ponta a ponta por um caminho de
produto real (UI → HTTP → runtime → store), não por chamada directa de função.

## 1. Complementos transversais — verificados nesta sessão (com call site)

| ID | requisito (fonte) | UI | API | runtime/store | teste | resultado medido | estado | SHA |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| C-01 | Memória partilhada conforme permissões (compl. MEMÓRIA) | não tem UI própria (contrato) | 8 rotas de memória já existentes | `ContextStore.SearchMemoriesInScope` + `memory_isolation.go` | `memory_isolation_test.go` (herdado) | escopo **sem ator** exclui `private` por construção; reusado pelo MCP | `TESTED` | `22e696569` |
| C-02 | **Hades servindo MCP** (§3 MCPs) | não (protocolo para clientes externos) | `POST /api/agent/v1/mcp/rpc` (`server/agent_routes.go`, `mcpRPC`) | `agent.ServidorMCP` (`internal/agent/mcp_server.go`) → `ContextStore` | `mcp_server_test.go` (6) + `server/mcp_transport_test.go` (2) | 6/6 e 2/2 PASS; organização **vem da construção** e o projecto é verificado por construção; erros de protocolo em HTTP 200 | `TESTED` | `03eecbf69` |
| C-03 | Despertar coalescido ligado à fila (§6 Paperclip) | — | `POST /api/agent/v1/wakeups` (`wakeups`) | `Runtime.DespertarAgentes` → `EnqueueMission`; tique em `SupervisorScheduler.RunDue` | `despertar_coalescido_test.go` (7) + `despertar_das_missoes_test.go` (6) + `despertar_da_fila_test.go` (2) | 15 PASS; janela **fixa**; só «pronto a executar» vai à fila; idempotente por missão | `TESTED` | `143ff433f` |
| C-04 | Atribuição agente↔missão (`Mission.ActorID`/`AgentID`) | — | `CreateMissionRequest.AgentID` | `CreateMission` copia para o registo | idem C-03 (`TestDespertarAgentes…` prova que a atribuição sobrevive) | aditivo; o `ActorID` já vinha no pedido e **perdia-se** no registo | `TESTED` | `aa8f19bd7` |
| C-05 | Avaliação de desempenho (§10 Paperclip) | `PerformanceCard` montado em `AgenticConsole.tsx` | `GET /api/agent/v1/performance` | `agent.AvaliarDesempenho` (puro, sem LLM) | `desempenho_do_agente_test.go` (6) + `PerformanceCard.test.tsx` (4) | custo desconhecido **declarado**, não somado; `not_measured` **renderizado**; payload mau **não derruba a página** | `TESTED` | `146350696` |
| C-06 | Histórico/restauro de versões de skill (§9 Paperclip) | `SkillVersionsCard` ← `SkillCatalogPanel` | `GET /skill-versions/:id` · `POST /skill-versions/:id/restore/:version` | `ContextStore.arquivar`/`lerManifestoArquivado`/`registerSkill` | `skill_historico_test.go` (3) + `skill_restauro_test.go` (3) + `skill_versions_routes_test.go` (3) | arquivar **fail-closed**; restaurar **não herda confiança**; travessia recusada; outro inquilino vê **vazio** | `TESTED` | `4909af69e` |
| C-07 | Comparar revisões (§9 Paperclip) | botão «Comparar com a activa» no `SkillVersionsCard` | `GET /skill-versions/:id/compare?from&to` | `ContextStore.CompararVersoesDeSkill` | `skill_comparacao_test.go` (4) + rota (1) | a **versão activa não está no arquivo** (só as superadas) — defeito encontrado pelo teste; `trusted`/`enabled` não são conteúdo | `TESTED` | `f461ec86a` |

### 1.1 O caminho completo, provado (exemplo C-05, o mais curto)
```
AgenticConsole.tsx:395  <PerformanceCard/>
  → PerformanceCard.tsx:21  getPerformance()
  → agenticClient.ts:427    agentFetch("/api/agent/v1/performance")
  → server/agent_routes.go:725  group.GET("/performance", a.performance)
  → server/agent_routes.go:~2700 a.performance → missoesDoChamador(c)   [escopo de organização]
  → internal/agent/desempenho_do_agente.go  AvaliarDesempenho(missoes)
  → resposta {performance, scope}
PROVA UI: vitest 4/4 (inclui payload `{}` → não rebenta) · PROVA API: contrato de rotas no CI ·
PROVA NÚCLEO: 6/6 Go.  ***E2E do CARTÃO específico: NÃO corrido*** — logo `TESTED`, não `E2E_VERIFIED`.
```

## 2. O que os 38 estágios + extensões 39–42 + complementos 43–47 usam como fonte

Não se duplica aqui o pareamento: a tabela única é **`HADES_STAGE_MAP.md`** (gerada por
`scripts/hades_stage_map.py`, com `--check` no CI, e completude verificada por
`hades_mission_check.py`). O que a §2 pede além disso é a coluna de call site — preenchida na
secção 1 para os requisitos que **esta sessão mediu**, e a preencher incrementalmente para os
restantes (ver §4).

Complementos transversais já com documento próprio:
`HADES_PAPERCLIP_PARITY_MATRIX.md` · `HADES_CLOUDROOM_PARITY_MATRIX.md` ·
`HADES_ADAPTIVE_COWORK_GAP_MATRIX.md` · `HADES_MEMORY_CONTEXT_RELIABILITY.md`.

## 3. Testes de aceite (§11) — o que está medido e o que NÃO está

| teste | estado | evidência |
| --- | --- | --- |
| A) `go test ./...`, `-race`, lint, build de frontend, typecheck, unit, Playwright, CI | `TESTED` (parcial local + CI) | CI do PR `6ef2aed65`: **zero vermelhos** nos concluídos; local: `go test ./server/` ok (29,7 s), vitest 630/630, `tsc -b` 0, `BUNDLE_BUDGET=PASS`, bateria-espelho do job **28/28** |
| B) Missão de software real ponta a ponta | **`NOT_MEASURED` nesta fatia** | não corri uma missão real de software nesta sessão |
| C) Missão multimodal por provider real | **`EXTERNALLY_BLOCKED`** (sem provider configurado medido) | — |
| D) Missão multiagente por empresa | `IMPLEMENTED_UNTESTED` nesta fatia | código existe (`company_factory`), não medi aqui |
| E) Compactação + troca de modelo + reinício + handoff | `TESTED` (de sessões anteriores) — **não re-medido nesta fatia** | `HADES_MEMORY_CONTEXT_RELIABILITY.md` |
| F) Computer-use com aprovação e isolamento | `TESTED` (anterior) | — |
| G) Paperclip: heartbeat real, reserva atómica, versionamento | `TESTED` (C-03/C-04/C-06) | esta sessão |
| H) Cloudroom: dois executores de verdade | **`EXTERNALLY_BLOCKED`** | sem infraestrutura remota; simulação **não** conta como homologação |
| I) Instalação por utilizador leigo | `TESTED` (anterior) | `HADES_STAGE_21…` |
| J) Benchmark igualitário vs concorrentes | **`NOT_MEASURED`** | sem cenário reproduzível nem acesso às referências |

## 4. Próxima acção, por ordem

1. Preencher a coluna de call site para os **38 estágios** a partir de `HADES_STAGE_REPORT.md`,
   começando pelos que a matriz Paperclip/Cloudroom marcam como parciais.
2. Correr um **E2E do cartão de desempenho** para promover C-05 de `TESTED` a `E2E_VERIFIED`.
3. Actualizar `HADES_MASTER_STATE.json` com esta fatia (valores medidos, `measured_at` explícito).
4. Testes de aceite B–F: cada um precisa do seu cenário reproduzível; **não promovo nenhum sem ele**.
