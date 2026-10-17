#!/usr/bin/env python3
"""`hades_testes_hermeticos_check.py` — a catraca dos testes NAO-HERMETICOS (`E-1094`).

***A CLASSE DO DEFEITO, VISTA 3 VEZES.***

    E-1087  server/agent_companion_routes_test.go   NewRuntime sem DataRoot => %APPDATA%   => FAIL
    E-1092  server/builder_studio_routes_test.go    NewRuntime sem DataRoot => %APPDATA%   => FAIL (4 testes)
    E-1094  internal/agent/runtime_test.go          NewRuntime sem DataRoot => %APPDATA%   => FAIL
    E-1094  internal/agent/workspace_write_test.go  os.UserCacheDir()      => %LOCALAPPDATA% => FAIL (2 testes)

=> *curar cada um **nao** cura a classe: o proximo teste com `NewRuntime` sem `DataRoot` volta a falhar num
ambiente confinado, e ninguem sabe porquê.* => ***esta catraca fecha a classe.***

***O QUE ELA VERIFICA.***

  1. Toda chamada `NewRuntime(RuntimeConfig{...})` num `*_test.go` tem `DataRoot:` no bloco do literal.
     *Sem ele, o `resolveRuntimeDataRoot` cai no `%APPDATA%` do utilizador.*
  2. Toda chamada `NewRuntime(agent.RuntimeConfig{...})` idem.

***E O QUE ELA **NAO** FAZ.*** *Nao exige `DataRoot` em codigo de **producao** -- ai' o `%APPDATA%` e' o
comportamento **certo**. => *a regra e' dos **testes**: um teste que escreve na maquina do utilizador nao e'
um teste, e' uma visita.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DIRS = [RAIZ / "server", RAIZ / "internal", RAIZ / "cmd", RAIZ / "app"]

# *A forma do inicio da chamada.*
INICIO = re.compile(r"(?:agent\.)?NewRuntime\((?:agent\.)?RuntimeConfig\{")


def blocos(texto: str):
    """Devolve (linha_inicial, texto_do_bloco) de cada `NewRuntime(RuntimeConfig{...})`."""
    for m in INICIO.finditer(texto):
        ini = m.start()
        # *contar as chaves a partir do `{` do literal*
        i = texto.index("{", m.start())
        prof = 0
        j = i
        while j < len(texto):
            c = texto[j]
            if c == "{":
                prof += 1
            elif c == "}":
                prof -= 1
                if prof == 0:
                    break
            j += 1
        bloco = texto[ini:j + 1]
        linha = texto.count("\n", 0, ini) + 1
        yield linha, bloco


def main() -> int:
    check = "--check" in sys.argv
    problemas: list[str] = []
    total = 0

    for d in DIRS:
        if not d.is_dir():
            continue
        for p in sorted(d.rglob("*_test.go")):
            try:
                texto = io.open(p, encoding="utf-8").read()
            except UnicodeDecodeError:
                continue
            for linha, bloco in blocos(texto):
                total += 1
                if "DataRoot:" not in bloco:
                    rel = p.relative_to(RAIZ)
                    # *o numero da linha e' do inicio da chamada; e' o que o autor procura.*
                    problemas.append("%s:%d" % (rel, linha))

    # ***A BASELINE DECLARADA.*** *Medida no `E-1094`: 107 chamadas de `NewRuntime` em testes sem
    # `DataRoot`. O CI passa (o `%APPDATA%` e' gravavel la'); o **processo confinado deste host** e' que
    # falha. => a divida fica **declarada com tecto**, e a catraca **falha se o numero SUBIR** -- um teste
    # novo sem `DataRoot` nao entra.* => ***`uma divida declarada com tecto nao e' o mesmo que uma divida
    # escondida.`***
    BASELINE = 107
    if len(problemas) > BASELINE:
        print("  CRESCIMENTO DE DIVIDA: %d chamadas sem DataRoot (baseline declarada: %d)" % (
            len(problemas), BASELINE))
        for pr in problemas:
            print("    - %s" % pr)
        print("")
        print("  *Um teste NOVO sem `DataRoot` nao entra: acrescente `DataRoot: t.TempDir(),`.*")
        return 1
    if problemas:
        print("  TESTES NAO-HERMETICOS: %d (baseline declarada: %d -- NAO cresceu)" % (
            len(problemas), BASELINE))
        poucos = len(problemas) <= 5
        if poucos:
            for pr in problemas:
                print("    - %s" % pr)
        print("  *Divida conhecida e estavel; %d chamadas totais, %d com DataRoot.*" % (
            total, total - len(problemas)))
        if len(problemas) < BASELINE:
            print("  *E BAIXOU de %d para %d -- baixe a BASELINE no codigo para travar o ganho.*" % (
                BASELINE, len(problemas)))
        return 0

    print("  TESTES HERMETICOS: %d chamadas de NewRuntime em testes, todas com DataRoot" % total)
    return 0


if __name__ == "__main__":
    sys.exit(main())
