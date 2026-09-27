# Proteção da branch `main` — Ollama Full

> **Status: APLICADO (2026-09-24)** com a conta admin `DZ23-LTDA`.
>
> - `main-protection` [branch] — ruleset **active** (`main.protected = true`).
> - `release-tag-protection` [tag] — ruleset **active** (protege `refs/tags/v*`).
> - Private Vulnerability Reporting — **habilitado**.
>
> Os JSONs versionados em `.github/rulesets/` são a fonte de verdade; reaplique
> com os comandos abaixo se precisar recriar.

## O que o ruleset garante

- PR obrigatório para mudar a `main` (1 aprovação, resolução de threads).
- Bloqueio de **force-push** (`non_fast_forward`) e de **delete** (`deletion`).
- **Required status checks** com os workflows reais do projeto, e a branch
  precisa estar atualizada antes do merge (`strict`).
- Dismiss de aprovação stale ao novo push.

Os contexts exigidos sao **estaveis e sempre criados** em todo PR:
`Preserve Class A+ surfaces` (workflow `class-a-plus-integrity`, sem filtro de
paths) e `PR gate` (workflow `pr-gate`, agregador que roda em todo PR e, quando
o PR toca runtime/web/mobile, espera e verifica os gates profundos de
`dz23-agentic-quality`). Isso evita o deadlock de exigir checks com filtro de
paths que nao sao criados em PRs docs-only.

## Como aplicar (admin do repositório)

Ruleset pronto em [`.github/rulesets/main-protection.json`](../.github/rulesets/main-protection.json).

```bash
gh api -X POST repos/DZ23-LTDA/ollama-classe-a-plus/rulesets \
  --input .github/rulesets/main-protection.json
```

Ou via UI: **Settings → Rules → Rulesets → New ruleset → Import** o JSON acima.

## Revisão independente (single-maintainer)

Colaboradores reais no repo: **DZ23-LTDA** (org/admin) e **LMPrado-DZ23**
(maintainer) — controlados pela mesma pessoa. **Não há revisor humano
independente disponível** (projeto single-maintainer).

Por isso, `require_code_owner_review` fica **false** (o único code owner seria o
próprio autor — seria só cerimônia). Em vez de inventar revisores, aplicamos:

- `required_approving_review_count: 1` + `required_review_thread_resolution`;
- **`require_last_push_approval: true`** — a conta que fez o último push não pode
  ser a aprovadora; como existem duas contas reais (DZ23-LTDA e LMPrado-DZ23), o
  requisito é satisfazível (push por uma, aprovação pela outra) e adiciona
  fricção real contra self-merge cego;
- superfícies sensíveis (`.github/workflows/**`, `.github/rulesets/**`,
  `CODEOWNERS`, `SECURITY.md`, `scripts/check-class-a-plus-integrity.sh`,
  scripts de release/install) continuam guardadas pelo required check
  **Preserve Class A+ surfaces** (integrity guard) + **PR gate**.

> **Limitação honesta:** separação de contas ≠ separação de humanos. Revisão
> independente por um segundo humano permanece **BLOCKED** até haver um segundo
> mantenedor real.

## Itens adicionais

- **Private Vulnerability Reporting** — ✅ habilitado (ver [`../SECURITY.md`](../SECURITY.md)).
- Proteção de **tags de release** — ✅ aplicada via
  [`.github/rulesets/tag-protection.json`](../.github/rulesets/tag-protection.json).
- Exigir **commits/tags assinados** para releases (`required_signatures`): ainda
  não aplicado — depende de a equipe adotar assinatura GPG/Sigstore.
- `CODEOWNERS`: pode ser adicionado quando os owners reais forem confirmados;
  então habilite `require_code_owner_review: true` no ruleset.

## Verificação após aplicar

```bash
gh api repos/DZ23-LTDA/ollama-classe-a-plus/rulesets            # lista rulesets
gh api repos/DZ23-LTDA/ollama-classe-a-plus/branches/main -q '.protected'
```
