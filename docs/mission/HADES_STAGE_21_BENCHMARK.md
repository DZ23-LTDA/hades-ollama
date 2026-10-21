# Estágio 21 — benchmark comparável, com métricas MEDIDAS

## 1. O que o estágio pede, e o que é honesto entregar

O estágio pede avaliar tarefas concluídas, precisão, correção, tempo, custo, estabilidade,
recuperação, segurança, intervenções humanas, acessibilidade e usabilidade, **sem alegar
superioridade sem evidência**.

Executar Manus, Claude Code, Codex, Devin, Cursor, Lovable, Bolt, v0, Base44, n8n, Dify,
Perplexity, CrewAI ou LangGraph exigiria **contas, credenciais e custo** que este ambiente não
tem. Sem executá-los, qualquer comparação seria **invenção**. Então a suíte mede o que é
medível aqui e **declara a comparação externa como `NÃO EXECUTADA`**, com o motivo no próprio
relatório — o veredito nunca é "somos melhores".

## 2. O que passou a ser executável

`internal/agent/benchmark_suite.go` executa **cinco tarefas herméticas de ponta a ponta** no
próprio Hades e mede, por tarefa: **conclusão**, **passos de API**, **duração real**,
**aprovações humanas exigidas**, **prova de isolamento** entre organizações e **prova de
retomada/recuperação**. O relatório é um artefato JSON (`HADES_BENCHMARK_OUT`).

O teste `TestBenchmarkSuiteMeasuresHermeticTasks` é o executor: exige que **todas** as tarefas
concluam e falha nomeando a tarefa se alguma **regredir** — é o que dá sentido a acompanhar
números em vez de narrar desempenho.

## 3. Medição real (executada nesta máquina)

| Tarefa | Concluiu | Passos | Duração | Aprovações humanas | Isolamento | Retomada |
| --- | --- | --- | --- | --- | --- | --- |
| `jornada_empresa_sandbox` | sim | 1 | 41 ms | 1 | sim | sim |
| `pedido_idempotente` | sim | 7 | 12 ms | 0 | — | — |
| `atendimento_sla_handoff` | sim | 6 | 12 ms | 1 | — | — |
| `isolamento_entre_organizacoes` | sim | 5 | 10 ms | 0 | sim | — |
| `retomada_apos_reinicio` | sim | 8 | 27 ms | 0 | — | sim |

- **Veredito: `MEDIDO`** — **5/5** tarefas concluídas.
- **Duração total: 102 ms**.
- Ambiente: `windows/amd64 · go go1.27.2`.
- Comparação externa: **`NAO_EXECUTADA`**.

O que cada tarefa prova:

1. **Jornada de empresa em sandbox**: veredito `SANDBOX_APROVADO`, **21 passos**, **isolamento e
   recuperação provados**, **1 aprovação humana** exigida (a decisão de risco não é automática).
2. **Pedido idempotente**: mesma chave devolve o **mesmo** pedido, **referência parecida com
   cartão é recusada** e o pagamento válido é aceito — 1 pedido, 1 pago, receita 10 000 centavos.
3. **Atendimento com SLA e transferência**: resposta dentro do SLA não gera violação, resposta
   fora **é medida** (`sla_breached = 1`), transferência **sem motivo é recusada** e com motivo a
   conversa passa a `pending_human`.
4. **Isolamento entre organizações**: a organização B vê **0** produtos, **0** conversas e **0**
   métricas da A.
5. **Retomada após reinício**: um runtime **novo**, do mesmo diretório de dados, enxerga o
   estado (1 produto, 1 pedido) e **repetir a chave devolve o MESMO pedido** — o efeito não
   duplica.

## 4. O que este relatório NÃO afirma

- **Não** afirma que o Hades é melhor que qualquer concorrente: eles não foram executados.
- **Não** estima custo de tokens nem latência de modelos externos: exigiria credenciais.
- **Não** cobre acessibilidade e usabilidade por tarefa: essas têm evidência própria (E-099,
  axe/WCAG e o teste de acessibilidade), não uma métrica desta suíte.
- As durações são de execução **hermética local** (sem GPU, sem modelo externo), então medem o
  **motor** (comércio, inbox, isolamento, persistência), não a qualidade de geração de texto.

## 5. Verificação

- `go test ./internal/agent/ -run TestBenchmarkSuite -count=1` → **ok** (5/5 tarefas).
- Relatório medido: `_tmp/benchmark-report.json` (artefato citado em E-117).
- `go test ./internal/agent/` → **ok** · `go test ./server/` → **ok** · `golangci-lint` →
  **0 issues** (o falso positivo do `misspell` em "COMPETITIVO" foi **reescrito**, não silenciado).

---

## 6. Artefactos MEDIDOS deste estagio (indice, com comando)

*Um metodo que nao aponta para o artefacto e' um metodo que o auditor nao segue.* Os tres
documentos do estagio descreviam as medicões; **nenhum chegava ao resultado**. Esta secção liga
os dois -- e cada linha traz o comando que a reproduz.

| artefacto | o que contem | comando | evidencia |
| --- | --- | --- | --- |
| `docs/parity/HADES_T2_MATRIX.md` | matriz **RED -> GREEN** com **modelos reais**: 3/8 verde, **4 `HARNESS_TRAFICOU`**, 1 vermelho | `$env:HADES_T2_MEDIR_MODELO='1'; go test ./internal/agent/ -run 'TestMatrizRealComModelosLocais' -count=1 -timeout 30m` | `E-1047` (*`PASS` em 599,88 s*) |
| `docs/parity/HADES_T2_REPETICAO.md` | **repeticao estatistica**: `qwen2.5:7b`, 3 execucoes por fixture | `$env:HADES_T2_MEDIR_MODELO='1'; $env:HADES_T2_MODELS='qwen2.5:7b'; $env:HADES_T2_REPEATS_N='3'; go test ./internal/agent/ -run 'TestT2RepeticaoEstatistica' -count=1 -timeout 40m` | `E-1049` (*`PASS` em 209,78 s*) |
| `internal/agent/harness_benchmark.go` | o harness e o **anti-trapaca** (`:286`) | -- | `E-1047` |
| `cmd/bench` | medicao directa do Ollama (`load`/`ttft`/`prefill`/`generate`) | `go run ./cmd/bench -model llama3.2:3b -epochs 3 -max-tokens 200 -format csv` | `E-1047` |

### O que estes artefactos NAO provam

- **`NAO_EXECUTADA` para a comparacao externa**: executar Manus, Claude Code, Codex, Devin,
  Cursor, Lovable, Bolt, v0, Base44, n8n, Dify, Perplexity, CrewAI ou LangGraph exigiria
  **contas, credenciais e custo** que este ambiente nao tem. *Sem os executar, qualquer
  comparacao seria invencao.*
- **A taxa e' 3/8** com modelos locais de **3-7B**: *um modelo, um host, uma quantizacao.*
- **`N=3` por fixture** na repeticao: *contagem, nao probabilidade; nao ha' intervalo de confianca.*
- **O warm/cold do `cmd/bench` nao existe**: `unloadModel` (`bench.go:504`) corre **antes** de
  medir -- toda corrida e' fria por construcao. O par real esta' **dentro** da corrida, por
  epoca (92,2 / 89,7 / 91,3 tok/s -- **0,98x**): *o instrumento mede estabilidade, nao warm/cold.*
