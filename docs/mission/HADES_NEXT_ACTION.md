# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

**Atualizado por:** E-547 (checkpoint remedido — rodada 101) · **Evidência de criação:** E-316
**Regra:** este arquivo diz **o que fazer a seguir**, não o que já foi feito. Estado completo em
`HADES_MASTER_STATE.json`; prova em `HADES_EVIDENCE_INDEX.md`.

> ⚠️ **ESTE ARQUIVO ESTEVE PARADO 62 RODADAS** (*rodada 39 → 101*): dizia **353** evidências quando havia **547**, e a
> «fila imediata» era de uma era em que os **estágios 44–47 nem existiam**. ⇒ *Foi o `E-544` que o mediu: dos cinco
> documentos que a FASE 7 manda actualizar, **três** estavam parados há ~16 h, e este não estava na lista — *mas estava
> parado pela mesma razão*. É a **terceira** vez que um documento desta casa escreve a regra que ele próprio viola
> (*o `HANDOFF` e o `PENDING_WORK` também avisam que «os números envelhecem»*).*

---

## 📌 Checkpoint MEDIDO (rodada 101 — nada aqui é estimativa)

| Medida | Valor medido | Como foi medido |
| --- | --- | --- |
| `main` (remoto) | `4b3e21854` | `git rev-parse --short origin/main` |
| Branch do PR #438 | `eaad692e0` | `git rev-parse --short origin/fix/checker-relogio` |
| Evidências no índice | **547** | contagem de linhas `\| E-` |
| Estágios | **38** (*44–47 entregues*) | `HADES_MASTER_STATE.json` → `stages` |
| Bloqueios declarados | **17** (*17 itens / 17 ids únicos — o `B-15` duplicado foi removido no `E-545`*) | `hades_mission_check.py` |
| Conflitos com a `main` | **nenhum** | `git merge-tree --write-tree` → exit 0 |
| Credenciais expostas | **nenhuma** (*20 ocorrências de padrão `ghp_`, **20/20** em ficheiros de teste, **0** em produção*) | varrimento + classificação (`E-547`-era) |
| Veredito de release | **`NOT_RELEASE_READY`** | checker de missão |

> **Regra de honestidade sobre o SHA**: ele **envelhece a cada merge**. Este checkpoint vale para `eaad692e0`, medido em
> **2026-10-14**; qualquer trabalho posterior precisa **remedir** antes de citar.

---

## Fila imediata — REORDENADA pelas rodadas 40–101

### ✅ Já não está na fila (*entregue e testado*)
| O que era pendente | Onde ficou |
| --- | --- |
| **Estágio 44** — Cowork & Spark | `E-533`: `/spark`, `/cowork`, `/reject` na infra existente (*mesma rota de `/goal`*) |
| **Estágio 45** — Multimodal Studio | `E-529` matriz modalidade×engine · `E-535` a **recusa antes de disco/rede** · documentos PDF/DOCX/XLSX/PPTX já existiam |
| **Estágio 46** — Companion | `E-536` estados derivados de **eventos verificados** · `E-538` a11y ligada ao `prefers-reduced-motion` **existente** |
| **Estágio 47** — Artifact Drive | `E-537` busca + CAS · `E-539` comparar/restaurar **sem apagar história** · `E-542` dedup **dentro do tenant** |
| **FASE 1** — `race (macos)` | **provada**: 11/11 etapas, macOS 10 min 28 s vs ubuntu 6 min 32 s |

### ⏭️ O que resta, por ordem
| # | Ação | Critério de aceite |
| --- | --- | --- |
| 1 | **Revisão humana do PR #438** (*`LMPrado-DZ23`*) — **não** se contorna | aprovação registada; só depois o merge |
| 2 | **Contrato duplicado**: o `server` **repete** o número do `internal/agent` (*35→38 foi o sintoma*) | o `server` **lê** o valor em vez de o repetir — *dívida registada no `E-547`* |
| 3 | **P0-B / T2** harnesses externos (*Claude Code, Codex, Manus*) | `BLOCKED_BY_EXTERNAL_DEPENDENCY` até haver login ⇒ `NAO_MEDIDO` |
| 4 | **P0-A / T1** paridade Manus: pesquisa com fontes, planilha/documento, navegador autorizado | artefatos reais; navegador depende de ambiente autorizado |
| 5 | **P0-C / T3** app full-stack com auth, DB real, preview, E2E, rollback | **staging real** é dependência externa até haver alvo |
| 6 | **P1-E / T8** Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail | **MP4 real exige FFmpeg** ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` |
| 7 | **P1-F / T9** REA via MCP/skill governada sobre artefato próprio/autorizado | escopo, timeout e evidência; nunca burlar DRM |
| 8 | **P1-H** descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | **sem** prometer ranking ou recomendação por IAs |

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
