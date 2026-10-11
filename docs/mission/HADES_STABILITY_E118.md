# Estágio 2/22 — intermitência: campanha sob carga, causa endurecida e uma hipótese REFUTADA (E-138)

## 1. O que estava pendente

Desde E-118 eu vinha declarando: *"UMA execução das suítes completas reportou falha
(`TestStudioDeployRejectsWithoutCredentialsHonestGateStatus` no servidor); o teste passa isolado e nas 5
execuções seguintes as suítes passaram inteiras — a falha não foi reproduzida"*. Declarado não é
resolvido. Esta rodada tentou **reproduzir sob carga** e **endurecer a causa**.

## 2. Campanha de estabilidade (medida, não estimada)

Script `_tmp/stability_campaign.ps1` (não versionado), com o bloco de ambiente completo:

| Fase | Execuções | Resultado |
| --- | --- | --- |
| **1** — suítes completas de `internal/agent` e `server` **em paralelo** (contenção de CPU/IO) | **5 rodadas** (5 + 5 execuções de suíte) | **0 falhas**; durações 57,9 s – 108,4 s por rodada (a rodada 5, mais lenta, é o efeito da carga) |
| **2** — martelo no teste de E-118: 4 instâncias simultâneas, `-count=10` cada | **40 execuções** do teste | **0 falhas** (`PASS=10` em cada um dos 4 arquivos) |

**A falha NÃO foi reproduzida**: somando as duas fases, 9 execuções de suíte completa e 40 execuções do
teste específico, todas verdes.

**Como o resultado foi verificado (e um defeito no meu próprio instrumento):** o script tentou registrar
`ExitCode` das execuções e esses campos saíram **vazios**. Então o resultado **não** foi aceito por
código de saída: foi conferido nos **artefatos** — as saídas das suítes contêm
`ok github.com/ollama/ollama/internal/agent` e `ok github.com/ollama/ollama/server`, e os arquivos do
martelo contêm `--- PASS` dez vezes cada. Um campo vazio teria virado "sucesso" se eu não tivesse ido
conferir o arquivo — e isso fica registrado como cuidado, não como detalhe.

## 3. O endurecimento (teste, não produto)

O teste de E-118 engolia **todos** os erros de preparação:

```go
builder, _ := agent.NewBuilderService(tmpDir)
project, _ := builder.Create(...)
runtime, _ := agent.NewRuntime(...)
```

Consequência real: se o setup falhar (por exemplo sob pressão de disco/IO), o runtime inválido produz
**panic** — falha sem causa legível, que é exatamente o formato opaco relatado em E-118. Agora cada erro
é verificado com `t.Fatalf` e o motivo aparece.

As asserções também foram endurecidas: antes o teste aceitava **qualquer** código diferente de 200
(inclusive 404 ou 500), o que não distingue *"recusou honestamente"* de *"não achou a rota"* nem de
*"quebrou"*. Agora ele **exige 501** e **exige** o motivo declarado
(`no deployment providers are configured`), mantendo a regra central: nunca 200/202 fingindo publicação.

## 4. A hipótese que a sonda REFUTOU (registrado por honestidade)

Minha primeira justificativa escrita foi: *"com o ID vazio a rota não casa, e o teste passaria sem
exercitar o portão — falso verde"*. Isso era **plausível e errado**. Escrevi uma **sonda temporária**
(`server/zz_probe_false_pass_test.go`, executada e depois **removida** — `git status` limpo) que chamou a
rota com **ID vazio** e mediu a resposta:

```
codigo com ID vazio = 501 | assercao ANTIGA (!=200) passaria = true | assercao NOVA (==501) passaria = true
--- FAIL: TestProbeFalsePassHazard
```

Ou seja: **com ID vazio a resposta é 501 de qualquer forma**, porque o handler recusa antes de consultar
o projeto. Portanto o risco **não** era "teste que não testa nada". O comentário no código foi corrigido
para dizer a verdade, e o ganho real ficou sendo outro (e continua valendo): **nenhuma falha de setup
passa em silêncio**, e a recusa honesta passou a ser **exigida exatamente**, não apenas "diferente de
200". A sonda falhar foi o resultado útil: ela custou minutos e evitou que eu registrasse uma conclusão
bonita e falsa no relatório.

## 5. Verificação

| Comando | Resultado |
| --- | --- |
| `gofmt -l server/builder_studio_routes_test.go` | **sem saída** (formatado) |
| `go test ./server/ -run TestStudio -count=1` | **ok** |
| `go test ./server/ -count=1 -timeout 30m` | **ok** (18,487 s) |
| `golangci-lint run ./server/...` | **0 issues** |
| campanha de estabilidade | **0 falhas** em 9 execuções de suíte + 40 execuções do teste |

## 6. Limites declarados

- **A intermitência segue não reproduzida** (5 rodadas paralelas + 40 execuções). O que mudou é que uma
  falha futura nesse teste, se acontecer, chega com **causa legível** em vez de panic.
- **O padrão de erros engolidos não foi varrido no repositório todo**: no próprio
  `server/builder_studio_routes_test.go` há outras **12** ocorrências de `, _ :=` em preparação de testes.
  Endurecer todas é trabalho mecânico e de baixo risco, adequado a uma rodada própria — fica declarado em
  vez de esquecido.
- **O instrumento da campanha tinha o defeito do `ExitCode` vazio**; a correção do script não foi
  refeita porque a verificação por artefato é mais forte que o código de saída.
