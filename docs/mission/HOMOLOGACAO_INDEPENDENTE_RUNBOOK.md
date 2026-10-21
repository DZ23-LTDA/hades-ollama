## 0. COMANDO UNICO (faz isto primeiro)

Antes de seguir o resto do runbook, corre:

```
python -X utf8 scripts/hades_homologacao_verificar.py
```

Ele executa, na ordem da missao, os **10 geradores** e os **6 gates de documento**, recolhe a saida
literal de cada um e escreve `_tmp/HOMOLOGACAO_<sha>.md` com uma tabela `exit` + ultima linha. Termina
com `HOMOLOGACAO_VERIFICAR=PASS` (exit 0) ou `FAIL` (exit 1).

**O que esse relatorio NAO e':** nao e' homologacao. Foi produzido por uma ferramenta do repositorio,
nao por ti. Ele existe para te poupar a copia manual dos comandos -- a leitura, o juizo e a assinatura
continuam a ser teus. Em particular, **o requisito C2 (tentar falsificar a entrega) nao tem comando**:
tem de ser feito por uma pessoa a procurar testes fracos, gates desligados e promessas sem prova.

O sha do relatorio e' o do HEAD no momento em que correste. Se depois mexeres no repositorio, volta a
correr: um relatorio de outro SHA nao serve para homologar este.

---
# HADES — runbook de homologação independente (Camada C)

> **Quem executa:** um revisor que **não** seja o autor do diff (pessoa ou agente de contexto limpo).
> **O que ele faz:** reproduzir as Camadas A e B a partir de um **clone limpo**, tentar **falsificar**
> a entrega, e assinar ou recusar.
>
> O autor escreve este runbook; **não** o executa como se fosse a homologação — isso seria
> autoaprovação, proibida pela regra R3.

## 0. Pré-requisitos

- Windows com PowerShell 5.1 **ou** PowerShell 7 (os scripts são ASCII puro de propósito).
- Git, Python 3.11+, Node 20+.
- Acesso de leitura ao repositório `DZ23-LTDA/hades-ollama`.
- Nada deste ambiente de autor é necessário: o clone limpo traz tudo o que é preciso.

## 1. Clone limpo (não reutilizar a árvore de trabalho do autor)

```powershell
$sha = '<SHA_A_HOMOLOGAR>'
$clone = "$env:TEMP\hades-homologacao\$sha"
New-Item -ItemType Directory -Force -Path $clone | Out-Null
git clone https://github.com/DZ23-LTDA/hades-ollama $clone
Set-Location $clone
git checkout $sha
git status --short          # ESPERADO: vazio
git rev-parse HEAD          # ESPERADO: $sha
```

## 2. Ambiente Go (toolchain do `go.mod`, sem rede obrigatória para testar)

```powershell
$repo = (Get-Location).Path
New-Item -ItemType Directory -Force -Path "$repo\_tmp\gopath","$repo\_tmp\gotmp" | Out-Null
$env:GOPATH="$repo\_tmp\gopath"; $env:GOTMPDIR="$repo\_tmp\gotmp"
$env:TMP="$repo\_tmp\gotmp"; $env:TEMP="$repo\_tmp\gotmp"
$env:GOMODCACHE="$repo\_tmp\gomodcache"; $env:CGO_ENABLED="0"
$env:GOPROXY="https://proxy.golang.org,direct"; $env:GOSUMDB="sum.golang.org"
go version                   # ESPERADO: >= go1.26.10 (o go.mod exige-o por causa de GO-2026-6809)
```

## 3. Sequência de verificação (copiar um passo de cada vez)

