# HADES — ESTÁGIO 42: LIVE COMMERCE & AI SHOPPING AGENTS

**Complemento oficial anexado ao V2.2** — não é missão nova, não reinicia o projeto e **não revoga controles de
segurança**. O executor **não** precisou de vídeo ou imagem: o complemento chegou convertido em requisitos.
**Evidência de criação:** E-332 · **Estado:** `IN_PROGRESS` · **Release global:** `NOT_RELEASE_READY`.

---

## 42.0 — Inventário REAL dos módulos (medido antes de escrever)

Comando: `git ls-files 'internal/agent/*<área>*.go'` + `git grep` por símbolos.

| Área | Módulo existente | Reaproveitado por este estágio? |
| --- | --- | --- |
| mídia | `media.go` (+ `media_process_approval.go`) | **sim** — assets com aprovação; o módulo de live **não** guarda bytes |
| social | `company_social.go` | sim (publicação aprovada) |
| campanha | ingestão em `company_*` | sim |
| comércio | `commerce.go` (catálogo, estoque, pedidos, devoluções) | **sim** — SKU/pedido são a fonte da verdade |
| marketplace | `company_marketplace.go` | sim |
| inbox/canais | `inbox.go` | sim |
| aprovações | `mission_*`, `deployment_approval.go`, `media_process_approval.go` | **padrão reaproveitado** (aprovação humana identificada) |
| storage | `runtime_storage.go` | sim (referência de storage por tenant) |
| artefatos | `artifacts.go` | sim |
| **live commerce** | **AUSENTE** | **novo**: `live_commerce.go` — ciclo de vida da sessão e conformidade |
| **transcodificação/FFmpeg** | **AUSENTE** | 42.1: trabalho futuro |
| **Creator Studio (aba)** | não localizado como módulo próprio | 42.5: trabalho futuro |

**Não houve duplicação**: nenhum módulo de mídia, catálogo, inbox, marketplace ou aprovação foi recriado. O que faltava
era o **ciclo de vida** e as **regras de conformidade** — e é exatamente isso que este estágio acrescenta.

## Matriz por requisito (estados canônicos)

| Requisito (complemento) | Estado | Evidência / motivo |
| --- | --- | --- |
| Contratos canônicos versionados (LiveProject, MediaAsset, Playlist, AudioTrack, MixingPreset, LiveSession, CommerceEvent, ProductShowcase, OperatorApproval, ModerationEvent, métricas da sessão) | **IMPLEMENTADO_E_TESTADO** | `live_commerce.go` + `live_commerce_test.go` (E-332) |
| Estado de sessão com 8 estados e transições válidas | **IMPLEMENTADO_E_TESTADO** | grafo `LiveSessionTransitions`; teste de transição |
| **Aprovação humana obrigatória** antes de entrar no ar | **IMPLEMENTADO_E_TESTADO** | `ErrLiveApprovalRequired`; aprovação anônima recusada |
| **Encerramento automático no fim autorizado** (fail-closed) | **IMPLEMENTADO_E_TESTADO** | `SweepLiveSessions` + trava ao entrar depois do prazo |
| **Recuo diante de aviso da plataforma** (sem evasão) | **IMPLEMENTADO_E_TESTADO** | `RecordPlatformWarning` ⇒ `EXTERNALLY_BLOCKED` + incidente + revisão humana; **sem** caminho de volta automático |
| **Evento de carrinho ≠ venda confirmada** | **IMPLEMENTADO_E_TESTADO** | `IngestLiveCommerceEvent` + `ConfirmedSales`; receita só com `purchase_confirmed` **e** pedido |
| Deduplicação/idempotência de evento (replay) | **IMPLEMENTADO_E_TESTADO** | dedupe por (external_id, kind) **e** por pedido |
| Evento sem assinatura verificada é recusado | **IMPLEMENTADO_E_TESTADO** | `ErrLiveEventUnverified` |
| Isolamento por organização | **IMPLEMENTADO_E_TESTADO** | teste de não mistura |
| Destaque de produto como **sugestão** (sem automatizar clique) | **IMPLEMENTADO_E_TESTADO** | `PinnedByPlatform=false` |
| Upload/biblioteca/misturador com ducking e MP4 real | **AUSENTE** | 42.1 (FFmpeg em worker isolado) |
| Painel de saúde (frames, bitrate, CPU, memória) | **PARCIAL** | métricas existem na plataforma; falta a leitura da sessão em produção |
| Conector oficial TikTok Shop/LIVE (publicação, comentários, eventos) | **BLOCKED_BY_EXTERNAL_DEPENDENCY** | exige conta, sandbox, escopos e adesão aos Termos |
| Câmera virtual | **BLOCKED_BY_EXTERNAL_DEPENDENCY / desenho restrito** | só **componente desktop explícito**; **proibido** injetar no Chrome |
| UI (aba “Lives & Comércio”) e comandos `/live…` | **AUSENTE** | 42.5 — só com backend operacional e teste de autorização |
| Agentes especializados (7 papéis) | **AUSENTE** | 42.4 — herdam a interseção de capabilities, sem elevação |
| Auditoria das APIs oficiais (elegibilidade regional, escopos, quotas, Termos) | **PENDENTE** | próxima fatia; **não** se assume que extensão Chrome ⇒ permissão de API |

## Limites e proibições (registrados como requisito, não como observação)

1. **Nada de evasão.** Diante de aviso da plataforma: **pausar/encerrar**, registrar evidência e **escalar para
   humano**. Não existe (e não será criada) função para ocultar, contornar ou "proteger contra violações".
2. **Nada de spam, engajamento simulado, falsa escassez, falsa compra, perfil falso ou personificação.**
3. **Nada de injeção em Chrome** nem burla de proteções; mobile apenas **supervisão remota**.
4. **Sem promessa de transmissão oficial** enquanto não houver **conector homologado**; sem conector, entrega-se
   **plano e checklist assistido** para o operador.
5. **Sem "live infinita"**: `AuthorizedUntil` é **obrigatório** e no futuro; benchmarks próprios obrigatórios
   (o material de referência mostra preparação, **não** prova 24/7 nem conformidade).
6. **PII**: alertas de compra só quando adequados e seguros; nome de comprador não é exposto sem necessidade.

## Operacionalização e rollback

- **Windows/Linux**: o núcleo é Go puro e roda onde o servidor roda; captura/`câmera virtual` depende de componente
  desktop autorizado e é `BLOCKED_BY_EXTERNAL_DEPENDENCY`.
- **Android/iOS**: apenas **console de supervisão** (aprovar, pausar, encerrar, ver estado) — sem transmissão nativa
  nesta fatia.
- **Rollback**: como nada externo é publicado nesta fatia, desfazer = encerrar sessão (`ENDED`) e parar de usar o
  módulo; o estado fica em arquivo por organização e pode ser removido sem afetar mídia, catálogo ou pedidos.
- **Custos**: nenhum gasto é autorizado por este documento. Benchmark de CPU/memória será medido em fatia própria.

## Conclusão honesta do estágio

`COMPLETED_VERIFIED` **não** vem de UI, catálogo, adapter ou vídeo simulado. O que existe hoje é o **núcleo de
contratos e segurança**, provado por teste; **não** existe transmissão, **não** existe conector TikTok e **não** há
homologação com conta real. Onde a plataforma não oferecer a operação, o Hades **declara o limite** e oferece **fluxo
assistido** — sem contorno de restrições.
