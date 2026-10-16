# HADES — MATRIZ T2 (medida)

**Gerado por:** `TestMatrizRealComModelosLocais` · **Data:** 2026-10-16T23:41:07Z

Medição de **RED → GREEN** em fixtures próprios, com harness mínimo sobre Ollama local. Harness externo (Claude Code/Codex/Manus) exige credencial ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒ `NAO_MEDIDO`. **Nenhuma alegação de paridade** sai deste arquivo.

| modelo | fixture | red | green | tentativas | hunks | linhas | duração (s) | veredito |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen2.5:7b | fixture-t2-soma | 1 | 0 | 2 | 1 | 2 | 38.2 | GREEN |
| qwen2.5:7b | fixture-t2-comparacao | 1 | 1 | 2 | 0 | 0 | 39.7 | HARNESS_TRAFICOU |
| qwen2.5:7b | fixture-t2-caso-limite | 1 | 1 | 2 | 0 | 0 | 43.8 | HARNESS_TRAFICOU |
| qwen2.5:7b | fixture-t2-fatiamento | 1 | 1 | 2 | 0 | 0 | 63.9 | HARNESS_TRAFICOU |
| llama3.2:3b | fixture-t2-soma | 1 | 0 | 1 | 1 | 1 | 15.4 | GREEN |
| llama3.2:3b | fixture-t2-comparacao | 1 | -1 | 2 | 0 | 0 | 6.6 | HARNESS_TRAFICOU |
| llama3.2:3b | fixture-t2-caso-limite | 1 | -1 | 2 | 0 | 0 | 362.9 | RED |
| llama3.2:3b | fixture-t2-fatiamento | 1 | 0 | 1 | 1 | 1 | 14.8 | GREEN |

**Resumo medido:** total 8 · verde 3 · verde na 1ª tentativa 2 · trapaceou 4 · permaneceu vermelho 1 · **taxa 3/8**

**Limites:** modelos locais apenas; `veredito=HARNESS_TRAFICOU` significa que o modelo tentou alterar o arquivo de TESTE (o anti-trapaça registra e o laço continua com feedback); `green=-1` significa **não executado** (ausência de medição não é aprovação).
