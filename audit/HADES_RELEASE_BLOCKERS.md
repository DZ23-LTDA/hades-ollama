# HADES — BLOQUEIOS DE RELEASE

**Veredito:** `NOT_RELEASE_READY`. Este documento é a lista do que, hoje,
**impede** um release verificado — separando o que depende do projeto do que
depende de terceiros.

## 1. Gate final (estágio 38) — situação

| Requisito do gate final | Estado | Motivo |
| --- | --- | --- |
| Todos os estágios 1–38 em `COMPLETED_VERIFIED` | **NÃO** | 0 de 38 verificados |
| Auditoria completa das superfícies | **NÃO** | 12 superfícies `PENDENTE_DE_AUDITORIA` |
| CI obrigatório verde no head de release | **PARCIAL** | verde em `main` e no head dos PRs de missão |
| E2E de jornada crítica | **PARCIAL** | primeira execução coberta; Studio e demais fluxos não |
| Smoke nativo por SO | **NÃO** | só verificação de build de instalador |
| Release assinado | **NÃO** | sem certificado (B-06) |
| Isolamento por organização provado | **NÃO** | sem evidência anexada |
| Canais externos homologados | **NÃO** | B-02, B-03, B-04, B-05 |

## 2. Bloqueios internos (dependem do projeto)

| ID | Bloqueio | Impacto no release | Saída |
| --- | --- | --- | --- |
| R-01 | E2E do Studio (reorder + persistência) ausente | regressão silenciosa no builder | escrever spec Playwright (P0-2) |
| R-02 | Decisão pendente: falha de `GET /settings` cai no shell | onboarding pode ser pulado | decidir + fixar com teste (P0-4) |
| R-03 | Jornada E2E duplicada (#61 × #71) | manutenção divergente | triagem com autorização (P0-5) |
| R-04 | 12 superfícies sem auditoria | não se sabe o que existe | auditar com comando reproduzível |
| R-05 | 4 falhas ambientais de teste (B-09) | cobertura local incompleta | sanear host sem desativar teste |
| R-06 | Instaladores sem smoke nativo | release pode quebrar no usuário final | smoke por SO em runner nativo |
| R-07 | Benchmarks apenas documentais | afirmação competitiva sem prova | benchmark reproduzível (P3-3) |
| R-08 | Backlog datado com premissa errada do Studio | decisão de produto baseada em dado falso | corrigir `audit/BACKLOG_*` (P0-3) |

## 3. Bloqueios externos (dependem de terceiros)

| ID | Bloqueio | O que falta exatamente | Desbloqueia |
| --- | --- | --- | --- |
| X-01 | B-01 | autorização de merge para #56–#71 | entrega das correções |
| X-02 | B-02 | login real em Codex/Claude | paridade de harnesses (11) |
| X-03 | B-03 | tokens reais Telegram/WhatsApp | canais de IM (16, 28, 33, 34, 35) |
| X-04 | B-04 | provedor de telefonia/voz | telefonia e voz (29, 36, 37) |
| X-05 | B-05 | contas de marketplace/social/pagamento | negócio (24–27, 30, 31) |
| X-06 | B-06 | certificado de assinatura | release assinado (20, 38) |
| X-07 | B-07 | chaves de provedor de modelo | gateway, contexto, RAG, BI (4, 5, 8, 15, 23, 32) |
| X-08 | B-10 | credenciais por conector | 104 conectores hoje `APENAS_CATALOGO` |

## 4. O que **não** é bloqueio (não usar como desculpa)

- CI verde: não é bloqueio — está verde.
- Falta de tempo: não é bloqueio.
- Falta de acesso ao repositório: resolvido.
- Ausência de evidência sobre uma superfície: **é fila de auditoria**, não
  bloqueio externo.

## 5. Critério objetivo para sair de `NOT_RELEASE_READY`

1. R-01 … R-08 resolvidos ou explicitamente aceitos por decisão registrada.
2. Todas as superfícies auditadas com entrada no índice de evidências.
3. Gate final (estágio 38) reproduzido em ambiente limpo, por SO, com release
   assinado.
4. Nenhum item `BLOCKED_EXTERNAMENTE` pendente que seja **obrigatório** para a
   promessa de release — ou a promessa é reduzida para excluí-lo, de forma
   explícita e documentada.

Enquanto 1–4 não forem verdadeiros, o veredito permanece `NOT_RELEASE_READY`, e
qualquer comunicação de release deve dizer exatamente isso.
