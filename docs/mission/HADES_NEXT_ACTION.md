# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

**Atualizado por:** E-340 (checkpoint remedido) · **Evidência de criação:** E-316
**Regra:** este arquivo diz **o que fazer a seguir**, não o que já foi feito. Estado completo em
`HADES_MASTER_STATE.json`; prova em `HADES_EVIDENCE_INDEX.md`.

---

## 📌 Checkpoint MEDIDO (rodada 25 — nada aqui é estimativa)

| Medida | Valor medido | Como foi medido |
| --- | --- | --- |
| `main` (remoto) | `070d1c57f` | `git rev-parse --short origin/main` |
| Evidências no índice | **340** | contagem de linhas `\| E-` |
| Achados de auditoria adversarial | **19** (ADV-01…ADV-19) | `HADES_MASTER_STATE.json` → `v22.auditoria_adversarial` |
| Pendências de escopo de tenant | **9** (eram 18 há duas fatias) | `scripts/hades_tenant_scope_check.py --check` |
| Branches remotas | **359** | `git ls-remote --heads origin` |
| Veredito de release | **`NOT_RELEASE_READY`** | checker de missão |

**Achados por estado (medido)**: `CORRIGIDO`/`CORRIGIDO E MEDIDO` **15** · `SEM DEFEITO` **2** (ADV-02, ADV-08) ·
`REVERTIDO por contrato testado` **1** (ADV-19) · `DECLARADO E PROVADO POR CATRACA` **1** (ADV-16).

---

## Fila imediata (ordem do adendo §6)

| # | Ação | Artefato/Comando | Critério de aceite |
| --- | --- | --- | --- |
| 1 | **Medir por comportamento** as **9 pendências** restantes da catraca (`voice_active_call`, `company_factory`, `sales_workforce`, `company_ledger`, `business_snapshots`, `marketplace_sync` + vizinhos) — usar o patcher **v3** (whitelist + assinatura de uma linha + **zero certo por tipo**) | `scripts/hades_tenant_scope_check.py --check` (o número **cai**) + testes | cada arquivo sai de `NAO_MEDIDO` **só** com teste que ataque a API com organização vazia |
| 2 | **Exercitar os métodos vizinhos** dos arquivos já medidos (`SuggestReply`, `SearchKnowledge`, `SetFirstResponseSLA`, `ListOpportunities`, `PipelineSummary`) | teste de comportamento | whitelist atual **não** é auditoria completa do arquivo |
| 3 | **P0-B / T2** — benchmark de tarefa equivalente (bug real em repo fixture): red → green → diff por hunk → PR → regressão | `docs/parity/HADES_HARNESS_BENCHMARKS.md` | resultado por harness **ou** `NAO_MEDIDO`; nunca “ok” sem comando |
| 4 | **P0-E / T4** — três especialistas em **worktrees isolados** com merge controlado | teste E2E + evidência | artefatos verificáveis, limites de custo/profundidade |
| 5 | **P0-A / T1** — missão Manus-parity: pesquisa **com fontes e data**, planilha/documento, navegador autorizado, estado recuperado após reinício | `docs/parity/HADES_MANUS_PARITY_GATE.md` | artefatos reais; navegador depende de ambiente autorizado |
| 6 | **P0-C / T3** — app full-stack de estoque com auth, DB real (staging), preview, E2E, export e rollback | evidência de staging | **staging real** é dependência externa até haver alvo |
| 7 | **P1-E / T8** — Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail; upload só com OAuth | `docs/parity/…` + evidência | proibido herdar as **simulações** do TUBEGENT |
| 8 | **P1-F / T9** — REA via MCP/skill governada sobre artefato **próprio/autorizado** | skill + evidência | escopo, timeout e evidência; nunca burlar DRM/acesso |
| 9 | **P1-H** — descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | estágio 39 | **sem** prometer ranking ou recomendação por IAs |

## Bloqueios que **não** se resolvem por código

`BLOCKED_BY_EXTERNAL_DEPENDENCY` com roteiro em `HADES_V22_IMPLEMENTATION_PLAN.md` §3: Ollama Cloud · CLIs
(Codex/Claude/Gemini) · canais Meta/Telegram · telefonia/SIP · contas de negócio · Apple/EAS · YouTube OAuth ·
assinatura Windows · licenças de ferramentas comerciais de REA · **FFmpeg** (produção real de MP4) · **TikTok**
(conector oficial).

## Válvulas de honestidade (lembrete permanente)

- Resultado ausente = **`NAO_MEDIDO`**; provedor indisponível **não** é teste aprovado.
- `existe módulo` ≠ capacidade homologada; estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhuma superioridade sobre Manus/Claude/Codex/Lovable/etc. é alegada **sem métrica independente**.
- O SHA medido envelhece a cada merge: registra-se **quando** e **qual revisão** foi medida.
- Chaves coladas em chat são **COMPROMETIDAS** até revogação comprovada pelo proprietário.

## Lições de processo desta fase (para não repetir)

1. **Gerar derivados → validar → commitar** (nesta ordem). Validar antes de regenerar reprova por divergência de gate.
2. **Nunca cortar branch de uma `main` sem o PR anterior mesclado** — numeração de evidências e contagens quebram.
3. **`git push -u`** em branch nova; **confirmar com `git ls-remote`** em vez de confiar no silêncio do comando.
4. **Contrato testado do repositório ganha da preferência do agente** (ADV-19): reverter e **declarar** o motivo.
5. **Patcher genérico sobre código desconhecido é arma carregada**: whitelist + assinatura de uma linha + **zero certo
   por tipo**; o **compilador é o árbitro** das arities.
6. **Nunca `git reset --hard` em branch de trabalho**; para reconciliar, `cherry-pick` sobre a base certa (**sem
   force-push**).
