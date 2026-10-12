# Estágio 22 — relatório de estágios e bloqueios entram na checagem (E-181)

## 1. O limite que a E-180 deixou

A E-180 declarou, com todas as letras: *"`HADES_STAGE_REPORT.md` e `HADES_BLOCKERS.md` seguem **fora** da checagem
automática"*. Estender a checagem a esses dois documentos **não** foi formalidade: ela encontrou **dois problemas
reais** que estavam no repositório, e um **falso positivo meu** — os três ficam registrados abaixo.

## 2. Os dois problemas reais que a extensão encontrou

| Documento | Problema encontrado | Consequência |
| --- | --- | --- |
| `HADES_STAGE_REPORT.md` | **item 127 DUPLICADO** (139 itens numerados para números 1..138, dois deles "127") — o mesmo item da E-169 tinha sido inserido **duas vezes**, com textos quase idênticos | numeração ambígua: citar "a rodada 127" não identifica uma rodada |
| `HADES_BLOCKERS.md` | **B-11 e B-12 declarados no estado e AUSENTES do documento** | impedimento registrado que **ninguém lê** — o oposto do propósito do documento |

Correções: removido o item 127 mais curto (mantida a versão completa; agora **138 itens, 1..138, sem duplicata**) e
acrescentadas as linhas de **B-11** e **B-12** (agora **11 bloqueios nos dois lados**).

## 3. O falso positivo — e a regra que o corrigiu

A primeira execução acusou: *"relatório cita evidências que NÃO existem no índice: **E-999**"*. Era **falso
positivo**: `E-999` aparecia no texto **do meu próprio item da E-180**, explicando *o que o verificador pega*.
Medição: no relatório inteiro, **1** ID aparece entre crases (`E-999`, o exemplo) e **110** aparecem entre
parênteses (as citações reais).

Regra adotada: **ID entre crases é MENÇÃO (exemplo/identificador), não CITAÇÃO** — o texto entre crases é removido
antes da varredura. É uma decisão **declarada**, com o custo dela: uma citação real escrita dentro de crases
**não** seria verificada.

## 4. O que passou a ser conferido

| Documento | Checagens novas |
| --- | --- |
| `HADES_STAGE_REPORT.md` | itens `N. **` **únicos** e **contíguos** a partir de 1; toda evidência `E-NNN` **citada** existe no índice (ignorando crases) |
| `HADES_BLOCKERS.md` | paridade com o estado **nos dois sentidos**: bloqueio declarado e ausente do documento é problema; linha no documento sem bloqueio no estado também |

## 5. O verificador continua sendo TESTADO contra os defeitos que promete pegar

`scripts/hades_mission_check_test.py` foi de **9 para 19 testes**: item duplicado, buraco na numeração, evidência
fantasma **em texto normal**, menção entre crases **não** virando problema, relatório sem itens, paridade de
bloqueios nos dois sentidos, e os documentos **reais** consistentes. **19 testes ok.**

## 6. Verificação

| Comando | Resultado |
| --- | --- |
| `python scripts/hades_mission_check.py` | **exit 0** — 180 evidências, 138 itens (até 138), 11 bloqueios nos dois lados |
| `python scripts/hades_mission_check_test.py` | **19 testes ok** |
| `python scripts/hades_release_gate.py --check` | consistente |

## 7. Limites declarados

- A checagem do relatório confere **numeração e citações** — **não** o conteúdo: um item pode estar mal escrito,
  repetido em substância ou descrever algo que não foi feito, e passar.
- A remoção do item 127 duplicado foi decidida por **tamanho** (manteve-se o texto mais completo). Um item pode
  ter sido perdido se as duas versões diziam coisas diferentes — a diferença observada era cosmética (crases e
  caixa), mas isso é **julgamento meu**, não medição.
- A regra "ID entre crases é menção" é uma **escolha**: citação real em crases escaparia da verificação.
- `HADES_BLOCKERS.md` é conferido por **paridade de IDs**; o texto de cada bloqueio (procedimento de desbloqueio)
  continua **manual** e não é validado.
