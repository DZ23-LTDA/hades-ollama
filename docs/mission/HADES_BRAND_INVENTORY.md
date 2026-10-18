# HADES - INVENTARIO DE MARCA E EXCECOES DECLARADAS (gerado, nao editar a mao)

> Gerado por `python scripts/hades_brand_inventory.py`. O `--check` falha se divergir.
> Cumpre o **§FASE-5, itens 1, 2 e 3**: inventario das strings visiveis, migracao da
> superficie visivel e **documentacao de cada excecao de compatibilidade**.

## 1. As cinco strings, como o §FASE-5 item 1 as nomeia

> **Fonte das contagens (E-997)**: os ficheiros sao lidos **do commit (`HEAD`)**, e nao do disco.
> *Medido no run **38928255782**, onde este passo falhou*: o disco tinha **13 ficheiros
> modificados** com **24** ocorrencias de marca que **nao existem no commit**, e o inventario
> divergia entre a minha maquina e o CI. => *um instrumento que mede o disco mede **tambem o
> trabalho nao commitado** -- e publica um numero que so' e' verdadeiro onde foi gerado.*
> Origens medidas neste snapshot: `HEAD (sujo no disco)`: 13, `disco (limpo)`: 2649.

| String | Ocorrencias |
| --- | --- |
| **Hades/HADES** (canonica) | **1335** |
| DZ23 Agentic | 256 |
| Ollama Full | 177 |
| Ollama Classe A+ | 99 |
| Ollama DZ23 | 59 |

**Legadas somadas:** 591 ocorrencias
(262 em ficheiros **migraveis**, 329 em ficheiros de **excecao**).

*O total **nao** decide: os mesmos numeros distribuem-se por sitios de tipos muito
diferentes, e e' o sitio que decide.* Um ``README`` migra; um `bundle ID` nao.

## 2. A regra que separa os dois casos

| Categoria | Quem le' o texto | Acao |
| --- | --- | --- |
| **MIGRAVEL** | o **utilizador** (titulos, prosa, `README`) | migrar para HADES |
| **EXCECAO** | **outro sistema** (instalador, env var, credencial, bundle id) | manter, e dizer por que |
| **HISTORICO** | **ninguem em execucao**: registo datado | preservar como esta' |

*As tres categorias sao diferentes, e tratar a terceira como a segunda seria pior que nao
fazer nada: reescrever um registo historico produz um documento **coerente com a marca de
hoje** e **falso sobre o que foi medido**.*

## 3. Excecoes declaradas (o §FASE-5 item 3)

| Caminho | Tipo | Por que se mantem | Desbloqueio |
| --- | --- | --- | --- |
| `app/ollama.iss` | **instalador** | Nome do artefacto do instalador (Inno Setup). E' o ficheiro que o utilizador procura e que o desinstalador usa para se remover. | So' com migracao de artefacto + registo de desinstalacao da versao antiga. |
| `scripts/dz23-configure.ps1` | **env var** | Escreve variaveis de ambiente. Um nome novo nao e' lido por configuracoes existentes. | Aceitar os dois nomes em leitura durante uma janela de transicao. |
| `internal/multillm/credentials.go` | **credencial** | Codigo que resolve chaves de credencial por nome. Renomear muda a chave armazenada. | Alias de leitura + migracao do armazenamento, com rollback. |
| `app/ui/connectors_test.go` | **teste** | Teste que depende do codigo de conetores; o texto legado esta' no dado de teste, nao na interface. | Acopla-se a' renomeacao do codigo que ele testa. |
| `docs/evidencias/` | **historico** | Artefactos de medicao (auditorias de consola). Sao **registos**, nao documentos de marca. | Nao se desbloqueia: preserva-se. |
| `docs/mission/HADES_EVIDENCE_INDEX.md` | **historico** | Indice de evidencias: cita os nomes que os artefactos **tinham** quando foram medidos. Reescreve-lo tornaria o indice **falso** sobre o que foi medido. | Nao se desbloqueia: a evidencia nao se reescreve (§FASE-7 item 1). |
| `audit/` | **historico** | Fonte historica datada (checkpoints append-only, handoffs com data no nome). | Nao se desbloqueia: preserva-se. |

## 4. Ficheiros migraveis (apenas os com legado)

