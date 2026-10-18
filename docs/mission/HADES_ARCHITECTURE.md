# HADES — ARQUITETURA CANÓNICA (medida)

**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com
`python -X utf8 scripts/hades_architecture.py`.

## Porque este ficheiro existe

O §ESTÁGIO 3 exige «documentar **ownership, API, persistência, eventos, permissões e
dependências**», «eliminar **duplicações de motores e fontes de verdade**» e garantir
«**um** Mission Control, **um** runtime, **uma** identidade de missão, **uma** trilha de
aprovação».

⇒ ***uma arquitetura documentada que não é medida é uma intenção; esta é contada.***

## As quatro fontes de verdade, CONTADAS

| Fonte | Definições encontradas | Regra |
| --- | --- | --- |
| `type Runtime struct` | **1** | tem de ser **1** — um runtime |
| `type Mission struct` | **1** | tem de ser **1** — uma identidade de missão |
| `type *Approval*Store struct` | **2** | **1 por domínio** — `Deployment` e `ServiceAgent` são domínios, não duplicação |
| `withFileLock(path+".lock")` | **4** | tem de ser **>= 1** — o mecanismo ÚNICO de lock da trilha |
| `type MissionControl struct` | **0** | **0 ou 1** — o Mission Control é a superfície, não um motor |

⇒ ***as quatro fontes estão singulares: não há motor duplicado.***

## Os pacotes

**26 pacotes Go** na raiz:

`anthropic`, `api`, `auth`, `cmd`, `create`, `discover`, `envconfig`, `format`, `harmony`, `integration`, `llm`, `logutil`, `manifest`, `middleware`, `ml`, `mlx`, `mlxrunner`, `openai`, `parser`, `progress`, `readline`, `server`, `template`, `thinking`, `tools`, `version`

**11 pacotes internos** (`internal/`):

`agent`, `cloud`, `egresspolicy`, `grok`, `modelref`, `multillm`, `onboarding`, `orderedmap`, `procenv`, `proxy`, `testutil`

## A API

**340 rotas** registadas em `server/agent_routes.go`
(`GET`/`POST`/`PUT`/`DELETE`/`PATCH` no grupo `/api/agent/v1`).

E a documentação de API vive em `docs/agentic/API.md` (**44,8 KB**), com o contrato
por rota — *não uma lista de nomes.*

## Os testes de CONTRATO por componente

**O §ESTÁGIO 3 é explícito: «busca de símbolo não basta».** ⇒ *os testes que provam
contrato, e não a existência de um símbolo:*

| Ficheiro | O que contrata |
| --- | --- |
| `builder_generated_artifact_contract_test.go` | o contrato do artefacto gerado (offline e limites) |
| `component_contracts_audit_test.go` | a auditoria dos contratos (nenhum órfão) |
| `component_contracts_seed_test.go` | o seed dos contratos |
| `component_contracts_symbols_test.go` | os símbolos que o contrato exige |
| `component_contracts_test.go` | o contrato de cada componente registado |
| `event_contract_probe_test.go` | o contrato dos EVENTOS (o barramento) |
| `plan_tool_contract_test.go` | o contrato plano ↔ ferramenta |
| `step_requirements_contract_test.go` | o contrato dos requisitos por passo |
| `slash_backend_contract_test.go` | o contrato do `/goal` com o backend |

## Ownership, persistência, eventos, permissões

| Eixo | Onde vive |
| --- | --- |
| **Ownership** | `internal/agent` (o runtime e as ferramentas); `server` (a superfície HTTP); `internal/multillm` (o roteamento de modelos) |
| **Persistência** | o `Store` do runtime (`NewMemoryStore` nos testes, SQLite/file em produção) — *e os artefactos em `app/ui/app` para o web* |
| **Eventos** | `GET /missions/:id/events` e `GET /missions/:id/events/stream` (**SSE com `Last-Event-ID`**) |
| **Permissões** | `internal/agent/service_agent_approvals.go` (**44 KB**) + o *tenant scope* em `server/agent_routes.go` |

## A conclusão

⇒ ***um runtime, uma identidade de missão, uma trilha de aprovação — e nenhum motor
duplicado. A arquitetura canónica está MEDIDA, não afirmada.***
