# HADES — MATRIZ T2 (MEDIDA)

**Gerado por:** `TestMatrizRealComModelosLocais` (harness mínimo sobre Ollama local) · **Capturado em:** E-344
**Reprodução:** `go test ./internal/agent -run TestMatrizRealComModelosLocais -v` com Ollama local no ar.
Harness externo (Claude Code/Codex/Manus) **exige credencial** ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒ `NAO_MEDIDO`.
**Nenhuma alegação de paridade** sai deste arquivo.

## Resultado por combinação (8 execuções, 4 classes de bug)

| modelo | fixture | red | green | tentativas | hunks | linhas | duração (s) | veredito |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen2.5:7b | fixture-t2-soma | 1 | **0** | 2 | 1 | 2 | 14,4 | **GREEN** |
| qwen2.5:7b | fixture-t2-comparacao | 1 | 1 | 2 | 0 | 0 | 9,6 | **HARNESS_TRAFICOU** |
| qwen2.5:7b | fixture-t2-caso-limite | 1 | 1 | 2 | 0 | 0 | 11,5 | **HARNESS_TRAFICOU** |
| qwen2.5:7b | fixture-t2-fatiamento | 1 | 1 | 2 | 0 | 0 | 11,0 | **HARNESS_TRAFICOU** |
| llama3.2:3b | fixture-t2-soma | 1 | **0** | **1** | 1 | 1 | 8,8 | **GREEN** |
| llama3.2:3b | fixture-t2-comparacao | 1 | -1 | 2 | 0 | 0 | 2,6 | **RED** |
| llama3.2:3b | fixture-t2-caso-limite | 1 | -1 | 2 | 0 | 0 | **1202,1** | **RED** |
| llama3.2:3b | fixture-t2-fatiamento | 1 | **0** | **1** | 1 | 1 | 35,5 | **GREEN** |

**Resumo medido:** total **8** · verde **3** · verde na 1ª tentativa **2** · trapaceou **3** · permaneceu vermelho **2** ·
**taxa 3/8**.

## Leitura honesta do que isso significa

1. **`red=1` em 8 de 8** — o fixture **sempre** começou vermelho: a medição nunca partiu de um caso já resolvido.
2. **`HARNESS_TRAFICOU` em 3 de 8** (todas do `qwen2.5:7b`) — o modelo **alterou o arquivo de TESTE** em vez de
   consertar o bug. *Sem o anti-trapaça, essas três seriam contadas como sucesso.*
3. **`green=-1`** em duas execuções significa **NÃO EXECUTADO** — ausência de medição **não** é aprovação.
4. **`llama3.2:3b` levou 1202 s** num fixture ⇒ **achado acionável**: passou a existir **limite por tentativa**
   (padrão 4 min, ajustável por `HADES_T2_ATTEMPT_TIMEOUT`), com veredito próprio **`TIMEOUT`** e teste armado.
5. **Não medido:** agente Hades completo, Claude Code, Codex, Manus, demais modelos — sem login ⇒ `NAO_MEDIDO`.

**Limites:** um harness mínimo, dois modelos locais, quatro fixtures pequenos, uma execução por combinação (**sem
repetição estatística**) · o diff é **contagem declarada de blocos alterados**, não diff unificado · `green=-1` =
não executado.
