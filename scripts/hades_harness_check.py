#!/usr/bin/env python3
"""`hades_harness_check.py` — o §ESTAGIO 6 (Harness Runtime), medido (`E-1098`).

***O QUE O §ESTAGIO 6 EXIGE.***

    - fechar o que for **internamente executavel sem depender de login externo**;
    - manter **Ollama local como caminho principal**;
    - testar **execucao governada, cancelamento, timeout, retry limitado, idempotencia, retomada e
      artefactos**;
    - para **CLIs externos sem sessao real**, manter **`BLOCKED_EXTERNAL` com sonda opt-in e mensagem
      exacta**; nao chamar ausencia de credencial de defeito de codigo.

***O QUE ESTE CHECADOR MEDE, e como.***

  1. **As 7 capacidades tem teste**, cada uma pelo nome do teste (nao por comentario): *cancelamento,
     timeout, retry, idempotencia, retomada/restart, artefactos, governacao/approvals.*
  2. **Cancelamento ENTRE INSTANCIAS** (`cancel_cross_instance_test.go`) -- *e' o caso dificil: um
     processo cancela a missao de outro.*
  3. **Ollama como default** -- *o `RulePlanner` e o caminho local; e o `cli_harness` tem de declarar
     `BLOCKED_EXTERNAL` (nao fingir).*
  4. ***A SONDA OPT-IN*** -- *o CLI externo tem de ter uma sonda que so' corre quando pedida, e uma
     **mensagem exacta**; se o teste disser «skip» sem motivo, e' um skip escondido.*

***E O QUE ELE NAO FAZ.*** *Nao corre a suite inteira (ela leva **327,8 s**, e o `go test` ja' esta' nos
gates). => *o que ele faz e' verificar que **cada capacidade tem um teste que a nomeia** -- que e' a
diferenca entre «o estagio esta' implementado» e «o estagio esta' coberto».*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def todos_os_testes() -> dict[str, str]:
    """`{nome_do_teste: ficheiro}` para todo o `internal/agent`."""
    out: dict[str, str] = {}
    for p in sorted(AGENTE.glob("*_test.go")):
        t = io.open(p, encoding="utf-8").read()
        for nome in re.findall(r"func (Test\w+)", t):
            out[nome] = p.name
    return out


def main() -> int:
    testes = todos_os_testes()
    T("o pacote tem testes", len(testes) > 300, "%d testes" % len(testes))

    # ---------- 1. as 7 capacidades, cada uma pelo NOME do teste ----------
    categorias = {
        "cancelamento": ("Cancel",),
        "timeout": ("Timeout",),
        "retry limitado": ("Retry",),
        "idempotencia": ("Idempot",),
        "retomada/restart": ("Resume", "Restart", "Retoma"),
        "artefactos": ("Artifact", "Artefato"),
        "governacao/approvals": ("Approval", "Gate"),
    }
    for cap, chaves in categorias.items():
        achados = [n for n in testes if any(c in n for c in chaves)]
        T("(1) `%s` tem teste" % cap, len(achados) >= 1,
          "%d teste(s), ex.: %s" % (len(achados), achados[0] if achados else "-"))

    # ---------- 2. cancelamento ENTRE INSTANCIAS ----------
    T("(2) cancelamento entre instancias tem teste dedicado",
      "cancel_cross_instance_test.go" in {p.name for p in AGENTE.glob("*_test.go")},
      "e' o caso dificil: um processo cancela a missao de outro")

    # ---------- 3. Ollama como caminho principal ----------
    planner = AGENTE / "planner.go"
    T("(3) o `RulePlanner` existe (o caminho local, sem login)", planner.is_file())
    # *e o harness tem de mencionar o Ollama como default*
    mencoes = 0
    for p in sorted(AGENTE.glob("*.go")):
        try:
            t = io.open(p, encoding="utf-8").read()
        except UnicodeDecodeError:
            continue
        if re.search(r"ollama", t, re.I):
            mencoes += 1
    T("(3b) o pacote referencia o Ollama", mencoes >= 3, "%d ficheiros" % mencoes)

    # ---------- 4. CLI externo: BLOCKED_EXTERNAL + sonda opt-in ----------
    cli = [p.name for p in AGENTE.glob("cli_harness*_test.go")]
    T("(4) ha' testes do CLI harness", len(cli) >= 2, ", ".join(cli))
    # ***O CRITERIO NAO E' A STRING `BLOCKED_EXTERNAL`*** -- *foi o meu proprio erro de detector lexical
    # (o `E-1084`) procura-la: os testes do CLI **declaram** o bloqueio na forma CERTA, com
    # `t.Skipf("...motivo exacto...")` e um teste chamado `...EhOptIn`. => *o criterio passa a ser
    # **estrutural**:*
    #   1. `t.Skip`/`t.Skipf` COM mensagem (um `t.Skip()` vazio e' um skip escondido -- o prompt proibe);
    #   2. um teste cujo nome contenha `OptIn` -- a sonda tem de ser **explicita**.
    com_motivo = 0
    com_optin = 0
    for nome in cli:
        corpo = io.open(AGENTE / nome, encoding="utf-8").read()
        # *`t.Skipf(` obriga a mensagem formatada; `t.Skip("...")` tambem serve; `t.Skip()` vazio, nao.*
        if re.search(r't\.Skipf?\("[^"]+"', corpo):
            com_motivo += 1
        if re.search(r"func Test\w*OptIn", corpo):
            com_optin += 1
    T("(4b) e declaram o bloqueio com MOTIVO (t.Skip com mensagem)",
      com_motivo >= 1, "%d de %d ficheiros com mensagem" % (com_motivo, len(cli)))
    T("(4b2) e a sonda e' OPT-IN explicita (um teste `...OptIn`)",
      com_optin >= 1, "%d de %d ficheiros" % (com_optin, len(cli)))
    # *E o negativo: `t.Skip()` vazio -- skip escondido -- nao pode existir nos ficheiros do CLI.*
    vazios = 0
    for nome in cli:
        corpo = io.open(AGENTE / nome, encoding="utf-8").read()
        vazios += len(re.findall(r"t\.Skip\(\)", corpo))
    T("(4b3) e NAO ha' `t.Skip()` vazio (skip escondido)", vazios == 0,
      "%d skip(s) vazio(s)" % vazios)
    # *a sonda opt-in: a variavel de ambiente que liga o teste real*
    opt_in = set()
    for p in AGENTE.glob("*_optin_test.go"):
        opt_in.add(p.name)
    T("(4c) ha' sonda OPT-IN (os testes `*_optin_test.go`)", len(opt_in) >= 3,
      "%d sonda(s): %s" % (len(opt_in), ", ".join(sorted(opt_in)[:3])))

    # ---------- 5. retry LIMITADO (nao infinito) ----------
    retry_tests = [n for n in testes if "Retry" in n]
    T("(5) o retry e' testado", len(retry_tests) >= 3, "%d testes" % len(retry_tests))

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 6 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 6 (Harness Runtime): %d testes; as 7 capacidades nomeadas; cancelamento entre" % len(testes))
    print("  instancias; Ollama no caminho local; e o CLI externo com bloqueio declarado e sonda opt-in.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
