# Estágio 22 — a escrituração da missão passou a ser CONFERIDA por máquina (E-180)

## 1. O limite que a E-179 deixou

A E-179 fez o **gate** ser gerado e conferido. Mas ficou dito: *"cobre **apenas** os dois documentos que ele gera
— índice de evidências, relatório de estágios e bloqueios seguem **manuais** e podem divergir sem alarme"*.

O índice de evidências **é** manual, e continuará sendo (é prosa). O que não pode continuar manual é a **estrutura**
dele. A seção 3 exige que toda capacidade promovida tenha evidência anexada — e isso só vale se alguém conferir:

- um estágio pode citar `E-999` que **nunca existiu**;
- dois estágios podem disputar o **mesmo ID**;
- a numeração pode ter **buraco** (evidência que sumiu em alguma edição);
- um estágio pode apontar para um **bloqueio órfão**;
- e o veredito pode ser declarado completo **com pendência em aberto**.

Nada disso apareceria como falha. É exatamente o tipo de erro que uma missão longa acumula em silêncio.

## 2. O que foi feito

| Peça | O que confere |
| --- | --- |
| `scripts/hades_mission_check.py` | IDs do índice **únicos** e **sem buracos** (E-001..E-NNN); toda evidência citada por um estágio **existe**; todo `blocker` citado **está declarado**; `execution` **presente** e dentro do conjunto permitido pela seção 7; `release_state` permitido e **coerente** (`COMPLETED_VERIFIED` com estágio pendente ou bloqueio externo em aberto ⇒ **falha**); e o gate gerado continua consistente |
| `scripts/hades_mission_check_test.py` | **injeta os defeitos** em fixtures e exige que o verificador os encontre — **um verificador que sempre passa não vale nada** |
| `.github/workflows/class-a-plus-integrity.yaml` | dois passos novos: `Verify mission bookkeeping` e `Verify the mission bookkeeping checker itself (fault injection)` |

O verificador aceita `--state`/`--index` para poder rodar contra fixtures — foi o que tornou o defeito
**injetável** em vez de só descrito.

## 3. O verificador foi testado contra os defeitos que ele promete pegar

| Teste (fixture com defeito injetado) | Resultado |
| --- | --- |
| evidência **fantasma** (`E-999` citada por um estágio) | acusada |
| ID **duplicado** no índice | acusado |
| **buraco** na numeração (E-001, E-003) | acusado |
| **bloqueio órfão** (`B-77` inexistente) | acusado |
| `execution` **ausente** | acusado |
| `execution` **inválido** (`QUASE_PRONTO`) | acusado |
| veredito **otimista** (`COMPLETED_VERIFIED` com estágio pendente **e** bloqueio externo aberto) | acusado em **duas** frentes |
| fixture **saudável** | passa (não é falso positivo) |
| **índice real do repositório** | consistente |

**9 testes, ok.**

## 4. Estado real medido

```
índice: 179 evidências (E-001..E-179)
estágios: 38 · bloqueios declarados: 11
veredito: NOT_RELEASE_READY
ESCRITURAÇÃO CONSISTENTE: nenhuma evidência fantasma, nenhum bloqueio órfão, veredito coerente.
```

## 5. Limites declarados

- O verificador confere **estrutura**, não **verdade**: ele garante que a evidência `E-NNN` **existe** e está
  citada — **não** garante que o comando dela foi executado, nem que sustenta a afirmação. Isso continua sendo
  julgamento (e é o que o índice registra linha a linha).
- O índice **continua manual**: este script não o gera, só o confere. Gerar prosa não é o objetivo.
- Só o **índice de evidências** e o `HADES_MASTER_STATE.json` são conferidos; `HADES_STAGE_REPORT.md` e
  `HADES_BLOCKERS.md` continuam **fora** da checagem automática.
- A exigência de numeração **sem buracos** é deliberada, mas é uma **escolha**: um projeto que renumere evidências
  de propósito precisaria relaxar essa regra (e diria por quê).
