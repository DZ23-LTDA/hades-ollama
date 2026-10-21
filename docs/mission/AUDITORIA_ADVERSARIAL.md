# HADES — auditoria adversarial da entrega (prova P3)

> **Regra do exercício:** eu escrevo as tentativas de **falsificar** a minha própria entrega e mostro
> que cada uma **falha**. Se alguma passasse, a entrega estava comprometida e eu teria de a corrigir
> antes de a declarar.
>
> SHA auditado: `764fd1fbd` · branch `fix/checker-relogio` · PR #438 · 2026-10-21.
> **Este documento não é homologação independente** — é o autor a atacar o próprio trabalho. A
> Camada C exige alguém que não seja o autor (ver `HOMOLOGACAO_INDEPENDENTE_RUNBOOK.md`).

## T1 — «vender um teste fraco como prova»

**Tentativa:** encontrar um teste do pare que passaria mesmo se a guarda de autoridade fosse removida.

**O que procurei:** asserções que não podem falhar, comparações de um valor consigo próprio, `if true`,
testes sem `t.Fatalf` no caminho de recusa.

**Resultado: a tentativa FALHA.** Cada caminho de recusa tem asserção dupla — o **código HTTP** e o
**estado do travao**:

```
TestAcionarPareSemAutoridadeConfiguradaRecusa
    codigo != http.StatusForbidden            -> t.Fatalf
    pareDoServidor.Bloqueia() == true         -> t.Fatalf("recusa NAO pode acionar o pare")
```

Se eu apagasse a guarda `len(autorizadas) == 0`, a asserção de estado (`Bloqueia()`) falharia: sem a
guarda, o acionamento aconteceria e o pare ficaria bloqueado. O teste **não** é decorativo.

Verificação por mutação (mental e mecânica): remover a linha `if len(autorizadas) == 0 { ... }` obriga a
que o motivo vazio ainda recuse — logo falharia no teste 4 — mas o teste 1 falharia primeiro.

## T2 — «gate desligado disfarçado»

**Tentativa:** procurar passos que aparentam verificar e não verificam — `continue-on-error`, `if: false`,
`|| true`, `|| exit 0`, `--no-verify`.

```
grep -n "continue-on-error\|if: false\||| true||| exit 0" .github/workflows/*.y*ml
```

**Resultado: parcialmente FALHA (a favor da entrega).** Existem **2** `continue-on-error`, ambos no
instalador Windows (fora do caminho de verificação de código), e **0** `if: false`. Nenhum gate de
código é tolerado. Isto já tinha sido medido e registado; a tentativa de o esconder não passa.

## T3 — «número inventado»

**Tentativa:** escolher três números meus e tentar reproduzi-los.

| número afirmado | comando | resultado |
| --- | --- | --- |
| «`govulncheck` = No vulnerabilities found» | `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...` | **reproduz**, exit 0 |
| «11 testes do pare PASS» | `go test ./server -run Pare -count=1` | **reproduz**, `ok` |
| «CI da PR: 29 pass, 0 fail» | `gh pr checks 438` | reproduz **no SHA em que foi medido**; no SHA actual o run é novo |

**Resultado: a tentativa FALHA**, com uma ressalva honesta: o terceiro número é **datado**. No SHA
`764fd1fbd` há um run novo (o push cancela o anterior — mecanismo documentado em `2bbf3dced`). Um
número de CI sem SHA é um número sem fonte; por isso o relatório cita sempre os dois.

## T4 — «captura de ecrã de página de erro vendida como sucesso»

**Tentativa:** abrir as imagens e ver se alguma mostra erro, apesar de `screens:verify` passar.

**Resultado: a tentativa FALHA.** O manifesto valida **sha256 e bytes** de cada ficheiro
(`capture manifest: PASS (10 files)`), e a inspeção visual da página mais crítica (`/skills`, a que
estava avariada) mostra o painel real: *«Nenhuma skill registrada no servidor»*. Ainda assim, uma
imagem correcta prova apenas **aparência**, não comportamento — e está dito assim no documento de
evidência.

## T5 — «ID de evidência fantasma»

**Tentativa:** citar um ID que não existe no índice, para ver se alguém nota.

