# HADES — MATRIZ T2 (medida)

**Gerado por:** `TestMatrizRealComModelosLocais` · **Data:** 2026-10-14T01:48:45Z

Medição de **RED → GREEN** em fixtures próprios, com harness mínimo sobre Ollama local. Harness externo (Claude Code/Codex/Manus) exige credencial ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` ⇒ `NAO_MEDIDO`. **Nenhuma alegação de paridade** sai deste arquivo.

| modelo | fixture | red | green | tentativas | hunks | linhas | duração (s) | veredito |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| qwen2.5:7b | fixture-t2-soma | 1 | 0 | 2 | 1 | 2 | 11.3 | GREEN |
| qwen2.5:7b | fixture-t2-comparacao | 1 | 1 | 2 | 0 | 0 | 9.2 | HARNESS_TRAFICOU |
| qwen2.5:7b | fixture-t2-caso-limite | 1 | 1 | 2 | 0 | 0 | 10.5 | HARNESS_TRAFICOU |
| qwen2.5:7b | fixture-t2-fatiamento | 1 | 1 | 2 | 0 | 0 | 10.6 | HARNESS_TRAFICOU |
| llama3.2:3b | fixture-t2-soma | 1 | 0 | 1 | 1 | 1 | 9.5 | GREEN |
| llama3.2:3b | fixture-t2-comparacao | 1 | -1 | 2 | 0 | 0 | 6.0 | HARNESS_TRAFICOU |
| llama3.2:3b | fixture-t2-caso-limite | 1 | -1 | 2 | 0 | 0 | 362.0 | RED |
| llama3.2:3b | fixture-t2-fatiamento | 1 | 0 | 1 | 1 | 1 | 14.7 | GREEN |

**Resumo medido:** total 8 · verde 3 · verde na 1ª tentativa 2 · trapaceou 4 · permaneceu vermelho 1 · **taxa 3/8**

**Limites:** modelos locais apenas; `veredito=HARNESS_TRAFICOU` significa que o modelo tentou alterar o arquivo de TESTE (o anti-trapaça registra e o laço continua com feedback); `green=-1` significa **não executado** (ausência de medição não é aprovação).
