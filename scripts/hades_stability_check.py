#!/usr/bin/env python3
"""`hades_stability_check.py` — o §ESTAGIO 2 (Estabilizacao e consolidacao da base), medido (`E-1108`).

***O QUE O §ESTAGIO 2 EXIGE.*** *O titulo e' «Estabilizacao e consolidacao da base», e o `next` do estado dizia:
«**Flake do macOS em ./server (E-197)**: obter o NOME do teste que falhou (o log do MCP trunca) e verificar a
pista de que o job roda com `-bench=. -benchtime=1x`».*

***O QUE FOI MEDIDO (e o que este checador mede).***

  1. **O flake do macOS: `NAO_REPRODUZIDO`** => *o `E-197` registou um `FAIL` no attempt 1 e um `SUCCESS` no
     attempt 2 do **mesmo SHA** (`68ee67b5a`) -- um flake classico* => *mas nos **6 SHAs seguintes** o
     `test (macos-latest)` passou **sempre*** => *e o **comando exacto do CI**
     (`go test -count=1 -bench=. -benchtime=1x ./server/`) passa no Windows em **51,3 s*** => *o flake **nao se
     reproduziu**; o honesto e' `NAO_REPRODUZIDO`, nao `CORRIGIDO`.*
  2. **A base e' HERMETICA (com divida declarada)** => *o `hades_testes_hermeticos_check.py` mede as chamadas
     `NewRuntime(RuntimeConfig{...})` sem `DataRoot`: **107 de 192** (85 com `DataRoot`) => *a baseline e'
     **107**, e o criterio e' que **nao cresca** => *divida conhecida e estavel, nao escondida.*
  3. **A base e' DETERMINISTA na codificacao** => *o `hades_encoding_check.py` mede **2574 ficheiros**, **4
     dispensas declaradas** (o detector, a fixture dele, o checker e a fixture).*
  4. **As catracas EXISTEM e nao regridem** => *o `hades_approval_hook_ratchet.py` e o
     `hades_bridge_optin_ratchet.py`, cada um com o seu teste.*
  5. **O `go test` do pacote mais instavel passa** => *`./server` com o comando do CI: **ok 4,762s** (medido).*

***E A LICAO***: *«um flake que nao se reproduz nao esta' corrigido -- esta' **latente**» => *logo o criterio
honesto e' `NAO_REPRODUZIDO` com a **janela** em que foi observado, e nao «resolvido».*
"""
import io
import json
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr(script: str) -> tuple[int, str]:
    p = SCRIPTS / script
    if not p.is_file():
        return 2, "AUSENTE"
    r = subprocess.run([sys.executable, "-X", "utf8", str(p)],
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    # ---------- 1. o flake do macOS: NAO_REPRODUZIDO ----------
    # *o criterio NAO e' «o flake esta' corrigido» (isso exigiria reproduzi-lo) -- e' que ele foi
    # **observado uma vez**, e que a **janela seguinte** esta' limpa.*
    ev = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"
    T("(1) o `E-197` (o flake do macOS) esta' registado", ev.is_file() and "| E-197 |" in io.open(ev, encoding="utf-8").read(),
      "o flake foi OBSERVADO: FAIL no attempt 1 e SUCCESS no attempt 2 do mesmo SHA")
    estado = RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json"
    if estado.is_file():
        d = json.loads(io.open(estado, encoding="utf-8").read())
        # *procurar a declaracao do flake no estado*
        txt = json.dumps(d, ensure_ascii=False)
        T("(1b) e o estado declara o flake como `NAO_REPRODUZIDO` (nao como corrigido)",
          "NAO_REPRODUZIDO" in txt or "nao reproduz" in txt.lower(),
          "o honesto e' `NAO_REPRODUZIDO`, nao `CORRIGIDO`")

    # ---------- 2. a base e' HERMETICA, com divida declarada ----------
    rc, saida = correr("hades_testes_hermeticos_check.py")
    m = re.search(r"TESTES NAO-HERMETICOS:\s*(\d+)", saida)
    n_nao = int(m.group(1)) if m else -1
    T("(2) a hermeticidade e' MEDIDA (o `hades_testes_hermeticos_check.py` corre)", rc == 0,
      "nao-hermeticos=%d (a baseline nao pode crescer)" % n_nao)
    T("(2b) e a divida NAO cresceu (baseline 107)", n_nao == 107,
      "n=%d" % n_nao if n_nao >= 0 else "nao medido")

    # ---------- 3. a base e' DETERMINISTA na codificacao ----------
    rc, saida = correr("hades_encoding_check.py")
    m = re.search(r"(\d+)\s*arquivo", saida)
    n_enc = int(m.group(1)) if m else -1
    T("(3) a codificacao e' MEDIDA e limpa", rc == 0, "%d ficheiros" % n_enc)

    # ---------- 4. as catracas existem e tem teste ----------
    catracas = sorted(p.name for p in SCRIPTS.glob("*ratchet*.py"))
    T("(4) as catracas existem", len(catracas) >= 4, "%d ficheiros" % len(catracas))
    sem_teste = [c for c in catracas if not c.endswith("_test.py")
                 and (SCRIPTS / c.replace(".py", "_test.py")).is_file() is False]
    T("(4b) e cada catraca tem o seu teste", not sem_teste,
      "todas" if not sem_teste else str(sem_teste))

    # ---------- 5. o pacote mais instavel passa com o COMANDO DO CI ----------
    # *o `./server` e' o que o `E-197` viu falhar => *e o criterio e' que ele passe com
    # `-count=1 -bench=. -benchtime=1x` (o comando do job do CI).*
    T("(5) o `./server` passa com o comando EXACTO do CI",
      _server_passa(),
      "`go test -count=1 -bench=. -benchtime=1x ./server/`")

    # ---------- 6. o estagio 2 no estado ----------
    if estado.is_file():
        est = d.get("stages") or []
        s2 = next((s for s in est if s.get("id") == 2), None)
        if s2:
            T("(6) o estagio 2 esta' `IN_PROGRESS` (a consolidacao tem divida declarada)",
              s2.get("execution") == "IN_PROGRESS",
              "os %d testes nao-hermeticos SAO a divida que impede fechar" % n_nao)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 2 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 2 (Estabilizacao): o flake do macOS esta' `NAO_REPRODUZIDO`, a base e'")
    print("  hermética com divida declarada (%d), a codificacao e' limpa (%d ficheiros)," % (n_nao, n_enc))
    print("  as catracas tem teste, e o `./server` passa com o comando EXACTO do CI.")
    print("")
    print("  ***UM FLAKE QUE NAO SE REPRODUZ NAO ESTA' CORRIGIDO -- ESTA' LATENTE.*** *E o honesto e'")
    print("  `NAO_REPRODUZIDO` com a janela em que foi observado, nao «resolvido».*")
    return 0


def _server_passa() -> bool:
    """Corre o comando do CI no `./server` (o pacote que o `E-197` viu falhar)."""
    go = RAIZ / "_tmp" / "goroot1269" / "go" / "bin" / "go.exe"
    if not go.is_file():
        go = Path("go")
    # ***O `env` TEM DE SER O AMBIENTE INTEIRO, NAO UM DICIONARIO NOVO.*** *Medido: passar um `env` so' com
    # as minhas variaveis **substitui** o ambiente -- e o Windows precisa do `SystemRoot`, `windir`,
    # `COMSPEC`, etc. para o `go` sequer arrancar => *o (5) dizia «nao passa» quando o comando passa.*
    import os as _os
    env = dict(_os.environ)
    env.update({
        "GOROOT": str(RAIZ / "_tmp" / "goroot1269" / "go"),
        "GOPATH": str(RAIZ / "_tmp" / "gopath"),
        "GOCACHE": str(RAIZ / "_tmp" / "gocache"),
        "GOTMPDIR": str(RAIZ / "_tmp" / "gotmp"),
        "TMP": str(RAIZ / "_tmp" / "webtemp"),
        "TEMP": str(RAIZ / "_tmp" / "webtemp"),
        "APPDATA": str(RAIZ / "_tmp" / "appdata"),
        "LOCALAPPDATA": str(RAIZ / "_tmp" / "localappdata"),
        "GOPROXY": "off", "GOSUMDB": "off", "GOPRIVATE": "*",
        "CGO_ENABLED": "0", "GOOS": "windows", "GOARCH": "amd64",
        "PATH": str(Path(go).parent) + ";" + _os.environ.get("PATH", ""),
    })
    try:
        r = subprocess.run([str(go), "test", "-count=1", "-bench=.", "-benchtime=1x", "./server/"],
                           cwd=str(RAIZ), env=env, capture_output=True, text=True,
                           encoding="utf-8", errors="replace", timeout=900)
    except (OSError, subprocess.TimeoutExpired):
        return False
    return r.returncode == 0


if __name__ == "__main__":
    sys.exit(main())
