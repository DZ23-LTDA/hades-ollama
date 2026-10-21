"""`hades_firefox_e2e.py` — a extensao **INSTALA e ARRANCA** num Firefox REAL (`E-1122`).

***O QUE O `E-1116` TINHA.*** *O pacote Firefox com o `web-ext lint` em **0/0/0** (o lint do proprio Mozilla). Mas
o lint **nao instala** -- e o `E-1116` declarou `INSTALL=nao` e `E2E=nao`.*

***O QUE ESTE SCRIPT FAZ, e como.*** *Nao ha' Firefox nativo neste host* => *mas ha' o **Firefox do Playwright**
(`firefox-1555`, versao **157.0**), que e' uma build real do Firefox* => *e o **`web-ext`** (10.8.0, instalado)
sabe arrancar um Firefox com a extensao.* => *logo:*

    web-ext run --source-dir <stage do pacote Firefox> --firefox <firefox.exe> --no-reload --start-url about:blank

=> *e a **prova** esta' na linha que o `web-ext` imprime:*

    Installed <caminho do stage>

=> *«Installed» e' o proprio `web-ext` a dizer que o **Firefox aceitou o pacote** -- *nao e' uma inferencia nossa.*

***E O QUE ELE **NAO** PROVA, e e' declarado.*** *O Playwright **nao controla** o Firefox que o `web-ext` lanca
(nao ha' `connectOverCDP` no Firefox, e o Playwright-Firefox nao carrega extensoes pela API dele)* => *logo o
Firefox fica com **`INSTALL=sim`** e **`E2E=parcial`**: a extensao **instala e arranca**, mas nao **opera** (DOM,
clique) por este caminho* => *para operar seria preciso um Firefox nativo + `--remote-debugging-port` (que o Firefox
nao expoe com o protocolo do Chrome).*
"""
import io
import os
import re
import subprocess
import sys
import time
from pathlib import Path

# E-1519: a raiz vem do FICHEIRO, nao de um caminho absoluto da maquina de quem o escreveu.
RAIZ = Path(__file__).resolve().parents[1]
WEB_EXT = RAIZ / "_tmp" / "appdata" / "npm" / "web-ext.cmd"
STAGE = RAIZ / "dist" / "browser-bridge-firefox" / "hades-browser-bridge-firefox-UNSIGNED"
LOG = RAIZ / "_tmp" / "web-ext-run.log"
ESPERA = 40


def firefox_do_playwright() -> Path | None:
    # ***`E-1122`/`E-1121`: O `LOCALAPPDATA` TEM DE SER O REAL.*** *A bateria redirecciona-o para
    # `_tmp\localappdata`, e o Firefox do Playwright vive em `%LOCALAPPDATA%\ms-playwright`* => *com ele
    # substituido, o `web-ext` **nao encontra o Firefox** e o teste falha por **ambiente**, nao por defeito* =>
    # ***`e' a 2.a vez na mesma sessao: um ambiente substituido nao e' um ambiente filtrado -- o programa nem
    # arranca.`***
    base = Path(os.environ.get("LOCALAPPDATA", "")) / "ms-playwright"
    if not base.is_dir():
        base = Path(os.environ.get("USERPROFILE", "")) / "AppData" / "Local" / "ms-playwright"
    if not base.is_dir():
        return None
    for d in sorted(base.glob("firefox-*"), reverse=True):
        exe = d / "firefox" / "firefox.exe"
        if exe.is_file():
            return exe
    return None


def matar() -> None:
    for nome in ("firefox",):
        try:
            subprocess.run(["taskkill", "/F", "/IM", nome + ".exe"], capture_output=True, timeout=30)
        except Exception:
            pass


falhas: list[str] = []


def T(nome: str, ok: bool, det: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    ff = firefox_do_playwright()
    T("(1) ha' um Firefox REAL para o `web-ext` arrancar", ff is not None,
      str(ff) if ff else "o Playwright Firefox nao esta' em %LOCALAPPDATA%\\ms-playwright")
    T("(2) o `web-ext` esta' instalado", WEB_EXT.is_file(), "10.8.0 instalado neste host")
    T("(3) o stage do pacote Firefox existe", (STAGE / "manifest.json").is_file())
    if not ff or not WEB_EXT.is_file() or not (STAGE / "manifest.json").is_file():
        print("  (sem os 3 pre-requisitos, o E2E nao pode correr)")
        return 1

    # *o `web-ext run` fica a correr; lancamos, esperamos, lemos o log e matamos*
    matar()
    if LOG.exists():
        LOG.unlink()
    env = dict(os.environ)
    env["TMP"] = str(RAIZ / "_tmp" / "webtemp")
    env["TEMP"] = env["TMP"]
    # ***O LOG E' ESCRITO PELO `Popen`, NAO PELO `>` DO `cmd`.*** *A 1.a versao punha o `>` dentro do comando
    # passado ao `cmd /c`, e o `cmd` **nao escreveu o log** (o `>` e as aspas aninhadas partiram-se) -- e o teste
    # falhou por **invocacao**, nao por defeito da extensao:* *a mesma invocacao, feita com `Start-Process` de
    # fora, funcionou (7 processos e `Installed`).* => *logo: os argumentos vao **separados**, e o `stdout` do
    # processo e' o ficheiro de log.*
    f = io.open(LOG, "w", encoding="utf-8", errors="replace")
    proc = subprocess.Popen(["cmd.exe", "/c", str(WEB_EXT), "run", "--source-dir", str(STAGE),
                             "--firefox", str(ff), "--no-reload", "--start-url", "about:blank", "--no-input"],
                            env=env, stdout=f, stderr=subprocess.STDOUT)
    try:
        # *esperar que o `web-ext` instale e arranque*
        instalou = False
        for _ in range(ESPERA):
            time.sleep(1)
            if LOG.is_file():
                txt = io.open(LOG, encoding="utf-8", errors="replace").read()
                if "Installed" in txt:
                    instalou = True
                    break
        txt = io.open(LOG, encoding="utf-8", errors="replace").read() if LOG.is_file() else ""
        T("(4) e o `web-ext` diz **`Installed`** -- o Firefox ACEITOU o pacote", instalou,
          "a linha e' do proprio web-ext, nao uma inferencia nossa")

        # *e o Firefox tem de estar a correr*
        r = subprocess.run(["tasklist", "/FI", "IMAGENAME eq firefox.exe", "/NH"],
                           capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=30)
        n = len(re.findall(r"firefox\.exe", r.stdout or "", re.I))
        T("(5) e o Firefox arrancou (processos `firefox.exe`)", n > 0, "%d processos" % n)
        T("(5b) e o pacote instalado e' o do **stage** (o do `.zip` verificado no `E-1116`)",
          "hades-browser-bridge-firefox-UNSIGNED" in txt, "o caminho citado e' o do pacote")
    finally:
        matar()
        try:
            proc.kill()
        except Exception:
            pass
        try:
            f.close()
        except Exception:
            pass

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO E2E DO FIREFOX ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  FIREFOX: a extensao **INSTALA e ARRANCA** num Firefox REAL (157.0 do Playwright), e o")
    print("  `web-ext` — a ferramenta do PROPRIO Mozilla — diz `Installed`.")
    print("")
    print("  ***O QUE ELE NAO PROVA***: *o **operar** (DOM, clique). O Playwright **nao controla** o Firefox")
    print("  que o `web-ext` lanca, e o Playwright-Firefox nao carrega extensoes pela API dele. => o Firefox")
    print("  fica **`INSTALL=sim`** e **`E2E=parcial`**: instala e arranca, mas nao opera por este caminho.*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
