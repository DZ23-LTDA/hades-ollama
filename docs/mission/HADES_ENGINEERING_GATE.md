# HADES — GATE DE ENGENHARIA INTERMEDIÁRIO (ESTÁGIO 22)

> Artefato obrigatório do estágio 22. **Este NÃO é o release final**: os
> estágios 23–38 seguem pendentes/bloqueados. O gate final (estágio 38) exige
> `docs/mission/HADES_FINAL_RELEASE_GATE.md` e `HADES_RELEASE_REPORT.md`.
>
> Rodada: 2026-10-10 · Base: `main@889750abe` · Branch: `docs/checkpoints-missao-v21`
> Veredito deste gate: **APROVADO COM RESSALVAS** (engenharia verde; produto
> `NOT_RELEASE_READY`).

## 1. Relatório de 20 itens

| # | Item | Estado | Base |
| --- | --- | --- | --- |
| 1 | Funcionalidades implementadas (contagem honesta) | DONE | 12 capacidades `IMPLEMENTADO_E_TESTADO`, 11 `IMPLEMENTADO_NAO_HOMOLOGADO`, 10 `PARCIAL` em `audit/HADES_CAPABILITY_MATRIX.md` |
| 2 | Funcionalidades homologadas ponta a ponta | NOT_DONE | 0 estágios `COMPLETED_VERIFIED` |
| 3 | Plugins | PARTIAL | `plugin_scope.go` (escopo por organização) + `plugin_lifecycle_test.go`, `plugin_persistence_test.go` |
| 4 | MCPs | PARTIAL | catálogo + escopos `mcp:call`/`mcp:remote:call`; execução real não homologada (B-10) |
| 5 | Skills | PARTIAL | `SkillManifest`, `validateSkillManifest`, assinatura ed25519 e testes de não-confiança |
| 6 | Agentes | PARTIAL | `swarm.go` (`AgentOrchestrator` com papéis, orçamento, plano, run, cancel) |
| 7 | Subagentes | PARTIAL | `SubagentRunner` + testes; sem missão multiagente real contra modelo |
| 8 | Modelos locais | PARTIAL | Ollama como padrão + `OllamaEmbedder`; descoberta real sem chamada live (B-07) |
| 9 | APIs externas | BLOCKED | adaptadores Anthropic/OpenAI/xAI-Grok/Gemini/Groq/DeepSeek/Mistral; **sem** vLLM, llama.cpp, LM Studio, Lemonade, OpenRouter; homologação exige chave (B-07) |
| 10 | Harnesses | AUSENTE (em `main`) | `harness` não existe no código de `main`; implementação está no PR #62 (branch-local) |
| 11 | Plataformas | PARTIAL | CI multi-SO verde; smoke nativo e instalador assinado não homologados (B-06) |
| 12 | Testes | DONE | Go: `internal/agent` 102 arquivos/583 testes, `server` 83/434; Web: 65 arquivos/396 testes; E2E: 10 testes |
| 13 | Defeitos corrigidos | DONE | misspell (PR #67), `flowGraph` 31/31 (PR #68), E2E primeira execução (PR #71), E2E Studio (PR #73) |
| 14 | Riscos residuais | DONE (documentados) | `audit/HADES_RELEASE_BLOCKERS.md` R-02…R-08 |
| 15 | Dependências externas | DONE (documentadas) | `docs/mission/HADES_BLOCKERS.md` B-01…B-10 |
| 16 | Artefatos | PARTIAL | SBOM gerado no CI; pacotes por SO sem assinatura |
| 17 | Instruções | DONE | `docs/mission/HADES_HANDOFF.md` (ordem de leitura, ambiente, comandos) |
| 18 | Commits/PRs | DONE | checkpoints (#72) e E2E do Studio (#73) verdes; #56–#71 abertos |
| 19 | Evidências | DONE | `docs/mission/HADES_EVIDENCE_INDEX.md` E-001…E-047 |
| 20 | Status real | DONE | `NOT_RELEASE_READY` — declarado, não maquiado |

## 2. Suíte executada neste gate

| Suíte | Comando | Resultado |
| --- | --- | --- |
| Go vet/build | `go vet ./internal/agent/` · `go build ./...` | exit 0 · exit 0 |
| Go lint/format | `golangci-lint run` · `gofmt -l` · gofumpt | `0 issues.` · vazio · exit 0 |
| Go testes | `go test ./internal/agent/` | 3 falhas **ambientais pré-existentes** (B-09) |
| Web lint | `npm run lint` | exit 0 (1 warning pré-existente) |
| Web unit | `npx vitest run` | 65 arquivos / 396 testes |
| Web build | `npm run build` | ok |
| E2E | `npx playwright test` | 10 passed (1 worker) |
| Classificador CI | `python scripts/classify-ci-surfaces*.py` | 13/13 testes; `PR gate` verde nos PRs #71/#72/#73 |

## 3. Critério do gate

- **Aprovado** o que é engenharia: base estável, gates verdes, CI multi-SO,
  suíte relevante existente, causa raiz corrigida nos defeitos tratados.
- **Reprovado** o que é produto: homologação ponta a ponta, canais reais,
  provedores reais, assinatura de release, isolamento multitenant provado.
- Este gate **não** promove nenhum estágio a `COMPLETED_VERIFIED`.

## 4. Próximo gate

O gate final (estágio 38) exige, além deste: empresa fictícia operada de ponta a
ponta em sandbox, isolamento por organização provado, smoke nativo por SO e
release assinado. Enquanto qualquer requisito obrigatório estiver sem evidência
real, o estado global permanece `NOT_RELEASE_READY`.