| Ficheiro | Legadas | Categoria |
| --- | --- | --- |
| `MEMORIA.md` | 49 | migravel |
| `docs/agentic/READINESS_2026-09-22.md` | 18 | migravel |
| `scripts/hades_brand_inventory.py` | 18 | migravel |
| `docs/agentic/PRODUCT_TREE.md` | 10 | migravel |
| `docs/mission/HADES_NEXT_ACTION.md` | 10 | migravel |
| `docs/OLLAMA_FULL_PRODUCT_VISION.md` | 9 | migravel |
| `docs/agentic/PARITY_MATRIX.md` | 9 | migravel |
| `docs/agentic/INTEGRATIONS.md` | 8 | migravel |
| `docs/mission/HADES_PENDING_WORK.md` | 8 | migravel |
| `CHANGELOG.md` | 7 | migravel |
| `apps/mobile-agentic/MIGRACAO_MARCA.md` | 6 | migravel |
| `docs/RELEASE_READINESS.md` | 6 | migravel |
| `docs/agentic/HARNESSROUTER.md` | 6 | migravel |
| `docs/mission/HADES_MASTER_STATE.json` | 6 | migravel |
| `docs/agentic/ROADMAP.md` | 5 | migravel |
| `docs/mission/HADES_HANDOFF.md` | 5 | migravel |
| `docs/agentic/XAI_GROK.md` | 4 | migravel |
| `scripts/install.ps1` | 4 | migravel |
| `UPSTREAM_POLICY.md` | 3 | migravel |
| `docs/agentic/ARCHITECTURE.md` | 3 | migravel |
| `docs/agentic/COMPOSIO.md` | 3 | migravel |
| `docs/agentic/HARNESS_COMPARISON_SYNTHESIS.md` | 3 | migravel |
| `docs/dz23-multi-provider.md` | 3 | migravel |
| `scripts/classify-ci-surfaces.py` | 3 | migravel |
| `scripts/classify_ci_surfaces_test.py` | 3 | migravel |
| `app/ui/app/e2e/shell.spec.ts` | 2 | migravel |
| `apps/mobile-agentic/package-lock.json` | 2 | migravel |
| `docs/BRANCH_PROTECTION.md` | 2 | migravel |
| `docs/WEB_DEPENDENCY_AUDIT.md` | 2 | migravel |
| `docs/agentic/COMPANY_OS.md` | 2 | migravel |
| `docs/agentic/DESKTOP_COMMANDER_REMOTE.md` | 2 | migravel |
| `docs/agentic/PHASE5_DELIVERY.md` | 2 | migravel |
| `docs/mission/HADES_FINAL_REPORT.md` | 2 | migravel |
| `internal/agent/builder_studio_test.go` | 2 | migravel |
| `server/agent_routes.go` | 2 | migravel |
| `.github/workflows/dz23-agentic-quality.yaml` | 1 | migravel |
| `.github/workflows/pr-gate.yaml` | 1 | migravel |
| `README.md` | 1 | migravel |
| `app/updater/updater.go` | 1 | migravel |
| `app/updater/updater_live_test.go` | 1 | migravel |
| `docs/CLASS_A_PLUS_GUIDE.md` | 1 | migravel |
| `docs/LARGE_FILES.md` | 1 | migravel |
| `docs/agentic/EVALUATION.md` | 1 | migravel |
| `docs/agentic/GATEWAY_ROTACAO.md` | 1 | migravel |
| `docs/agentic/PHASE3_DELIVERY.md` | 1 | migravel |
| `docs/agentic/PHASE9_DELIVERY.md` | 1 | migravel |
| `docs/agentic/POLSIA_COMPARISON.md` | 1 | migravel |
| `docs/audit/LIMITACOES.md` | 1 | migravel |
| `docs/decisions/ADR-001-REMOTE-CLOUD-PROVIDERS.md` | 1 | migravel |
| `docs/decisions/ADR-004-POSTGRES-TENANT-RLS-MIGRATION.md` | 1 | migravel |
| `docs/dz23-parity-backlog.md` | 1 | migravel |
| `docs/images/mockups/MOCKUP_NOTES.md` | 1 | migravel |
| `docs/mission/HADES_ADAPTIVE_COWORK_GAP_MATRIX.md` | 1 | migravel |
| `docs/mission/HADES_FAILURE_INDEX.md` | 1 | migravel |
| `docs/mission/HADES_PLATFORM_MATRIX.md` | 1 | migravel |
| `docs/mission/HADES_STAGE_19_ONBOARDING_E2E.md` | 1 | migravel |
| `docs/superpowers/plans/2026-10-03-custom-ai-providers.md` | 1 | migravel |
| `internal/agent/company_test.go` | 1 | migravel |
| `internal/agent/git_repo_test.go` | 1 | migravel |
| `internal/agent/media.go` | 1 | migravel |
| `internal/agent/runtime.go` | 1 | migravel |
| `internal/agent/tools_test.go` | 1 | migravel |
| `internal/agent/workspace_snapshot_test.go` | 1 | migravel |
| `scripts/hades-quickstart.ps1` | 1 | migravel |
| `scripts/hades_ci_gate_check_test.py` | 1 | migravel |
| `scripts/hades_linux_package.py` | 1 | migravel |
| `scripts/hades_local_gates.ps1` | 1 | migravel |
| `scripts/hades_orphan_instruments_check.py` | 1 | migravel |

