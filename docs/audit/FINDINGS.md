# FINDINGS do PR #438 — como foram lidos, e o que NÃO se afirma

**Este ficheiro é gerado de `FINDINGS.json`**, que por sua vez foi capturado da GitHub API em
**2026-10-17T06:03:49Z** sobre o checkout `093390737`.

**Total: 14** — **5 Major** · **9 Minor**.

**Validados contra o código: 6 de 14** — #1 (VALIDADO_JA_CORRIGIDO), #2 (VALIDADO_JA_CORRIGIDO), #3 (VALIDADO_JA_CORRIGIDO), #4 (VALIDADO_JA_CORRIGIDO), #5 (VALIDADO_JA_CORRIGIDO), #6 (VALIDADO_JA_CORRIGIDO).

## O que este ficheiro NÃO afirma

- **não afirma que algum finding foi corrigido** — nenhum foi marcado como resolvido no GitHub;
- **não afirma que um finding do bot é verdadeiro** — *o prompt diz: comentário de bot é
  hipótese, não facto*; validar exige confrontar com o código e criar contraprova;
- **não afirma que a lista está completa** — são os **review comments** que a API devolveu
  naquele instante; *uma thread sem comentário de linha não apareceria aqui.*

## Como validar um finding, em vez de o aceitar

O `E-1070` e o `E-1074` são o modelo: **reproduzir o defeito num checkout limpo**, medir, e só
então decidir. *«Comentário de bot como hipótese não confiável: confirme no código atual, crie
contraprova e só então corrija.»*

## Validados contra o código

### #1 — `scripts/hades_test_clock_check_test.py` — **VALIDADO_JA_CORRIGIDO**

*«Use a mesma lista de triagem no teste e no `--check`.»*

