#!/usr/bin/env python3
"""E-422 — Checker: TESTE QUE USA O RELÓGIO DA MÁQUINA COMO INSTANTE DE NEGÓCIO.

**A CLASSE, MEDIDA DUAS VEZES EM PRODUÇÃO**: um teste que deriva um **instante de negócio** (agendamento,
vencimento de lembrete, janela) de `time.Now()` **passa 23 horas por dia e falha na 24ª**.
  - E-417: `scheduling_test.go:17` — `agora := Now()+24h`, `agora.Add(90min)` ⇒ *às 23:41 cai em 01:11 do dia
    seguinte, fora de um turno `00:00–24:00`*;
  - E-421: `scheduling_waitlist_test.go:187` — **deixou o CI vermelho** ✗ — `agora := Now()`, `agora.Add(90min)`
    ⇒ *às 22:23 cai em 23:53, e o serviço termina depois da meia-noite*.

**POR QUE ESTE CHECKER EXISTE**: a varredura do E-418 (feita à mão, descartável) procurou **`time.Now()` seguido
de `.Add(...)` na MESMA linha** — e por isso **não viu** a forma em duas linhas
(`agora := time.Now()` … `agora.Add(...)`), que é exatamente a do E-421. ***O ponto cego não era do produto: era
do instrumento.*** Este checker cobre **as duas formas**.

O QUE ELE **NÃO** É: ele não tenta provar que o instante é perigoso. Ele **triagem** — aponta o sítio, e a
inspeção decide. *Excesso de zelo aqui custa uma leitura; falta de zelo custa um CI vermelho de madrugada.*

CLASSIFICAÇÃO (por que alguns usos são legítimos):
  - **vencimento** (`ClaimDue(now+2min)`, `NextRunAt = now-1min`): *"está vencido?"* não depende do calendário;
  - **timestamp** (`CreatedAt: now`, `UpdatedAt`, `ExpiresAt`): registra quando aconteceu, não escolhe quando;
  - **`now` puro**: passado como "agora" para uma função que compara;
  - **instante de negócio** (o que este checker procura): um `StartsAt`/`marcado`/`alvo`/`slot` **derivado** do
    relógio e deslocado — *é este que cruza fronteiras de calendário.*

USO: python3 scripts/hades_test_clock_check.py [--check]
  --check  sai != 0 se houver sítio de ALTO risco (para a bateria)
"""

import re
import sys
from pathlib import Path

# Nomes de variável/campo que indicam INSTANTE DE NEGÓCIO (escolhido, não observado).
NEGOCIO = (
    "StartsAt", "marcado", "alvo", "slot", "NextRunAt", "RunAt", "Due", "due",
    "janela", "livre", "abertura", "fechamento", "ExpiresAt",
)
# Formas que tornam o uso legítimo, mesmo com deslocamento.
SEGURO = (
    "-time.Second", "-time.Minute", "-time.Hour", "-1*time.Minute", "-2*time.Hour",
    "Truncate(24 * time.Hour)", "time.Date(",
)
# 🚨 E-422 (MEDIDO pelo próprio teste do checker): **consulta de VENCIMENTO** não é instante de negócio.
# `ClaimDue(now+1s)` e `ExpiresAt = now-1s` perguntam *"já venceu?"* — e a resposta **compara instantes**, não
# dias de calendário ⇒ *um `+1s` nunca cruza uma fronteira de dia de forma que mude a resposta*. O meu
# classificador marcava `ClaimDue(Now+Add(time.Second))` como ALTO ✗ porque `Due` estava na lista de negócio e o
# deslocamento era **positivo** (só os negativos estavam em SEGURO). *Um verificador que acusa vencimento obriga a
# inspecionar falso positivo toda vez — e falso positivo repetido ensina a ignorar o instrumento.*
VENCIMENTO = ("ClaimDue", "claimDue", "IsDue", "isDue", "vencido", "Vencido", "due(", "Due(", "Expira", "expira")
# Campos que são claramente observação, não escolha.
OBSERVACAO = ("CreatedAt", "UpdatedAt", "ObservedAt", "RecordedAt", "At:", "at:")

DIRETO = re.compile(r"time\.Now\(\)[^\n;]*?\.Add\(")
VARIAVEL = re.compile(r"(\w+)\s*:=\s*time\.Now\(\)")


