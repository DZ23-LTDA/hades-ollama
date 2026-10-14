# HADES ADAPTIVE / COWORK — MATRIZ DE LACUNAS E ADR

> **Pré-etapa obrigatória do complemento `HADES_MASTER_V2_2_COMPLEMENTO_GENSPARK_ESTAGIOS_43_A_47`**, executada
> **antes** de qualquer código novo, como o documento exige (`AUDITAR -> ENTENDER -> PLANEJAR -> IMPLEMENTAR`).
>
> **Regra desta matriz**: *não supor ausência pelo nome do arquivo, e não supor presença pela existência dele.* Cada
> linha traz **o que foi medido** e **como**. Onde eu não medi, a linha diz **`NAO_MEDIDO`** — *que não é "ok"*.

## 0. Estado medido da base

| item | medido |
| --- | --- |
| `main` | `4b3e21854` (*#436 e #437 mesclados e verificados*) |
| evidências | **426** (`E-001…E-426`, sem buraco — o `hades_mission_check.py` acusa buraco de numeração) |
| achados | 65 (`ADV-01…ADV-67`) |
| CI do `main` | `dz23-agentic-quality`, `class-a-plus-integrity`, `pr-gate` **verdes**; `test` **vermelho por lint** — causa medida em **E-423** |
| PR aberto | **#438** (*checker do relógio + E-423…E-426*) |

## 1. Auditoria de reuso — o que **JÁ EXISTE** (medido)

| capacidade do complemento | **já existe** | onde (medido) | classificação |
| --- | --- | --- | --- |
| Filtro por **capacidades exigidas** | ✅ | `multillm/router.go:29` `RequiredCapabilities`, `supports()`/`supportsPath()` em `:63` | **IMPLEMENTADO_E_TESTADO** |
| **`LOCAL_ONLY`** como restrição rígida | ✅ | `router.go:35` `LocalOnly bool`; bloqueio em `:70` (`!provider.AllowPrivate`) | **IMPLEMENTADO_E_TESTADO** |
| Ranking com **múltiplos candidatos** | ✅ | `router.go:52` `RouteCandidates() ([]RouteDecision, error)` | **IMPLEMENTADO_E_TESTADO** |
| **Justificativa inteligível** da escolha | ✅ | `router.go:134` `routeReason(model, health, request)` | **IMPLEMENTADO_E_TESTADO** |
| **Saúde** do provedor | ✅ | `router.go:9` `ProviderHealth` | **IMPLEMENTADO_E_TESTADO** |
| Estratégias **`FASTEST`** e **`BEST_QUALITY`** | ✅ | `router.go:23-24`; reweight em `:102-106` | **IMPLEMENTADO_E_TESTADO** |
| Estratégia **`BALANCED`** (*padrão*) | ✅ | `router.go:99` (*preferência vazia mantém o padrão balanceado*) | **IMPLEMENTADO_E_TESTADO** |
| Planejamento em passos | ✅ | `agent/planner.go` `Planner`, `RulePlanner.Plan()`, `UnconfiguredPlanner` | **IMPLEMENTADO_E_TESTADO** |
| **Roteamento dentro do plano** | ✅ | `planner.go:31` `RoutedPlannerResolver` | **IMPLEMENTADO_NAO_HOMOLOGADO** (*não medi o comportamento em missão real*) |
| Contabilização de **uso/custo** do plano | ✅ | `planner.go:43-48` `PlanUsageSink`, `PlanUsageReporter` | **IMPLEMENTADO_E_TESTADO** |
| **Capacidades declaradas** por modelo | ✅ | `agent/model_capabilities.go` (8 KB) | **IMPLEMENTADO_E_TESTADO** |
| **Capacidade verificada por PROBE real** | ✅✅ | `agent/model_capability_probe.go` (6 KB) + teste de 7 KB | **IMPLEMENTADO_E_TESTADO** |
| Política de capacidade (*allow/deny*) | ✅ | `agent/capability_policy.go` | **IMPLEMENTADO_E_TESTADO** |
| **Media** (providers, workspace seguro, loopback) | ✅ | `agent/media.go` 35 KB; `isLoopbackHost`, `safeMediaWorkspaceRoot`, `openMediaWorkspaceRoot` | **IMPLEMENTADO_NAO_HOMOLOGADO** (*FFmpeg ausente nesta máquina — E-* e *`BLOCKED_BY_EXTERNAL_DEPENDENCY`*) |
| **Pipeline** de media (*jobs, estágios*) | ✅ | `agent/media_pipeline.go` 16 KB + teste 11 KB | **IMPLEMENTADO_NAO_HOMOLOGADO** |
| **Aprovação** para processar media | ✅ | `agent/media_process_approval.go` | **IMPLEMENTADO_E_TESTADO** |
| Ferramenta de media exposta ao agente | ✅ | `agent/media_tool.go` | **IMPLEMENTADO_E_TESTADO** |
| **Artefatos** de missão | ✅ | `agent/artifacts.go`, `builder_generated_artifact_contract.go` | **PARCIAL** (*existe contrato de artefato do builder; **não** existe o modelo canônico do Estágio 47 — ver §2*) |
| **Skills** | ✅ | `agent/skill_activation.go` (+ testes de não-inferência entre tenants) | **IMPLEMENTADO_E_TESTADO** |
| **Studio Canvas** (UI multimodal) | ✅ | `app/ui/.../StudioCanvasPage.tsx` **72 KB** | **IMPLEMENTADO_NAO_HOMOLOGADO** |
| **Console agêntico** | ✅ | `AgenticConsole.tsx` 33 KB | **IMPLEMENTADO_NAO_HOMOLOGADO** |
| **Workspace de produto** (*tarefas/aprovações/artefatos*) | ✅ | `ProductWorkspacePage.tsx` 49 KB (*paridade do #437*) | **IMPLEMENTADO_E_TESTADO** (*E2E 9 specs/18 testes*) |
| **Visualizador de artefatos** | ✅ | `ArtifactsViewer.tsx` 9 KB | **PARCIAL** (*não faz versionamento/pesquisa — §2*) |
| **Seletor de modelo** | ✅ | `ModelPicker.tsx` 17 KB | **IMPLEMENTADO_E_TESTADO** |

## 2. Lacunas **concretas** por estágio

### Estágio 43 — Adaptive Intelligence Router
| lacuna | por que é lacuna (medido) | prioridade |
| --- | --- | --- |
| **Roteamento POR SUBTAREFA** | o router é chamado por **requisição** (`Route(request)`), e o planner produz `[]Step` — *não medi nenhum nó de plano carregando `RequiredCapabilities` própria nem escolha por nó* | **P0** |
| **`MIN_COST`** e **`MANUAL`** (*pin por tarefa*) | `RoutePreference` tem `auto`/`fastest`/`quality`; **não há** `min_cost` nem pin explícito | **P0** |
| **DAG com dependências tipadas** | `RulePlanner.Plan` devolve passos; *não medi dependências entre nós nem orçamento/permissões por nó* | **P1** |
| **Registry VERSIONADO** de capacidades | existem capacidades e probe, mas *não medi versão de registro nem proveniência por entrada* | **P1** |
| **Circuit breaker / retry por classificação de falha** | *não medi* contador de falhas por provedor com abertura de circuito | **P1** |
| **Persistência da decisão** (*candidatos, vencedor, motivo, custo*) | existe `routeReason` e `PlanUsageSink`, mas *não medi registro durável por decisão com candidatos perdedores* | **P1** |
| **Especialização real de modalidade** (*imagem→gerador de imagem*) | *não medi* recusa explícita de rotear vídeo para modelo textual | **P0** (*o documento é enfático: "não mandar pedido de vídeo para modelo textual e afirmar sucesso"*) |
| **Provedor ≠ harness** | *não medi* distinção entre API de modelo e sessão de harness de código no motor | **P0** |
| **Controle de gasto por usuário/organização/missão** | existe contabilização de plano; *não medi teto por escopo* | **P1** |

### Estágio 44 — Cowork & Spark
| lacuna | medido | prioridade |
| --- | --- | --- |
| **Spark como modo sobre a MESMA engine** | não existe; *é requisito explícito ("não criar segundo runtime")* | **P0** |
| **Comandos `/goal /spark /cowork /status /resume /cancel /approve /reject`** | *não medi nenhum deles* — e o documento proíbe comandos decorativos | **P0** |
| **Shell adaptativo leigo/especialista** | as páginas existem; *não medi alternância de modo* | **P1** |
| **Sincronização desktop↔mobile com idempotência offline** | *não medi* | **P1** |

### Estágio 45 — Multimodal Production Studio
| lacuna | medido | prioridade |
| --- | --- | --- |
| **MP4 real** | **ausente**: FFmpeg não está instalado nesta máquina (*`BLOCKED_BY_EXTERNAL_DEPENDENCY`*) | **BLOCKED** (*adapter declarado, homologação externa bloqueada — como o próprio documento autoriza*) |
| **STT/TTS** | *não medi* provider real de áudio | **P1** (*depende de credencial*) |
| **Documentos PDF/DOCX/XLSX/PPTX com fontes** | *não medi* geração real com validação de formato | **P0** |
| **Matriz por modalidade × engine** | **não existe** — é o entregável que une 43 e 45 | **P0** |

### Estágio 46 — Companion
| lacuna | medido | prioridade |
| --- | --- | --- |
| **Mascote original + assets próprios** | **AUSENTE** — não existe `HadesEmblem`/mascote medido | **P1** |
| **`prefers-reduced-motion` e leitor de tela** | *não medi* componente de mascote | **P1** |
| **Estados derivados de eventos verificados** | *não medi* — *é o requisito que impede "inventar agente trabalhando"* | **P0** |

### Estágio 47 — Universal Artifact Drive
| lacuna | medido | prioridade |
| --- | --- | --- |
| **Modelo canônico de artefato** (*id, tenant, user, project, mission, task, agent, MIME verificado, checksum, versão, ACL, retenção*) | existe `artifacts.go`; *não medi esse conjunto de campos* | **P0** |
| **Versionar / comparar / restaurar** | *não medi* | **P0** |
| **Busca por metadados e conteúdo, sem atravessar tenant** | *não medi* | **P0** |
| **Deduplicação sem mistura entre organizações** | *não medi* | **P0** |
| **Proteção contra path traversal / symlink / IDOR** | **existe padrão** em media (`safeMediaWorkspaceRoot`, `os.Root`) ⇒ *reutilizar, não recriar* | **P1** |

## 3. ADR — decisões de arquitetura desta pré-etapa

**ADR-001 — Reutilizar o `multillm.Registry` como **única** engine de seleção.**
*Contexto*: o Estágio 43 pede um "selection engine" com 6 estratégias; o `Registry` já tem capacidades, `LocalOnly`,
saúde, `RouteCandidates`, `routeReason`, e 2 das 6 preferências. *Decisão*: **estender** `RoutePreference` e o
`RouteRequest` — **não** criar um segundo router. *Justificativa*: o complemento proíbe explicitamente "um segundo
router em conflito". *Consequência*: as estratégias novas (`MIN_COST`, `MANUAL`) entram como `RoutePreference`, e o
**por-subtarefa** entra como um tipo de requisição (nó de plano), não como engine nova.

**ADR-002 — O nó de plano é o portador das exigências; o `Registry` continua sendo quem decide.**
*Contexto*: hoje o `Step` do planner não carrega `RequiredCapabilities`. *Decisão*: acrescentar os campos de exigência
ao **nó** e deixar a **decisão** no `Registry`. *Justificativa*: preserva a separação modelo/harness/orquestrador que
o complemento exige (§"Preservar a separação entre modelo, harness e orquestrador").

**ADR-003 — Nada de mascote antes da engine.**
*Contexto*: o Estágio 46 é visual, o 43 é o núcleo. *Decisão*: **Companion por último**, e derivado de **eventos
verificados** já existentes (*não de estado inventado*). *Justificativa*: o documento proíbe "avatar que finja
possuir… acesso que não tem".

**ADR-004 — A matriz de modalidade × engine é o artefato que governa 43 e 45 juntos.**
*Contexto*: ambos precisam saber "quem gera imagem/vídeo/áudio/documento". *Decisão*: uma **matriz única**, com
classificação por capacidade, alimentando o `Registry` — *não uma lista paralela na UI*. *Justificativa*: evita duas
fontes de verdade sobre o que o sistema sabe fazer.

**ADR-005 — Onde não há provider, **declarar bloqueio**, nunca simular.**
*Contexto*: FFmpeg ausente, sem credenciais de mídia. *Decisão*: o adapter é **implementado e testado por contrato**;
a homologação externa fica `BLOCKED_BY_EXTERNAL_DEPENDENCY`. *Justificativa*: o documento diz literalmente *"Não
simular geração de vídeo/imagem quando não houver provider; exibir bloqueio verificável"*.

## 4. Ordem de execução proposta (*PRs pequenos*)

1. **E-428**: `RoutePreference` ganha `MIN_COST` e `MANUAL` (*pin*) + testes de seleção e de pin. *Contraprova: remover a preferência tem de falhar.*
2. **E-429**: o nó de plano carrega exigências (`RequiredCapabilities`, modalidade, orçamento) e o `Registry` decide por nó. *Teste: missão com duas subtarefas heterogêneas escolhe executores **diferentes**.*
3. **E-430**: recusa explícita de modalidade incompatível (*"vídeo para modelo textual"*) — **falha verificável**, nunca sucesso falso.
4. **E-431**: matriz de modalidade × engine (*ADR-004*) como dado, alimentando o `Registry`.
5. **E-432+**: Artefato canônico (47) → STT/TTS e documentos (45) → Spark e slash commands (44) → Companion (46).

## 5. Limites honestos desta matriz

- As classificações **`IMPLEMENTADO_*`** valem para o que **medi** (*existência, assinaturas, testes presentes*);
  **não** homologam comportamento ponta a ponta ✗.
- As linhas **`NAO_MEDIDO`** são **a maioria das lacunas** ✗ — *e `NAO_MEDIDO` não é "ausente" nem "ok"*: é fila de
  auditoria.
- **Não** executei nenhuma missão de ponta a ponta dos Estágios 43–47 ✗ ⇒ **nenhuma alegação de paridade com o
  Genspark** é feita aqui ✓.
- A comparação com o Genspark é por **capacidades públicas observáveis**; **não** copiei UI, identidade nem código ✓.