- usos(linha) devolve o CONTEXTO de cada .Add( -- dois usos na mesma linha sao classificados em separado (E-909(a))
- usos() devolve TODOS, nao o primeiro -- uma variavel com dois deslocamentos expoe o de negocio (E-909(b))
- TRIAGADOS indexado por assinatura(linha), o texto sem espacos e sem comentario -- reindentar nao reabre, alterar reabre (E-909(c))
- sem_comentarios(linhas) remove // e /* */ preservando o numero de linha
- o _test.py importa hades_test_clock_check as checker e usa checker.TRIAGADOS -- uma fonte, nao duas allowlists
- 12/12 testes OK (1.3 s); repo real: 112 sitios, ALTO triagado 1, ALTO NOVO 0
- CONTRAPROVA: um break no usos() (so o primeiro uso) faz falhar test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha -- "'ALTO' not found in ['timestamp']"

**Evidência**: `E-1080`.

### #2 — `scripts/hades_test_clock_check.py` — **VALIDADO_JA_CORRIGIDO**

*«Classifique cada uso de relógio separadamente.»*

- usos(linha) devolve o CONTEXTO de cada .Add( -- dois usos na mesma linha sao classificados em separado (E-909(a))
- usos() devolve TODOS, nao o primeiro -- uma variavel com dois deslocamentos expoe o de negocio (E-909(b))
- TRIAGADOS indexado por assinatura(linha), o texto sem espacos e sem comentario -- reindentar nao reabre, alterar reabre (E-909(c))
- sem_comentarios(linhas) remove // e /* */ preservando o numero de linha
- o _test.py importa hades_test_clock_check as checker e usa checker.TRIAGADOS -- uma fonte, nao duas allowlists
- 12/12 testes OK (1.3 s); repo real: 112 sitios, ALTO triagado 1, ALTO NOVO 0
- CONTRAPROVA: um break no usos() (so o primeiro uso) faz falhar test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha -- "'ALTO' not found in ['timestamp']"

**Evidência**: `E-1080`.

### #3 — `scripts/hades_test_clock_check.py` — **VALIDADO_JA_CORRIGIDO**

*«Ignore comentários antes de procurar expressões.»*

- usos(linha) devolve o CONTEXTO de cada .Add( -- dois usos na mesma linha sao classificados em separado (E-909(a))
- usos() devolve TODOS, nao o primeiro -- uma variavel com dois deslocamentos expoe o de negocio (E-909(b))
- TRIAGADOS indexado por assinatura(linha), o texto sem espacos e sem comentario -- reindentar nao reabre, alterar reabre (E-909(c))
- sem_comentarios(linhas) remove // e /* */ preservando o numero de linha
- o _test.py importa hades_test_clock_check as checker e usa checker.TRIAGADOS -- uma fonte, nao duas allowlists
- 12/12 testes OK (1.3 s); repo real: 112 sitios, ALTO triagado 1, ALTO NOVO 0
- CONTRAPROVA: um break no usos() (so o primeiro uso) faz falhar test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha -- "'ALTO' not found in ['timestamp']"

**Evidência**: `E-1080`.

### #4 — `scripts/hades_test_clock_check.py` — **VALIDADO_JA_CORRIGIDO**

*«Continue a busca após o primeiro `.Add(` da variável.»*

- usos(linha) devolve o CONTEXTO de cada .Add( -- dois usos na mesma linha sao classificados em separado (E-909(a))
- usos() devolve TODOS, nao o primeiro -- uma variavel com dois deslocamentos expoe o de negocio (E-909(b))
- TRIAGADOS indexado por assinatura(linha), o texto sem espacos e sem comentario -- reindentar nao reabre, alterar reabre (E-909(c))
- sem_comentarios(linhas) remove // e /* */ preservando o numero de linha
- o _test.py importa hades_test_clock_check as checker e usa checker.TRIAGADOS -- uma fonte, nao duas allowlists
- 12/12 testes OK (1.3 s); repo real: 112 sitios, ALTO triagado 1, ALTO NOVO 0
- CONTRAPROVA: um break no usos() (so o primeiro uso) faz falhar test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha -- "'ALTO' not found in ['timestamp']"

**Evidência**: `E-1080`.

### #5 — `scripts/hades_test_clock_check.py` — **VALIDADO_JA_CORRIGIDO**

*«Vincule a triagem ao conteúdo inspecionado, não apenas à linha.»*

- usos(linha) devolve o CONTEXTO de cada .Add( -- dois usos na mesma linha sao classificados em separado (E-909(a))
- usos() devolve TODOS, nao o primeiro -- uma variavel com dois deslocamentos expoe o de negocio (E-909(b))
- TRIAGADOS indexado por assinatura(linha), o texto sem espacos e sem comentario -- reindentar nao reabre, alterar reabre (E-909(c))
- sem_comentarios(linhas) remove // e /* */ preservando o numero de linha
- o _test.py importa hades_test_clock_check as checker e usa checker.TRIAGADOS -- uma fonte, nao duas allowlists
- 12/12 testes OK (1.3 s); repo real: 112 sitios, ALTO triagado 1, ALTO NOVO 0
- CONTRAPROVA: um break no usos() (so o primeiro uso) faz falhar test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha -- "'ALTO' not found in ['timestamp']"

**Evidência**: `E-1080`.

### #6 — `internal/agent/service_agent_approvals.go` — **VALIDADO_JA_CORRIGIDO**

*«Protect every per-organization approval write with the same lock.»*

- as 6 funcoes de escrita tem withFileLock (SetApprovers:321, Save:367, expireStale:473, Request:635, Decide:715, Consume:875)
- saveLocked aplica preservarConsumidas(:380) -- a invariante «o consumo e' terminal»
- 3 testes discriminadores PASS (6.7 s)
- CONTRAPROVA: sem preservarConsumidas o teste FALHA com «REGRESSAO: Save gravou o instantaneo antigo e reabriu um pedido»

**Evidência**: `E-1078`.

## Por severidade

### Major (5)

| # | Ficheiro | Título |
| --- | --- | --- |
| 2 | `scripts/hades_test_clock_check.py`:144 | Classifique cada uso de relógio separadamente. |
| 4 | `scripts/hades_test_clock_check.py` | Continue a busca após o primeiro `.Add(` da variável. |
| 5 | `scripts/hades_test_clock_check.py`:269 | Vincule a triagem ao conteúdo inspecionado, não apenas à linha. |
| 6 | `internal/agent/service_agent_approvals.go`:879 | Protect every per-organization approval write with the same lock. |
| 10 | `internal/multillm/router_pin_v2_test.go` | Remove the word that fails the test checks. |

### Minor (9)

| # | Ficheiro | Título |
| --- | --- | --- |
| 1 | `scripts/hades_test_clock_check_test.py` | Use a mesma lista de triagem no teste e no `--check`. |
| 3 | `scripts/hades_test_clock_check.py`:220 | Ignore comentários antes de procurar expressões. |
| 7 | `docs/mission/HADES_ADAPTIVE_COWORK_GAP_MATRIX.md` | Alinhe as linhas da tabela ao cabeçalho. |
| 8 | `docs/mission/HADES_ADAPTIVE_COWORK_GAP_MATRIX.md` | Corrija a descrição de `PreferredProvider`. |
| 9 | `docs/mission/HADES_EVIDENCE_INDEX.md` | Escape the pipe in the table cell. |
| 11 | `internal/agent/declared_categories_measured_test.go` | Correct the description of the dispensation check. |
| 12 | `internal/agent/scheduling_library_declared_test.go` | Detect constructor references without a direct call. |
| 13 | `internal/agent/types.go`:149 | Update the scope note. It no longer matches the code. |
| 14 | `scripts/hades_mission_check.py` | Do not classify partially closed findings as closed. |

## O que fazer a seguir

**Nenhum destes findings foi corrigido por este trabalho.** O `E-1076` entregou o pacote de
auditoria; os findings são **entrada** para a revisão humana, não saída de uma correcção
automática. *Um pacote que dissesse «corrigido» sem um teste que discrimina o defeito estaria
a inventar prova.*
