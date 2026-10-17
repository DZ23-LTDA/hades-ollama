# HADES — EVIDÊNCIA PERDIDA HISTÓRICA (`EVIDENCE_LOST_HISTORICAL`)

> **O que este documento é.** A marcação explícita, com **alcance e impacto**, das referências de evidência que
> foram **citadas** mas **nunca registadas** no índice — e que a arqueologia classificou como
> **irrecuperáveis**.
>
> **O que ele NÃO é.** Não é uma reconstrução. ***Não há texto inventado para preencher os números.*** O prompt é
> explícito: *«Não “reconstrua” conteúdo inexistente e não invente comandos/resultados retroativos»* e *«Não crie
> 134 textos fictícios para preencher números»*.

---

## 1. A dívida declarada, medida

O índice tem **902 entradas** (`E-001..E-1109`). A **faixa histórica** `E-571..E-777` — **207 IDs** — está
**ausente do índice** e está declarada como tal em:

- `docs/mission/HADES_MASTER_STATE.json` → `evidence_ids_nao_registrados_faixas: [[571, 777]]`;
- `scripts/hades_mission_check.py` (`E-778`): *a faixa é uma **FAIXA FECHADA**; um ID ausente **dentro** dela
  não é «buraco», é **dívida declarada**; um ID ausente **fora** dela **FALHA**».

⇒ **A catraca**: *a lista de tolerados **só pode diminuir**.* Um ID novo citado sem entrada **FALHA**.

### 1.1 Os 12 IDs citados fora da faixa-de-tolerância automática

*Estes 12 estão **dentro** de `E-571..E-777`, mas são **citados** por documentos e código — e por isso estão
listados nominalmente no estado (`evidence_ids_nao_registrados`), além da faixa.*

| ID | onde é citado | o que se perdeu | impacto | decisão |
|---|---|---|---|---|
| `E-618` | `HADES_EVIDENCE_INDEX.md:709`, `HADES_MASTER_STATE.json:4457/4480`, `HADES_PAPERCLIP_PARITY_MATRIX.md:208`, `HADES_PENDING_WORK.md:587`, `hades_mission_check.py:755/1001` | *o registo do que foi medido sobre a **paridade com o Paperclip*** | **médio**: a matriz de paridade cita-o como fonte, mas a fonte não existe | `EVIDENCE_LOST_HISTORICAL` |
| `E-619` | `HADES_CLOUDROOM_PARITY_MATRIX.md:5`, `HADES_MASTER_STATE.json:4458`, `HADES_PAPERCLIP_PARITY_MATRIX.md:6/50`, `HADES_PENDING_WORK.md:624` | *o registo da **paridade com o Cloudroom*** | **médio**: idem | `EVIDENCE_LOST_HISTORICAL` |
| `E-636` | `HADES_MASTER_STATE.json:4459`, `HADES_PENDING_WORK.md:657`, `hades_audit_instruments.py:32` | *o registo da **auditoria dos instrumentos*** | **alto**: o instrumento de auditoria cita-o | `EVIDENCE_LOST_HISTORICAL` |
| `E-642` | `HADES_MASTER_STATE.json:4460/4494`, `HADES_PENDING_WORK.md:646` | *o registo do **trabalho pendente** dessa janela* | **médio** | `EVIDENCE_LOST_HISTORICAL` |
| `E-650` | `HADES_EVIDENCE_INDEX.md:711/901`, `HADES_MASTER_STATE.json:4461/4485`, `HADES_PENDING_WORK.md:627/638` | *o registo de uma **medição** citada 4×* | **alto**: é o mais citado dos 12 | `EVIDENCE_LOST_HISTORICAL` |
| `E-651` | `HADES_MASTER_STATE.json:4462/4486/4487`, `HADES_PENDING_WORK.md:660` | *o registo de uma **medição** citada 3×* | **médio** | `EVIDENCE_LOST_HISTORICAL` |
| `E-653` | `HADES_MASTER_STATE.json:4463/4490` | *idem* | **baixo**: só o estado o cita | `EVIDENCE_LOST_HISTORICAL` |
| `E-654` | `HADES_MASTER_STATE.json:4464/4488` | *idem* | **baixo** | `EVIDENCE_LOST_HISTORICAL` |
| `E-655` | `HADES_MASTER_STATE.json:4465/4489` | *idem* | **baixo** | `EVIDENCE_LOST_HISTORICAL` |
| `E-656` | `HADES_MASTER_STATE.json:4466` | *idem* | **baixo** | `EVIDENCE_LOST_HISTORICAL` |
| `E-669` | `HADES_EVIDENCE_INDEX.md:826/836`, `HADES_MASTER_STATE.json:4467`, `HADES_PAPERCLIP_PARITY_MATRIX.md:223`, `HADES_PENDING_WORK.md:620`, `HADES_TEAMILY_PARITY_MATRIX.md:38` | *o registo da **paridade com o Teamily*** | **alto**: 3 documentos de paridade citam-no | `EVIDENCE_LOST_HISTORICAL` |
| `E-672` | `HADES_EVIDENCE_INDEX.md:710/712/713`, `HADES_MASTER_STATE.json:4468`, `HADES_TEAMILY_PARITY_MATRIX.md:46`, `hades_mission_check.py:746/836` | *o registo de uma **medição** citada 7× — e o **maior ID cunhado** antes de a faixa ser declarada* | **alto**: é o mais citado e o limite superior da era paralela | `EVIDENCE_LOST_HISTORICAL` |