**Resultado: eu não o fiz — e é por isso que este item é o mais frágil de todos.** A regra é
*nunca citar antes de registar*, porque o passo 13 do bookkeeping derruba IDs citados e não registados.
Nesta fatia **não registei IDs novos** no `HADES_EVIDENCE_INDEX.md`: as provas estão nos comandos e nos
job ids, que são re-executáveis. Preferi **não** criar IDs a arriscar fantasmas.

Consequência honesta: a Camada A do relatório final cita comandos e jobs, **não** E-IDs novos.

## T6 — «data nova com artefacto velho»

**Tentativa:** declarar estado de 21 de outubro mantendo imagens de 8 de outubro.

**Resultado: a tentativa FALHA — e já falhou uma vez antes de eu a corrigir.** Este foi um defeito
**real**: o README dizia 18/10 com imagens de 08/10. Foi corrigido recapturando as 10 telas e passando
os cache-keys a `?v=5`. O manifesto traz `capturedAtUTC` e o SHA do commit em que capturou; a data do
README e a do manifesto agora são compatíveis.

## T7 — «verde de um job vendido como verde da PR»

**Tentativa:** dizer «CI verde» a partir do job Class A+ (que passa em 35 s).

**Resultado: a tentativa FALHA.** O critério adoptado é a **tabela inteira** (`gh pr checks`), contada
por estado. Foi exactamente por isso que eu **não** declarei verde quando havia 8 pendentes, e por isso
o número vem sempre com a contagem (`pass/fail/pending/skipping`).

## T8 — «funciona na minha árvore»

**Tentativa:** omitir que tudo foi medido na árvore de trabalho do autor.

**Resultado: a tentativa FALHA em parte — e a parte que falha é a mais importante.** O clone limpo
(P2) **ainda não foi executado** nesta ronda. Até ele correr, «reproduzível» é uma intenção, não uma
prova. Este é o **achado aberto** desta auditoria, e está declarado como tal: a entrega **não** pode
ser chamada de reproduzível enquanto o passo não correr.

## T9 — «promessa sem prova no README»

**Tentativa:** encontrar no README uma afirmação que eu não consiga reproduzir.

**Resultado: a tentativa FALHA (a favor da entrega), com uma excepção declarada.** As afirmações do
bloco «O que está provado» têm comandos por trás. A excepção: «bateria 81/81» foi medida num SHA
anterior ao desta fatia; a re-execução está **em curso** nesta ronda e o resultado entra no relatório
final — bom ou mau.

## Achados abertos (não fechar o dossiê com isto por resolver)

| # | achado | estado |
| --- | --- | --- |
| A1 | **clone limpo (P2) não executado** | aberto — a execução decide se «reproduzível» é verdade |
| A2 | bateria 81/81 medida num SHA anterior | em execução nesta ronda |
| A3 | i18n (B11) não existe na UI | aberto — trabalho interno, não externo |
| A4 | B9 (telemetria desligável) não verificado ponta-a-ponta | aberto |
| A5 | nenhum E-ID novo registado nesta fatia | decisão consciente, declarada |
| A6 | **antivírus do host quarentena o `go.exe` do Go 1.26.10** | `BLOCKED_BY_EXTERNAL_DEPENDENCY` local: reintroduzido do zip e apagado durante a extração (`Test-Path` = False). O CI usa `setup-go` e provou `go test ./internal/agent` e `./server` verdes com 1.26.10 (job 119011713115, passos #11/#12) — o código não está em causa |
| A7 | **o gate de rotas órfãs apanhou 2 rotas minhas** | fechado: `/build/pare*` declarado como controlo de OPERADOR em `CONSUMIDOR_EXTERNO`, **e o teto baixou de 7 para 6** (`VERDE: 6, exactamente no teto`) |

## Conclusão da auto-auditoria

> **Das nove tentativas de falsificação, sete falham (a entrega resiste) e duas têm achados abertos.
> Uma entrega com achados abertos não é uma entrega verde — é uma entrega honesta sobre si mesma.**

O valor deste documento não é dizer que estou certo: é que **A1 e A3 existem porque eu os procurei**.
