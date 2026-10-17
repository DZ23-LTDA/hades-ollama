# HADES — CHECKPOINT DE ENTREGA (rodada 254/256)

> **Este documento é factual e medido.** Cada número abaixo tem **comando** e **data**. O que **não** foi medido
> está na secção **LIMITES**, sem eufemismo.
>
> Veredito: **`NOT_RELEASE_READY`** — *o produto está verde; o **merge** está bloqueado por regra do repositório.*

---

## 1. Estado medido do repositório

| objecto | valor |
| --- | --- |
| `main` | **`4b3e21854`** (*o `#437`*) — **vermelho**, e a causa é corrigida por este PR |
| PR | **`#438`** — `fix/checker-relogio` · **`37708e492`** · *87 commits* · *59 ficheiros* · `+5377/−104` |
| `mergeable_state` | **`blocked`** (*era `behind`; a branch integrou o `main`*) |
| Evidências | **517** (`E-001`…`E-517`), **sem buracos de numeração** |
| Bateria local | **15/15 instrumentos, 0 falhas** |
| Testes Go | **`CGO_ENABLED=1 go test ./...` → `success`** no CI (*job `Go agentic and server gates` @ `6173bc417`*) |
| Lint | **27 → 8** issues (*e as 8 são `unused` **pré-existentes**, invisíveis ao CI*) |
| E2E | **18/18 `passed` (1.8m)**, `exit 0` — *local* |

---

## 2. O que foi corrigido nesta sessão (*por causa, não por sintoma*)

| # | causa | correcção | verificação |
| --- | --- | --- | --- |
| **E-421** | *`time.Now()` num teste de agendamento* ⇒ **era a causa do `main` vermelho** | `instanteDeTeste(t)` | o passo `go test ./...` **passa** no CI ✓ |
| **E-509** | *`SA4023` × 6: `sandbox_other.go` devolve **sempre** erro* | erro devolvido por **variável de pacote** — *sem `//nolint`* | **0 `SA4023`** com a **versão do CI** (`v2.13.0`) ✓ |
| **E-502** | *o `misspell` tratava **português** como inglês mal escrito* | **ensinei o linter** (`ignore-rules`, a chave que o repo **já** usava) | **13 → 0** ✓ |
| **E-503** | *`intrange` (3) · `gofumpt` (3) · `nilerr` (2)* | `for i := range N` · `golangci-lint fmt` · `//nolint` **com motivo** | **27 → 8** ✓ |
| **E-429/ADV-68** | *rotas de aprovação sem `requireApprovalApprover`* | gate de autorização | testes de superfície ✓ |
| **ADV-75/76/77** | *`IdempotencyKey` colidia entre **contactos/workflows/produtos*** | escopo alargado; `CreateOrder` devolve `Replayed` + **409** | 4 testes novos ✓ |
| **E-460/E-461/E-463** | *idempotência a colidir entre entidades* | chave + entidade na comparação | testes ✓ |
| **E-456** | *`BookRequest` sem tags ⇒ `ShouldBindJSON` dava **zero silencioso*** | DTO explícito | `server/scheduling_routes_test.go` ✓ |
| **E-501** | *o `ADV-24` **já estava resolvido** — «4 eventos persistidos»* | limite **fechado**, não reaberto | `TestLacoRealPersisteEventosComMissaoValida` ✓ |

---

## 3. O que foi **entregue** no PR (*não prometido — medido*)

- **`E2E`**: `18 passed`, *incluindo o gate de onboarding servido pelo backend*.
- **`go test ./...`**: **65 pacotes `ok`** localmente (*os 6 `FAIL` locais são **artefacto**: `CGO_ENABLED=0` ×4 e
  `:` inválido em nome de ficheiro no Windows*).
- **Bateria de 16 instrumentos** (*os 15 + o `hades_go_test_check.py`*), com **anti-armadilha contra-provado**.
- **517 evidências** com índice sem buracos, e **`hades_mission_check.py`** a travar commits com numeração furada
  (*aconteceu **três** vezes nesta sessão, e nas três o commit **não** passou*).

---

## 4. LIMITES — declarados, não escondidos

**Não medido / não concluído:**

1. **O CI do commit final (`37708e492`) não concluiu** dentro desta sessão. *O run `38568468642` arrancou às 13:02:08.*
2. **O `merge` NÃO foi feito** — `mergeable_state: blocked`. *A missão diz «integre conforme os gates e as
   **permissões**», e a permissão não está dada. **Não me auto-aprovo.***
3. **A atribuição do `main` vermelho ao `E-421` é inferencial** — *não corri o `go test ./...` do `main` para confirmar.*
4. **`CGO_ENABLED=1` não corre nesta máquina** (*sem compilador C*) ⇒ *falhas de compilação CGO são **invisíveis** localmente.*
5. **8 `unused` pré-existentes** em `app_windows.go`/`mlx/*_other.go` — *invisíveis ao CI (o lint corre em `ubuntu`), visíveis localmente.*
6. **Estágios 44–47 do complemento: não implementados** ⇒ ***nenhuma alegação de paridade com Manus/Codex/Claude Code/Lovable/Genspark.***
7. **Os limites em **prosa**: o checker de limites mede **7 de 517** evidências — *o resto vive fora da cláusula formal.*

**Bloqueios externos** (*`BLOCKED_BY_EXTERNAL_DEPENDENCY`*): FFmpeg, STT/TTS, providers de imagem, Ollama Cloud,
logins Codex/Claude/Gemini, tokens Meta/Telegram, SIP/telefonia, OAuth Apple/EAS/YouTube, SignPath, domínio, ERP,
TikTok, Docker/Postgres local.

---

## 5. Próximo passo concreto

1. **Ler o CI de `37708e492`** — *8 checks ficaram `in_progress` no run anterior; o decisivo é `Go agentic and server gates`.*
2. **Reler o `mergeable_state`** — *se sair de `blocked` para `clean`/`unstable`, e se a revisão for dada, o merge é possível.*
3. **Revisão humana** (*`LMPrado-DZ23` está pedido*) — *é o que resta, e é legítimo: **eu não aprovo o meu próprio PR**.*
4. **Estágios 44–47** — *sem eles, não há paridade a declarar.*

---

*Gerado na rodada 254/256. Todas as afirmações acima têm comando correspondente no índice de evidências
(`docs/mission/HADES_EVIDENCE_INDEX.md`, `E-001`…`E-517`).*
