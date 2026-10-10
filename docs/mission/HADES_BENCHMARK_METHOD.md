# HADES — METODOLOGIA DE BENCHMARK REPRODUZÍVEL

> Artefato do **estágio 21**. Substitui, no que é mensurável localmente, o
> comparativo puramente documental que existia antes.
>
> **Limite declarado e inegociável:** este documento **não afirma superioridade**
> sobre Manus, Claude Code, Codex, Devin, OpenHands, Cursor, Lovable, Bolt, v0,
> Base44, n8n, Dify, Perplexity, CrewAI ou LangGraph. Uma comparação honesta
> exige a MESMA tarefa, a MESMA máquina e a licença/conta de cada produto — sem
> isso, qualquer número seria inventado. Aqui se mede apenas o custo
> determinístico dos motores que o Hades controla.

## 1. Comando reproduzível

```powershell
# ambiente: Go 1.27.2 (windows/amd64), host de desenvolvimento
go test ./internal/agent/ -run '^$' -bench 'Benchmark' -benchtime=300ms -count=1
```

- `-run '^$'` garante que **nenhum teste** roda: só benchmarks.
- `-benchtime=300ms` mantém a execução curta e estável para CI/local.
- `-count=1` evita cache e média entre execuções.
- Para perfil de alocação, qualquer benchmark aceita `-benchmem`.

Os benchmarks ficam em `internal/agent/benchmark_test.go` e são **ignorados**
pela suíte normal (`go test` sem `-bench`), então não atrasam o CI.

## 2. Ambiente desta medição

| Item | Valor |
| --- | --- |
| Sistema | Windows (host de desenvolvimento) |
| CPU | 13th Gen Intel(R) Core(TM) i9-13900HX |
| Toolchain | go1.27.2 windows/amd64 |
| Ponto de partida | `main@889750abe` + gerenciador de plugins (PR #78) |
| Iterações | `-benchtime=300ms -count=1` |

## 3. O que cada benchmark mede

| Benchmark | Mede | Não mede |
| --- | --- | --- |
| `BenchmarkMemorySearchContext` | busca por relevância em 500 memórias de um projeto (varredura + ordenação), sem embedder | qualidade semântica (exige modelo de embedding) nem latência de disco real de um banco externo |
| `BenchmarkPluginInstallUpdateRollback` | ciclo completo instalar → atualizar → rollback → remover **com persistência** | CPU puro: inclui I/O de diretório temporário por iteração |
| `BenchmarkPluginPromotionWithSignature` | verificação ed25519 + digest + persistência da promoção | confiança criptográfica além do que a política autoriza |
| `BenchmarkGroundedContextBuild` | montagem de contexto fundamentado com citações a partir de 32 fontes | geração de resposta pelo modelo (não há inferência aqui) |
| `BenchmarkUploadThroughput` (pré-existente) | vazão de upload do runtime | rede real |

## 4. Resultado desta medição

| Benchmark | ns/op | B/op | allocs/op |
| --- | --- | --- | --- |
| `MemorySearchContext` (500 memórias) | 363.037 | 221.406 | 6 |
| `PluginInstallUpdateRollback` (com I/O) | 8.078.241 | 26.814 | 173 |
| `PluginPromotionWithSignature` (com I/O) | 4.653.191 | 18.627 | 114 |
| `GroundedContextBuild` (32 fontes) | 20.512 | 15.368 | 76 |
| `UploadThroughput` | 364.339.800 | — | 287,80 MB/s |

Leitura honesta destes números:

1. **`MemorySearchContext` escala com o número de memórias** (varredura linear,
   sem índice vetorial). 363 µs para 500 memórias é aceitável para busca local
   interativa, mas é o candidato natural a índice/ANN quando o volume crescer —
   registrado como lacuna do estágio 15, não como virtude.
2. **Os dois benchmarks de plugin são dominados por I/O** (cada iteração cria
   diretório temporário e grava JSON). Eles provam que o caminho de persistência
   funciona e é mensurável; **não** devem ser citados como custo de CPU.
3. **`GroundedContextBuild` é barato** (20 µs): montar contexto com citações não
   é o gargalo do RAG; o gargalo é a recuperação e o modelo.

## 5. Como invalidar este resultado

Qualquer pessoa pode contestar a medição repetindo o comando em outra máquina:
os números são dependentes de hardware e de sistema de arquivos. Duas medições
com o mesmo commit e a mesma máquina devem ficar na mesma ordem de grandeza; se
não ficarem, o ambiente (antivírus, disco, carga) precisa ser registrado.

## 6. O que ainda falta para um comparativo competitivo

| Requisito | Situação | Bloqueio |
| --- | --- | --- |
| Mesma tarefa executada no Hades e em cada referência | não feito | exige instalar e licenciar cada produto |
| Métricas de tarefa (tarefas concluídas, bugs corrigidos, qualidade do artefato) | não feito | depende de inferência real (**B-07**) |
| Custo/tokens por tarefa | não feito | depende de chave de provedor (**B-07**) |
| Benchmarks de contexto e de memória exportar/retenção | pendentes neste arquivo | dependem dos PRs #75 (bootstrap) e #77 (governança de memória) |

Enquanto esses itens não tiverem evidência, o relatório de benchmark é
**parcial** e o estágio 21 permanece `PARCIAL`, sem nenhuma alegação de
supremacia.
