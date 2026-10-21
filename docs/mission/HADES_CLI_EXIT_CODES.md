# HADES CLI -- Codigos de Saida e Contrato de Automacao

> FASE 8 do adendo V3. Este documento descreve o contrato OBSERVADO da CLI, nao um
> contrato desejado. Cada linha abaixo foi verificada por teste em `cmd/`.

## Por que este documento existe

O adendo exige "codigos de saida documentados". Documentar depois de implementar seria
documentar a intencao; documentar junto com o teste e' documentar o comportamento.

A pergunta que este contrato responde e' a que uma automacao faz: **"terminou bem?"**.
`0` significa bem. Qualquer outro valor significa que algo tem de ser lido em `stderr`.

## Contrato geral (todos os comandos)

| codigo | significado | quem decide |
| --- | --- | --- |
| `0` | o comando fez o que pediu | `RunE` devolveu `nil` |
| `1` | falha; a mensagem esta' em `stderr` | `cobra.CheckErr` (padrao do projeto) |

**Este e' o unico contrato que vale para TODA a CLI.** Qualquer comando que queira um
codigo diferente tem de declarar `ExitCode() int` no erro devolvido; `main.go` honra a
interface de forma **aditiva** -- nenhum comando existente a implementa, por isso nenhum
mudou de comportamento.

> *Um contrato novo que muda o comportamento dos consumidores antigos nao e' um
> contrato: e' uma quebra.*

## `ollama agent doctor` -- codigos proprios

`doctor` e' o unico comando que hoje declara `ExitCode()`, porque e' o unico cuja
pergunta e' "em que estado esta' o endpoint?" e nao "passou ou falhou?".

| codigo | veredicto | significado |
| --- | --- | --- |
| `0` | `saudavel` | todos os checks responderam; nada a fazer |
| `2` | `degradado` | o check ESSENCIAL respondeu, mas algum nao-essencial falhou |
| `1` | `inutilizavel` | o check essencial NAO respondeu; nao ha agente utilizavel |

`2` existe por um motivo concreto: um cliente que so' distingue `0` de `1` trata
"o agente vive mas o catalogo de ferramentas esta' quebrado" como catastrofe. Nao e'.

### Checks

| check | rota | essencial |
| --- | --- | --- |
| saude do agente | `GET /api/agent/v1/health` | **sim** |
| catalogo de ferramentas | `GET /api/agent/v1/tools` | nao |
| catalogo de presets | `GET /api/agent/v1/presets` | nao |

### Rota ausente nao e' falha

Um servidor mais antigo que nao exponha `/presets` responde `404`. Isso e' classificado
como `indisponivel`, **nao** como `degradado` -- salvo se a rota for essencial.

> *Inventar uma falha para uma sonda que nao existe seria um diagnostico que mente.*

### O token nunca aparece no relatorio

Se `OLLAMA_AGENT_TOKEN` estiver definido, o relatorio diz que esta' definido e o valor
**nao e' lido para o relatorio**. Um diagnostico que imprime o segredo que deveria
proteger e' pior do que nao diagnosticar.

## Saida `--json`

`--json` e' **persistente** no comando `agent`:

| sem `--json` | com `--json` |
| --- | --- |
| JSON indentado, legivel por humano | **uma linha** compacta, para `jq` e scripts |

> *Um `--json` que continua indentado nao e' JSON para automacao: e' JSON bonito.*

## Erros acionaveis

Um erro que nao diz o que fazer obriga o utilizador a ler o codigo. Os comandos novos
seguem a regra: se o id nao existe, **nomear o id** e indicar o comando que lista os
validos.

Exemplo verificado: `agent preset show nao-existe` devolve um erro que contem o id
pedido e a instrucao `agent preset list`.

## O que este documento NAO cobre

- `server/` (API HTTP) tem os seus proprios codigos; este doc e' sobre a CLI.
- Superficies `goal`, `worktree`, `terminal`, `approvals`, `runtime`, `ci` ainda nao
  estao implementadas na CLI. Quando forem, o contrato de exit code delas pertence a
  este documento.
- Nao ha' garantia de estabilidade para flags de comandos marcados como experimentais.

## Referencia de implementacao

- `cmd/agent.go` -- `agent doctor`, `doctorErro`, constantes `exitSaudavel` /
  `exitDegradado` / `exitInutilizavel`.
- `main.go` -- honra `interface{ ExitCode() int }` de forma aditiva.
- `cmd/agent_doctor_test.go` -- 7 testes; os discriminadores sao
  `TestDoctorSaudavelSaiComZero`, `TestDoctorDegradadoSaiComDois` e
  `TestDoctorSemSaudeSaiComUm`.