## 5. Ficheiros de excecao (apenas os com legado)

| Ficheiro | Legadas | Categoria |
| --- | --- | --- |
| `docs/mission/HADES_EVIDENCE_INDEX.md` | 166 | excecao:historico |
| `audit/AUTONOMOUS_MISSION_STATE.md` | 32 | excecao:historico |
| `audit/OLLAMA_FULL_MISSION_STATE.md` | 27 | excecao:historico |
| `docs/evidencias/browser-console-audit-f12.json` | 16 | excecao:historico |
| `audit/HARNESS_CAPABILITY_MATRIX.md` | 13 | excecao:historico |
| `docs/evidencias/browser-console-audit-live.json` | 11 | excecao:historico |
| `app/ollama.iss` | 9 | excecao:instalador |
| `audit/AUDITORIA_HADES_2026-10-01.md` | 8 | excecao:historico |
| `audit/FINAL_HARNESS_AUDIT_2026-09-22.md` | 7 | excecao:historico |
| `audit/AUDIT_PONTA_A_PONTA_2026-10-01.md` | 5 | excecao:historico |
| `audit/COMPARACAO_ARVORES_MANUS_OLLAMA_FULL.md` | 4 | excecao:historico |
| `audit/HANDOFF_NEXT_MANUS_20260929.md` | 4 | excecao:historico |
| `scripts/dz23-configure.ps1` | 4 | excecao:env var |
| `audit/HANDOFF_MANUS_20260929.md` | 3 | excecao:historico |
| `audit/OLLAMA_FULL_HANDOFF_20260927.md` | 3 | excecao:historico |
| `internal/multillm/credentials.go` | 3 | excecao:credencial |
| `app/ui/connectors_test.go` | 2 | excecao:teste |
| `audit/CLAUDE_CODEX_RESUME_PROMPT_20260927.md` | 2 | excecao:historico |
| `audit/COMMERCIAL_INTEGRATIONS_SOURCES_2026-09-22.md` | 2 | excecao:historico |
| `audit/FINAL_THREE_AGENT_REVIEW.md` | 2 | excecao:historico |
| `audit/CI_DIAGNOSTICO_20260929.md` | 1 | excecao:historico |
| `audit/E2E_AUDIT_2026-09-30.md` | 1 | excecao:historico |
| `audit/FINAL_SECURITY_REAUDIT_POSTFIX_20260927.md` | 1 | excecao:historico |
| `audit/REAUDITORIA_INDEPENDENTE_POS_FIX_20261009.md` | 1 | excecao:historico |
| `audit/auditoria.md` | 1 | excecao:historico |
| `docs/evidencias/browser-console-audit-audfix2.json` | 1 | excecao:historico |

**Por tipo de excecao:** credencial: 3, env var: 4, historico: 311, instalador: 9, teste: 2

## 6. O que este documento **nao** faz

- **Nao** decide sozinho o que migra: a lista de `EXCECOES` esta' **escrita no gerador**,
  com razao e desbloqueio, e e' a parte que exige julgamento humano.
- **Nao** conta o que o utilizador *ve* na interface (`app/ui/`): isso exige browser real,
  e o escopo aqui e' **texto no repositorio**.
- **Nao** substitui a evidencia `E-990`: aquela e' o **retrato** que motivou esta tabela;
  esta e' a tabela **reconferivel**.