### 1.2 Os 2 placeholders

| ID | o que é | decisão |
|---|---|---|
| `E-000` | *placeholder de «sem evidência»* | `PLACEHOLDER` — **não é evidência perdida** |
| `E-999` | *placeholder de «evidência futura»* | `PLACEHOLDER` — **não é evidência perdida** |

---

## 2. O alcance: o que a perda **não** afeta

⇒ **Nenhum requisito ativo depende de um destes 12.** *Medido: os `REQ-01..10` (`docs/audit/REQUISITOS.csv`)
apontam para **código, teste, comando e resultado medidos agora** — e o `hades_requisitos.py --check` verifica
as **10** colunas de cada um.*

⇒ *E as **matrizes de paridade** (Cloudroom, Paperclip, Teamily) **citam** estes IDs como fonte histórica —
mas a classificação de paridade delas **não foi promovida por causa disso**: o `HADES_COMPETITIVE_CAPABILITY_MATRIX.csv`
marca as referências correspondentes como `UNVERIFIABLE`, **não** `PARITY_MEASURED`.*

---

## 3. A decisão, e o que ela custa

| | |
|---|---|
| **O que se faz** | *marcar como `EVIDENCE_LOST_HISTORICAL`, com o alcance acima* |
| **O que NÃO se faz** | *reconstruir o texto, inventar comandos, ou «fechar» os números* |
| **O que se preserva** | *a faixa `E-571..E-777` declarada, e a catraca que impede novos buracos* |
| **O custo** | *os documentos que citam `E-618/619/636/642/650/651/653/654/655/656/669/672` têm uma **fonte que não existe** — e a partir de hoje **sabem-no*** |
| **A alternativa rejeitada** | *inventar 207 registos seria **fabricar prova** — o pior defeito possível num projeto que exige homologação honesta* |

⇒ ***`um ID perdido que está marcado como perdido é uma dívida; um ID perdido que foi preenchido com texto inventado é uma mentira.`***

---

## 4. Como isto é verificado

O `scripts/hades_traceability_check.py` mede:

1. **a faixa histórica continua declarada** (`[[571, 777]]` no estado);
2. **os 12 IDs citados estão listados** em `evidence_ids_nao_registrados`;
3. **cada um dos 12 tem uma linha nesta tabela** (o documento não é opcional);
4. **os placeholders `E-000`/`E-999` estão separados** dos perdidos;
5. **nenhum requisito ativo (`REQ-*`) citado neste documento como dependente de um ID perdido**;
6. **a catraca**: o número de tolerados **não cresce** face ao commit anterior.

⇒ *e o teste discriminador (`hades_traceability_check_test.py`) injeta 3 defeitos: remover um dos 12 do estado,
remover uma linha da tabela, e acrescentar um ID **novo** citado sem entrada.*
