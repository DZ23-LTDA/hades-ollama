# Auditoria de dependências web — Ollama Full

Verificado em **2026-09-24** (`app/ui/app`, npm 10.9.8 / node 22).

## Resultado

| Escopo | high/critical | moderate | low |
|---|---|---|---|
| **Produção** (`npm audit --omit=dev`) | **0** | **0** | 0 |
| Dev + prod (`npm audit`) | 0 | 7 | 0 |

**Zero high/critical.** As 7 moderate são **exclusivamente de tooling de teste**
(não entram no bundle de produção nem no instalador).

## As 7 advisories (dev-only)

Todas derivam de uma única raiz na cadeia do Vitest:

- **GHSA-82fw-gwwq-j7x9** — *Vitest: Path Traversal / Arbitrary File Read via
  `@vitest/mocker` Redirect Mock* (moderate). Afeta o servidor de mock em tempo
  de **teste/dev**, não o artefato publicado.
- Pacotes atingidos por transitividade: `vitest`, `@vitest/mocker`,
  `@vitest/browser`, `@vitest/coverage-v8`, `@vitest/ui`, `storybook`,
  `@storybook/addon-vitest`.

## Decisão (por que não foi "corrigido" agora)

- O único fix disponível é **Vitest v5** — um salto **MAJOR** (de `^3.2.4`), que
  conflita com `storybook@9`/`@storybook/addon-vitest` e reescreve API/config de
  teste. Tentado nesta rodada: o install do major **não resolve limpo** e a
  migração quebraria a suíte.
- Regra do projeto (§10): não usar `--force` cegamente nem degradar o produto.
  Como a advisory é **dev-only, moderate, sem fix não-quebrante**, ela é
  registrada como **risco transitivo aceito**, não mascarado.
- `npm audit fix` (sem `--force`) foi executado e **não reduz** a contagem
  (os patches seguros do Storybook não tocam a raiz Vitest).

## Plano de correção (follow-up rastreável)

Migrar o tooling de teste para **Vitest v5** em uma branch dedicada, junto com a
compatibilização do Storybook, rodando `typecheck + lint + test + coverage +
build-storybook + build + Playwright` antes de mesclar. É melhora de
tooling, **não bloqueia release** (produção = 0 vuln). O Dependabot
(`.github/dependabot.yml`, grupo `web-dev-tooling`) abrirá os PRs de atualização.

## Mobile (`apps/mobile-agentic`)

O gate `Web and mobile quality` (workflow `dz23-agentic-quality.yaml`) já executa
`Mobile production dependency audit` e passa. Ver [RELEASE_READINESS.md](RELEASE_READINESS.md).
