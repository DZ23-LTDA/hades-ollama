# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

**Atualizado por:** E-354 (checkpoint remedido — rodada 39) · **Evidência de criação:** E-316
**Regra:** este arquivo diz **o que fazer a seguir**, não o que já foi feito. Estado completo em
`HADES_MASTER_STATE.json`; prova em `HADES_EVIDENCE_INDEX.md`.

---

## 📌 Checkpoint MEDIDO (rodada 39 — nada aqui é estimativa)

| Medida | Valor medido | Como foi medido |
| --- | --- | --- |
| `main` (remoto) | `bbc9a5412` | `git rev-parse --short origin/main` |
| Evidências no índice | **353** | contagem de linhas `\| E-` |
| Achados de auditoria adversarial | **28** (ADV-01…ADV-28) | `HADES_MASTER_STATE.json` → `v22.auditoria_adversarial` |
| Achados por estado | **19 corrigidos/medidos** · **2 sem defeito** (ADV-02, ADV-08) · **1 revertido por contrato testado** (ADV-19) · **1 resolvido por sonda** (ADV-24) · **1 demonstrado** (ADV-03) · **1 prova por catraca** (ADV-16) · **1 rede ampliada** (ADV-27) | contagem por `estado` |
| Pendências de escopo de tenant | **1** (era **18** há oito fatias) | `scripts/hades_tenant_scope_check.py --check` |
| Classificação de escopo | **24 caminho · 5 identificador · 2 comentário · 1 indefinido declarado** (*MEDIÇÃO CORRIGIDA em E-380: o `join` entrava na linha de comentário e escondia **um uso real de caminho** dentro de `comentario`; a contagem está agora **pinada por teste** — `24` é o número do instrumento, e as linhas históricas E-352/E-355 (22/23) **não** foram reescritas, porque evidência histórica é história: aponta-se a medição que a supera*) | `scripts/hades_scope_classifier.py` |
| Notas de estado registradas | **1029** | `HADES_MASTER_STATE.json` → `notes` |
| Branches remotas | **372** | `git ls-remote --heads origin` |
| Veredito de release | **`NOT_RELEASE_READY`** | checker de missão |

> **Regra de honestidade sobre o SHA**: ele **envelhece a cada merge**. Este checkpoint vale para a revisão
> `bbc9a5412`, medida em **2026-10-13**; qualquer trabalho posterior precisa **remedir** antes de citar.

---

## Fila imediata (ordem do adendo §6)

| # | Ação | Artefato/Comando | Critério de aceite |
| --- | --- | --- | --- |
| 1 | **Classificação por STATEMENT** (a catraca hoje classifica **por linha**: uso multi-linha cai em `indefinido`) | `scripts/hades_scope_classifier.py` + testes | nenhum `indefinido` por limitação do instrumento |
| 2 | **Orçamento acumulado** entre planos de onboarding (hoje o teto é **por plano**) | `internal/agent/company_onboarding.go` | teto declarado também para a soma |
| 3 | **P0-B / T2 nos harnesses externos** (Claude Code, Codex, Manus) | `docs/parity/HADES_T2_MATRIX.md` | **`BLOCKED_BY_EXTERNAL_DEPENDENCY`** até haver login ⇒ **`NAO_MEDIDO`** |
| 4 | **P0-A / T1** — missão Manus-parity: pesquisa **com fontes e data**, planilha/documento, navegador autorizado, estado recuperado após reinício | `docs/parity/HADES_MANUS_PARITY_GATE.md` | artefatos reais; navegador depende de ambiente autorizado |
| 5 | **P0-C / T3** — app full-stack de estoque com auth, DB real (staging), preview, E2E, export e rollback | evidência de staging | **staging real** é dependência externa até haver alvo |
| 6 | **P1-E / T8** — Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail; upload só com OAuth | `docs/parity/…` + evidência | proibido herdar as **simulações** do TUBEGENT; **MP4 real exige FFmpeg** (`BLOCKED_BY_EXTERNAL_DEPENDENCY`) |
| 7 | **P1-F / T9** — REA via MCP/skill governada sobre artefato **próprio/autorizado** | skill + evidência | escopo, timeout e evidência; nunca burlar DRM/acesso |
| 8 | **P1-H** — descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | estágio 39 | **sem** prometer ranking ou recomendação por IAs |

## Bloqueios que **não** se resolvem por código

`BLOCKED_BY_EXTERNAL_DEPENDENCY` com roteiro em `HADES_V22_IMPLEMENTATION_PLAN.md` §3: Ollama Cloud · CLIs
(Codex/Claude/Gemini) · canais Meta/Telegram · telefonia/SIP · contas de negócio · Apple/EAS · YouTube OAuth ·
assinatura Windows · licenças de ferramentas comerciais de REA · **FFmpeg** (produção real de MP4) · **TikTok**
(conector oficial) · **harnesses de paridade** (Claude Code/Codex/Manus exigem credencial).

## Válvulas de honestidade (lembrete permanente)

- Resultado ausente = **`NAO_MEDIDO`**; provedor indisponível **não** é teste aprovado.
- `existe módulo` ≠ capacidade homologada; estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhuma superioridade sobre Manus/Claude/Codex/Lovable/etc. é alegada **sem métrica independente**.
- O SHA medido envelhece a cada merge: registra-se **quando** e **qual revisão** foi medida.
- Chaves coladas em chat são **COMPROMETIDAS** até revogação comprovada pelo proprietário.

## Lições de processo acumuladas (para não repetir)

1. **Gerar derivados → validar → commitar** (nesta ordem).
2. **Nunca cortar branch de uma `main` sem o PR anterior mesclado** — numeração e contagens quebram.
3. **`git push -u`** em branch nova; **confirmar com `ls-remote`**; **nunca reusar nome de branch já mesclado**.
4. **Contrato testado do repositório ganha da preferência do agente** (ADV-19): revertar e **declarar**.
5. **Patcher genérico sobre código desconhecido é arma carregada**: whitelist + assinatura de uma linha + **zero certo
   por tipo**; o **compilador é o árbitro** das arities.
6. **Nunca `git reset --hard` em branch de trabalho**; reconciliar por `cherry-pick` (**sem force-push**).
7. **Hipótese não é diagnóstico; sonda é** (ADV-24): separar hipóteses por medição, não por plausibilidade.
8. **Instrumento que reconhece um único padrão transforma código certo em dívida** (ADV-21) — e o **desconhecido não
   pode passar em silêncio** (E-352).
9. **Limite que não está escrito não é limite: é surpresa** (E-353).
10. **Não declarar verificado o que não se observou**; e **não relatar artefato que não existe** (o PR precisa ser
    **criado** e a criação **confirmada**).
