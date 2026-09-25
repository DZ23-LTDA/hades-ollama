# Release Readiness — Ollama Classe A+

Este documento define a política de **um único SHA candidato passar por todo o
gate** antes de virar release. Ele não inventa checks: lista os workflows reais
já presentes em `.github/workflows/`.

## Princípio

Uma release só é válida quando **o mesmo commit** (o SHA candidato) passou por
todos os gates aplicáveis. Não vale juntar "parte verde na main" com "parte
verde num PR anterior". O RC é congelado num SHA e todos os workflows daquele
SHA devem estar verdes.

## Gates reais (contexts de status = nome do job)

| Workflow (`.github/workflows/`) | Job / context | Cobre |
|---|---|---|
| `class-a-plus-integrity.yaml` | **Preserve Class A+ surfaces** | guarda de integridade das superfícies + gofmt protegido |
| `dz23-agentic-quality.yaml` | **Go agentic and server gates** | go build/vet/test backend + server + browser operator |
| `dz23-agentic-quality.yaml` | **PostgreSQL RLS, Redis DLQ and OTLP integration** | RLS por org, DLQ/replay, OTLP |
| `dz23-agentic-quality.yaml` | **Web and mobile quality** | web typecheck/lint/tests/prod-audit + mobile typecheck/audit/policy |
| `release.yaml` | **Agentic release quality gate** | gate de qualidade agregado da release |
| `test.yaml` | test matrix (Linux/macOS/Windows) + race | testes Go multiplataforma |
| `test-install.yaml` | `test` | instala o fork de fonte e **verifica que o binário é o Classe A+** |
| `dz23-windows-installer.yaml` | `installer` | build CPU + Desktop + **instalador Windows (não assinado)** |
| `dz23-multi-provider.yaml` / `dz23-provider-smoke.yaml` | provider gates/smoke | roteamento multi-provider |

> GPU (CUDA/ROCm/Vulkan/MLX) permanece **manual / runner específico** — não é
> gate por commit. `release.yaml` cobre os builds pesados quando há runner.

## Gate automatizado (single-SHA)

O workflow [`release-readiness.yaml`](../.github/workflows/release-readiness.yaml)
**afirma** que o SHA candidato tem o conjunto real de provas verdes **naquele
commit**. Ele faz **espera ativa (poll, até 30 min)** para o caso de a tag ser
criada logo após o merge, pois os checks são disparados pelo `push:main`. Cada
check obrigatório só conta com conclusão **`success`** — `skipped`, `cancelled`,
`failure`, `timed_out`, `stale` ou `neutral` **reprovam** o gate.

Conjunto obrigatório verificado — apenas os contexts que **realmente rodam num
commit de `main`/tag** e constituem a prova de release **deste fork** (camada
agentic Classe A+): `Preserve Class A+ surfaces`, `Go agentic and server gates`,
`PostgreSQL RLS, Redis DLQ and OTLP integration`, `Web and mobile quality`,
`Generate SBOM`.

### O que NÃO entra neste gate (e por quê) — divulgação honesta

- **`PR gate`** é um agregador **exclusivo de `pull_request`**: não existe num
  commit de `main`/tag. Seu conteúdo real (os 3 deep gates acima) já é exigido
  individualmente aqui, então nenhuma cobertura se perde.
- **`test (ubuntu/windows/macos-latest)`**, **`race (ubuntu/macos-latest)`** e
  **`go_mod_tidy`** são a **matriz upstream do Ollama** (`test.yaml`), disparada
  **só em `pull_request`** pelo design do upstream (o job `changes` lê
  `github.event.pull_request.*` e quebra fora de PR). Torná-la tag-aware
  **forkearia o CI do upstream** e dificultaria puxar atualizações do Ollama —
  um objetivo explícito do projeto. Por isso ela roda como **sinal advisory nos
  PRs**, mas **não fica vinculada ao SHA exato do RC**.
  **Limitação conhecida:** a prova cross-OS de build/teste Go do RC vem do head
  do PR que originou o merge (validado antes do merge), **não** do commit de
  merge/tag. Quem quiser essa prova vinculada ao SHA do RC precisa rodar a
  matriz upstream por `workflow_dispatch` naquele SHA e anexar a evidência.

## Procedimento de RC (um SHA) — VALIDAÇÃO É PRE-TAG

> **Ordem obrigatória.** As tags `v*` são protegidas (sem delete, sem
> non-fast-forward). Por isso a validação principal acontece **ANTES** de criar
> a tag — nunca crie a tag para "depois validar", senão uma tag ruim fica presa.

1. Congele o candidato num SHA da `main` já verde.
2. **PRE-TAG:** rode `release-readiness` por `workflow_dispatch` com `ref=<SHA>`
   e confirme que ele passou (todos os checks obrigatórios `success` naquele SHA).
3. Gere os artefatos **daquele SHA** (installer + tarball via os workflows de
   build, por `workflow_dispatch`), rode o smoke, colete **checksums SHA-256** e
   **SBOM**, e associe a metadata ao SHA.
4. Só **então** crie a tag do RC (versão nova; **não** reciclar `v0.1.0`). Ex.:
   `v0.2.0-rc.1`, apenas se semanticamente apropriado.
5. O disparo de `release-readiness` no push da tag é **verificação defensiva
   adicional**, não o primeiro gate.
6. Publique a release imutável vinculada ao SHA testado (via `release.yaml`, que
   registra `SOURCE_SHA`/`VERSION`/`GATE_RUN_ID` — ver §8 do plano).

## Estado verdadeiro (não marque PRODUCTION_READY sem prova)

- **Instalador Windows (amd64, não assinado):** produzido pelo CI
  (`dz23-windows-installer.yaml`). **Assinatura Authenticode = BLOCKED** (sem
  certificado). Uma release não deve aparecer como "signed" sem certificado real.
- **macOS/Linux packaging:** `release.yaml` cobre builds; empacotamento final
  assinado/notarizado depende de credenciais externas → **BLOCKED**.
- **Jornada E2E de primeira execução (install→onboarding→missão→artifact→
  approval→cancel→restart→persistência→upgrade→rollback):** parcialmente coberta
  por `test-install.yaml`; a jornada UI-first completa ainda precisa de smoke
  Playwright dedicado.

O estado agregado atual do produto é **RELEASE_CANDIDATE** (núcleo verde,
instalador Windows não assinado disponível), com assinatura e publicação
multiplataforma **BLOCKED_BY_EXTERNAL_DEPENDENCY**.