| # | comando | saída esperada |
| --- | --- | --- |
| 1 | `go build ./...` | exit 0, sem saída |
| 2 | `gofmt -l internal server` | **sem ficheiros listados** |
| 3 | `go test ./internal/agent -run 'Vault\|Governad' -count=1` | `ok`, 31 PASS |
| 4 | `go test ./server -run Pare -count=1` | `ok`, 11 PASS |
| 5 | `go test ./internal/agent -run 'Skill\|Selection' -count=1` | `ok` (inclui `TestSkillSelectionVaziaSerializaComoLista`) |
| 6 | `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` | `No vulnerabilities found.` e `exit 0` |
| 7 | `python -X utf8 scripts\hades_release_gate.py --check` | `exit 0` |
| 8 | `python -X utf8 scripts\hades_audit_package.py --check` | `PACOTE DE AUDITORIA CONSISTENTE`, exit 0 |
| 9 | `python -X utf8 scripts\hades_index_shape_check.py --check` | exit 0 |
| 10 | `python -X utf8 scripts\hades_mission_check.py` | exit 0 |
| 11 | `python -X utf8 scripts\hades_handoff_window_check.py --check` | exit 0 |
| 12 | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts\hades_local_gates.ps1` | **81 passos, exit 0** (~25 min; exige `_tmp\gopath\bin` no PATH para o `golangci-lint`) |
| 13 | `gh pr checks 438 --repo DZ23-LTDA/hades-ollama` | **0 `fail`**, 0 obrigatório `pending`; `skipping` só com causa citada no workflow |
| 14 | `cd app\ui\app; npm ci; npx playwright test "accessibility\|first-run-journey\|shell\|smoke-panel\|agent-center-ciclo\|goal-command\|scheduleFlow" --project=chromium` | `passed`, 0 `failed` (pode haver `skipped` de backend real) |
| 15 | `npm run screens:verify` | `capture manifest: PASS (n files)` |

> Se a porta 4173 estiver ocupada por outro processo, definir `$env:HADES_E2E_PORT='4180'` antes do
> passo 14. O `playwright.config.ts` honra essa variável.

## 4. Checklist de FALSIFICAÇÃO (o revisor DEVE tentar cada uma)

| tentativa | como | o que conta como «a entrega resistiu» |
| --- | --- | --- |
| **teste fraco** | procurar asserções que não podem falhar (ex.: `if true`, comparação de valor consigo) em `server/pare_acionavel_http_test.go`, `internal/agent/skill_activation_serie_test.go` | cada teste tem um negativo que falha se a guarda for removida |
| **gate desligado** | procurar `continue-on-error`, `if: false`, `|| true`, `\|\| exit 0` em `.github/workflows/*` | zero em gates de código (só o instalador Windows tem exceções declaradas) |
| **número inventado** | escolher 3 afirmações do README e tentar reproduzi-las | todas reproduzem com o comando citado |
| **screenshot de erro vendida como sucesso** | abrir as imagens em `docs/images/screens/*.png` e comparar com o manifesto | as imagens mostram UI real; `screens:verify` valida sha256/bytes |
| **ID de evidência fantasma** | `grep -o 'E-[0-9]\+' docs -r \| sort -u`, e conferir cada um contra `HADES_EVIDENCE_INDEX.md` | todo o ID citado existe no índice |
| **data nova com artefacto velho** | comparar a data do README com a data das imagens e do manifesto | mesma rodada de captura; `capturedAtUTC` do manifesto é compatível |
| **«verde de um job» vendido como «verde da PR»** | correr `gh pr checks` e contar por estado | a tabela inteira fecha em `0 fail` |

## 5. Bloco de assinatura (preencher ou recusar — não deixar em branco)

```
REVISOR (nome/identificador): ______________________________________
NÃO É AUTOR DO DIFF? (sim/não): ____________________________________
DATA (UTC): ________________________________________________________
SHA HOMOLOGADO (40 chars): _________________________________________
AMBIENTE (SO, versão, go version): _________________________________

CAMADA A (24 requisitos):  [ ] conforme   [ ] não conforme   -> achados: ______
CAMADA B (INTERNO provado):[ ] conforme   [ ] não conforme   -> achados: ______
CAMADA C bloco 4 (falsificação): [ ] resistiu   [ ] cedeu   -> achados: ______

VEREDITO: [ ] HOMOLOGADO   [ ] RECUSADO
OBSERVAÇÕES:
```

## 6. Regra de ouro

> **Enquanto este bloco não estiver preenchido por quem não construiu, o estado é
> `BLOCKED_EXTERNAL` — mesmo com tudo o resto verde.**

Um verde assinado pelo autor não é homologação: é a mesma voz a concordar consigo própria.