def classificar(linha: str, var: str | None = None) -> str:
    if any(s in linha for s in SEGURO):
        return "vencimento/janela-segura"
    # E-422: vencimento tem PRECEDÊNCIA sobre "nome de negócio" — `ClaimDue(now+1s)` pergunta "já venceu?", e a
    # resposta compara instantes, não dias.
    if any(v in linha for v in VENCIMENTO):
        return "vencimento"
    if any(o in linha for o in OBSERVACAO) and not any(n in linha for n in ("StartsAt", "marcado", "alvo", "slot")):
        return "timestamp"
    if any(n in linha for n in NEGOCIO):
        return "ALTO"
    return "baixo"


def varrer(raiz: Path) -> list[tuple[str, int, str, str]]:
    achados = []
    for arq in sorted(raiz.rglob("*_test.go")):
        linhas = arq.read_text(encoding="utf-8").split("\n")
        for i, linha in enumerate(linhas):
            # Forma 1: Now()...Add(...) na MESMA linha.
            if DIRETO.search(linha):
                achados.append((arq.as_posix(), i + 1, classificar(linha), linha.strip()[:120]))
                continue
            # Forma 2 (o ponto cego do E-418): variavel := Now(); ... variavel.Add(...) ate 14 linhas depois.
            m = VARIAVEL.search(linha)
            if not m:
                continue
            var = m.group(1)
            for j in range(i + 1, min(len(linhas), i + 15)):
                if re.search(r"\b" + re.escape(var) + r"\.Add\(", linhas[j]):
                    achados.append((arq.as_posix(), j + 1, classificar(linhas[j], var), linhas[j].strip()[:120]))
                    break
    return achados


# E-422: sítios de ALTO já **inspecionados e medidos** como legítimos. *Um instrumento que acusa sempre não serve
# de gate* — a lista existe para que o `--check` reprove **o que é NOVO**, e para que cada aceite carregue a razão.
# NUNCA acrescente aqui só para o teste passar: meça primeiro e escreva a medição.
TRIAGADOS = {
    ("internal/agent/declared_categories_measured_test.go", 36):
        # E-432: era 28. *Reescrever o cabeçalho daquele teste moveu o sítio para 36* — e foi esta catraca que acusou
        # o movimento. **Pinar NÚMERO DE LINHA é brittleness conhecida**: o sítio continua sendo a **organização
        # vazia**, onde o portão de ESCOPO dispara ANTES do expediente, então o instante não decide nada.
        "a organizacao e' \"\" - o portao de ESCOPO dispara ANTES do expediente, entao o instante nao decide nada",
    ("internal/agent/scheduling_waitlist_test.go", 187):
        "e' a linha do COMENTARIO do E-421 (o codigo ja' chama instanteDeTeste)",
}


def main() -> int:
    check = "--check" in sys.argv
    raiz = Path("internal")
    if not raiz.is_dir():
        print("  (sem internal/ aqui — rode da raiz do repo)")
        return 0
    achados = varrer(raiz)
    altos = [a for a in achados if a[2] == "ALTO"]
    novos = [a for a in altos if (a[0], a[1]) not in TRIAGADOS]
    print("  sitios com relogio deslocado em testes: %d" % len(achados))
    for arq, linha, classe, trecho in achados:
        if classe != "ALTO":
            continue
        razão = TRIAGADOS.get((arq, linha))
        if razão:
            print("    ✅ %s:%d  [ALTO TRIAGADO]  %s" % (arq, linha, razão))
        else:
            print("    🔴 %s:%d  [ALTO NOVO]  %s" % (arq, linha, trecho))
    print("  ALTO triagado: %d | ALTO NOVO: %d" % (len(altos) - len(novos), len(novos)))
    print("  NOTA: ALTO nao prova defeito - e' TRIAGEM. Vencimento/timestamp/'now' puro nao entram porque o")
    print("        resultado nao depende do calendario.")
    if check and novos:
        print("  FALHA: %d sitio(s) de ALTO NAO TRIAGADO(S) - inspecione e, se for instante de negocio, use" % len(novos))
        print("         instanteDeTeste(t) (E-421); se for legitimo, acrescente a TRIAGADOS com a razao MEDIDA.")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
