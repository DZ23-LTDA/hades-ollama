# HADES — TRABALHO PENDENTE PRIORIZADO

> Ordem de execução: **P0 → P1 → P2 → P3**. Nada aqui é declaração de conclusão;
> é backlog com critério de pronto.

## P0 — bloqueia o release e a honestidade do produto

### P0-1 · Checkpoints da missão V2.1 (PR #72)
- **Pronto quando:** os 11 artefatos existem, o JSON é válido, `PR gate` verde.
- **Estado:** **ENTREGUE** — PR #72 aberto com os 11 artefatos e **15/15 checks
  sem falha** (E-036); JSON válido (`stages=38`, `blockers=9`). Falta autorização
  de merge.

### P0-2 · Cobertura E2E do Studio — **ENTREGUE (PR #73)**
- **Correção:** o drag-and-drop do Studio **existe** (reorder por arraste em
  `StudioCanvasPage.tsx:975-988` e por botão/teclado `1012/1024`, ambos
  persistindo via `updateBuilderVisual`, com 5 testes unitários de
  `reorderById`). O gap real era **falta de spec Playwright que assegure o
  comportamento**, não falta de funcionalidade. Nunca reafirmar "DnD ausente".
- **Entregue:** `e2e/studioReorder.spec.ts` (3 testes) + `e2e/studio-builder-fixture.mjs`,
  provando ordem vinda do servidor, persistência do arraste, paridade pelo botão
  acessível e sobrevivência ao reinício. Verde no CI: **23/23 checks**, com os 3
  testes passando no job `Web E2E (shell smoke)` (E-034, E-035).
- **Restante do escopo:** preview/export, undo/redo e edição de propriedades sob
  E2E — em backlog (P1/P2).

### P0-3 · Corrigir o backlog histórico — **ENTREGUE (PR #73)**
- `audit/BACKLOG_LACUNAS_PRIORIZADO_20261009.md` afirmava "arraste real
  inexistente" no Studio. Corrigido com a evidência do probe (E-008/E-009), mais
  a contagem honesta dos 6 itens do roadmap e a ordem recomendada.

### P0-4 · Decidir o gate de onboarding quando `GET /settings` falha
- **Situação:** `routes/index.tsx` engole o erro de `getSettings` e cai no shell
  sem redirecionar ao onboarding ⇒ usuário novo sem backend vê a aplicação e não
  o onboarding. Pode ser comportamento desejado (modo local) ou defeito.
- **Critério de pronto:** decisão documentada + teste que fixe o comportamento
  escolhido (hoje há apenas o comportamento implícito).

### P0-5 · Jornada de primeira execução duplicada
- PR **#61** (`test/e2e-jornada-primeira-execucao`, `bb45d7944`) e PR **#71**
  (`feat/e2e-primeira-execucao`, `3b5459ef5`) cobrem a mesma jornada.
- **Critério de pronto:** decidir qual supersede o outro e fechar o redundante
  **somente com autorização explícita**. Recomendação: manter #71 (já gateado no
  CI) e fechar #61 após o merge de #71.

## P1 — paridade competitiva (TypingMind / AutoClaw / Manus)

### P1-1 · Saída de ferramenta visível na linha do tempo
- PR #69 (`feat/saida-ferramenta-no-chat`, `525856a75`) está verde. Falta
  autorização de merge. Opcional: renderizar o excerto de `output` em
  `AgenticSplitShell.tsx` (a linha do tempo já mostra `payload` genérico).

### P1-2 · Paridade de chat TypingMind
- PR #56 (chat) e #58 (biblioteca de prompts) abertos. Próximo: comparação lado
  a lado multi-modelo e biblioteca de prompts usável de ponta a ponta.

### P1-3 · Canais de IM (Telegram/WhatsApp/inbox)
- PRs #63 (Telegram) e #65 (guia de conexão) abertos; PR #57 traz o comparativo
  AutoClaw. **Bloqueio real:** homologação com token exige B-03. Manter a
  classificação `APENAS_ADAPTER` até haver chamada real.

### P1-4 · Harness CLI governado
- PR #62 (`afea7e726`) é **branch-local**: `harness.cli` não está em `main`.
  Próximo: merge e prova de que a aprovação obrigatória (`RequiresApproval:
  true`, `Risk: RiskWrite`) é exigida de fato, com teste reproduzível.

### P1-5 · Catálogo honesto de conectores
- 104/122 exigem operador. Promover conectores **um por um**, com evidência de
  chamada real e mantendo `Auth` não-vazio. A metade "promoção automática de
  status" para linhas só-chave é **inviável** (`BLOCKED_BY_EXTERNAL_DEPENDENCY`).

## P2 — engenharia e infraestrutura

### P2-1 · Workflow Engine
- Provar execução real de schedule persistido (não só validação), persistir
  desfecho por passo, e implementar ou **recusar de forma fechada**
  `action.http`/`action.connector`; opcional `condition.if` com seletor de
  `expect` no editor de fluxo (PRs #67/#68).

### P2-2 · Aviso de depreciação do Node 20 no CI
- `actions/checkout@11d5960a…` e `actions/setup-node@49933ea5…` (v4 com SHA
  fixo) emitem `Node.js 20 is deprecated`. PR dedicada subindo os `uses:`
  fixados por SHA para v5 nos 15 workflows.

### P2-3 · Replay curado do upstream
- Branch `chore/upstream-YYYYMMDD` a partir de `main`; replay linear temático
  sobre `upstream/main` (73 commits, 39 arquivos em conflito, 15 conflitos
  reais). 4 gates obrigatórios + smoke Chromium antes de abrir PR.

### P2-4 · Falhas ambientais de teste (B-09)
- Sanear o host de teste (Python + `sh`, caminho curto) **sem** desativar teste.

### P2-5 · Prettier não governado na raiz
- `.prettierrc` existe só em `app/ui/app`; nenhum workflow roda
  `prettier:check`. Decidir: adotar na raiz ou declarar formalmente fora de
  escopo.

## P3 — release verificado e arquitetura

### P3-1 · Release assinado por SO
- Certificado (B-06), smoke nativo por SO, artefatos assinados publicados.

### P3-2 · Isolamento por organização (multitenancy)
- Decisão arquitetural + teste de que uma organização não lê dados de outra.

### P3-3 · Benchmarks com metodologia auditável
- Substituir o comparativo documental (`audit/COMPARATIVO_TYPINGMIND_20261009.md`,
  `docs/RELATORIO_BENCHMARK_COMPETITIVO_HADES_2026-10-04.txt`) por benchmarks
  reproduzíveis com versão, ambiente e comando.

### P3-4 · Estágios 5, 8, 15 e 34–37
- Auditar existência real (nada foi inspecionado) antes de qualquer
  classificação; depois avançar conforme a dependência externa permitir.
