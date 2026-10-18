# README DO AUDITOR — HADES

**Este pacote é gerado.** Não edite à mão: corrija a fonte e regenere com
`python -X utf8 scripts/hades_audit_package.py`.

## O que este repositório afirma, e onde

| Alegação | Onde vive |
| --- | --- |
| **Veredito**: `NOT_RELEASE_READY` | `docs/mission/HADES_MASTER_STATE.json` (`release_state`) |
| **1024 entradas** de evidência, de `E-001` a `E-1231` | `docs/mission/HADES_EVIDENCE_INDEX.md` |
| **84 gates** declarados | `scripts/hades_local_gates.ps1` |
| **18 bloqueios** | `LIMITACOES.md` (gerado do estado) |
| **Faixa histórica** `571..777` | declarada no estado |

## Como verificar, em três comandos

```powershell
cd <este checkout>
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_local_gates.ps1
python -X utf8 scripts/hades_audit_package.py --check
```

O segundo comando confirma que este pacote **bate com o repositório**. *Se bater, o pacote
foi gerado deste estado; se não bater, alguém mudou o repositório sem regenerar.*

## O que você **NÃO** vai encontrar aqui

- **nao prova que o CI esta' verde** -- isso e' do SHA, e o SHA muda a cada commit;
- **nao prova homologacao em dispositivo** -- nao houve dispositivo;
- **nao prova revisao humana independente** -- nao houve;
- **nao prova que o produto e' release** -- o veredito e' `NOT_RELEASE_READY`;
- **nao prova superioridade competitiva** -- nenhuma capacidade esta' `SUPERIOR_MEASURED`;
- **nao executa os gates** -- lista-os; quem os corre e' o auditor;

## O que este pacote **não** contém, e por que

- **as 207 evidências sem texto**: não foram reconstruídas (§FASE-7 item 2). O
  `PARIDADE_EVIDENCIAS.csv` lista as que **existem**; as da faixa histórica não estão nele
  porque não têm registo;
- **os artefactos de `dist/`**: nao vem no repositorio, portanto nao sao medidos; o
  `LIMITACOES.md` nomeia-os um a um com a proveniencia declarada;
- **o resultado dos gates**: `GATES.md` lista-os, não os executa. *Um pacote que dissesse
  «passam» estaria a afirmar o que não mediu.*

## Aviso sobre a natureza deste pacote

**Ele é honesto sobre o que não sabe**, e essa é a única qualidade que um pacote de auditoria
pode ter. *Um pacote que só lista o que existe é uma brochura; um que diz o que falta é um
instrumento.*
