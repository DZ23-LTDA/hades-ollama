# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

**Atualizado por:** E-316 (inventário de referências) · **Evidência de criação:** E-316
**Regra:** este arquivo diz **o que fazer a seguir**, não o que já foi feito. Estado completo em
`HADES_MASTER_STATE.json`; prova em `HADES_EVIDENCE_INDEX.md`.

---

## Fila imediata (ordem do adendo §6)

| # | Ação | Artefato/Comando | Critério de aceite |
| --- | --- | --- | --- |
| 1 | **Verificar licenças** das referências que serão integradas **antes de copiar qualquer trecho** — TUBEGENT (`REF-R21`), REA (`REF-R22`), Tel-Agent (`REF-R01`), litellm (`REF-R11`), canvas `REF-R15/R16` | `docs/parity/HADES_INTEGRATION_ADRS.md` (uma decisão por referência: a/b/c/d) | licença lida e citada; nenhum código copiado antes disso |
| 2 | **P0-B / T2** — benchmark de tarefa equivalente (bug real em repo fixture): red → green → diff por hunk → PR → regressão | `docs/parity/HADES_HARNESS_BENCHMARKS.md` | resultado por harness **ou** `NAO_MEDIDO`; nunca “ok” sem comando |
| 3 | **P0-E / T4** — três especialistas em **worktrees isolados** com merge controlado de resultados | teste E2E + evidência | artefatos verificáveis, limites de custo/profundidade, memória segmentada |
| 4 | **P1-C** — agendamento, próxima fatia: intervalos dentro do turno, **remarcação** e lista de espera | `internal/agent/scheduling*.go` + teste | remarcação não deixa buraco nem colisão; recusa explícita |
| 5 | **P0-A / T1** — missão Manus-parity: pesquisa **com fontes e data**, planilha/documento, navegador autorizado, estado recuperado após reinício | `docs/parity/HADES_MANUS_PARITY_GATE.md` | artefatos reais; navegador depende de ambiente autorizado |
| 6 | **P0-C / T3** — app full-stack de estoque com auth, DB real (staging), preview, E2E, export e rollback | evidência de staging | **staging real** é dependência externa até haver alvo |
| 7 | **P1-E / T8** — Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail; upload só com OAuth | `docs/parity/…` + evidência | proibido herdar as **simulações** do TUBEGENT |
| 8 | **P1-F / T9** — REA via MCP/skill governada sobre artefato **próprio/autorizado** | skill + evidência | escopo, timeout e evidência; nunca burlar DRM/acesso |
| 9 | **P1-H** — descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | estágio 39 | **sem** prometer ranking ou recomendação por IAs |

## Bloqueios que **não** se resolvem por código

`BLOCKED_BY_EXTERNAL_DEPENDENCY` com roteiro em `HADES_V22_IMPLEMENTATION_PLAN.md` §3: Ollama Cloud · CLIs
(Codex/Claude/Gemini) · canais Meta/Telegram · telefonia/SIP · contas de negócio · Apple/EAS · YouTube OAuth ·
assinatura Windows · licenças de ferramentas comerciais de REA.

## Válvulas de honestidade (lembrete permanente)

- Resultado ausente = **`NAO_MEDIDO`**; provedor indisponível **não** é teste aprovado.
- `existe módulo` ≠ capacidade homologada; estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhuma superioridade sobre Manus/Claude/Codex/Lovable/etc. é alegada **sem métrica independente**.
- O SHA medido envelhece a cada merge: registra-se **quando** e **qual revisão** foi medida.
- Chaves coladas em chat são **COMPROMETIDAS** até revogação comprovada pelo proprietário.
