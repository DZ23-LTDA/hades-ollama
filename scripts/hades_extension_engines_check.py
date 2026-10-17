"""`hades_extension_engines_check.py` — a extensao corre nos **motores REAIS** (`E-1121`).

***O QUE O `E-1105` TINHA ESCRITO, e que estava **meio errado**.*** *«Chrome 137+ removeu `--load-extension`, e o
Edge partilha o engine»* => *a conclusao implicita era: **nem o Chrome nem o Edge** carregam a extensao, e o E2E so'
e' possivel no Chromium do Playwright.*

***MEDIDO AGORA, motor a motor:***

    Chromium do Playwright  156.0.8078.4   carrega  =>  10 e 25 verificacoes passam
    Edge REAL               155.0.4283.45  carrega  =>  10 e 25 verificacoes passam
    Chrome REAL             155.0.8059.79  NAO      =>  o service_worker nao acorda

=> ***`partilhar o motor nao e' partilhar a politica`*** -- *o Edge **manteve** o `--load-extension` que o Chrome
removeu.* => *e isto muda a classificacao do §ESTAGIO 14: o **Edge** deixa de ser `BLOCKED` e passa a **medido**.*

***O QUE ESTE CHECKER EXIGE.***
  - *os **2** E2E (DOM e takeover) passam no **Chromium** -- o caminho que sempre funcionou;*
  - *e os **2** passam no **Edge REAL** -- o motor que o `E-1105` tinha dado por perdido;*
  - *o **Chrome** e' **reportado**, mas **nao exigido**: se a Google voltar a permitir o flag, o checker nao falha
    por isso -- *um criterio que exige a doenca nao deixa curar.*
"""
import os
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(r"C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama")
APP = RAIZ / "app" / "ui" / "app"
DOM = "e2e-extension-dom.mjs"
TAKEOVER = "e2e-extension-takeover.mjs"

falhas: list[str] = []


def T(nome: str, ok: bool, det: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
    if not ok:
        falhas.append(nome)


# ***`E-1121`/`E-1108`: UM FLAKE REGISTADO NAO E' UM FLAKE ESCONDIDO.*** *Na 1.a execucao completa da bateria
# o Chromium do Playwright falhou (o `(0)` nem apareceu -- o browser nao arrancou), e nas execucoes seguintes
# passou 5 vezes seguidas. E' um **flake de arranque do browser**, nao um defeito da extensao* => *logo o checker
# tenta **2 vezes** (tecto de 2, nao retry sem fim) e **REGISTA** quando precisou da 2.a* => ***`o que separa um
# flake tolerado de um flake escondido e' o registo.`***
TENTATIVAS = 2
flakes: list[str] = []

# ***`E-1121`/`E-1108`: O `LOCALAPPDATA` TEM DE SER O REAL.*** *A bateria redirecciona o `LOCALAPPDATA` para
# `_tmp\localappdata` (para nao sujar o perfil do utilizador) -- mas o **Chromium do Playwright vive em
# `%LOCALAPPDATA%\ms-playwright`*** => *com o `LOCALAPPDATA` substituido, o **Chromium nao e' encontrado** e o
# browser **nem arranca** (o `(0)` nao aparece).* => *e o **Edge** funciona porque esta' em `C:\Program Files` e
# nao depende do `LOCALAPPDATA`* => ***`e a mesma licao do E-1108: um ambiente substituido nao e' um ambiente
# filtrado -- o programa nem arranca.`*** => *logo: restaura-se o `LOCALAPPDATA` REAL, e redirecciona-se so' o
# `TMP`/`TEMP` (que e' o que precisa de ser escritorio temporario).*
_LOCAL_REAL = os.path.join(os.environ.get("USERPROFILE", ""), "AppData", "Local")


def correr(script: str, canal: str) -> tuple[int, str, str]:
    env = dict(os.environ)
    env["HADES_E2E_CHANNEL"] = canal
    env["TMP"] = str(RAIZ / "_tmp" / "webtemp")
    env["TEMP"] = env["TMP"]
    # *o `LOCALAPPDATA` real: o Playwright guarda os browsers la'. Sem isto, o Chromium nao existe.*
    if os.path.isdir(os.path.join(_LOCAL_REAL, "ms-playwright")):
        env["LOCALAPPDATA"] = _LOCAL_REAL
    rc, saida, versao = 1, "", "?"
    for tent in range(1, TENTATIVAS + 1):
        r = subprocess.run(["node", script], cwd=str(APP), capture_output=True, text=True,
                           encoding="utf-8", errors="replace", env=env, timeout=300)
        saida = (r.stdout or "") + (r.stderr or "")
        m = re.search(r"o motor: ([^\n]*?)\s+-- versao=(\S+)", saida)
        versao = m.group(2) if m else "?"
        rc = r.returncode
        if rc == 0:
            if tent > 1:
                flakes.append("%s (%s) precisou da tentativa %d" % (script, canal or "chromium", tent))
            break
    return rc, saida, versao


def main() -> int:
    T("(0) os 2 E2E existem", (APP / DOM).is_file() and (APP / TAKEOVER).is_file())

    # ---------- (1) os 2 E2E no Chromium do Playwright ----------
    for script, nome in ((DOM, "DOM"), (TAKEOVER, "takeover")):
        rc, saida, versao = correr(script, "")
        n = len(re.findall(r"^  OK ", saida, re.M))
        T("(1) o %s passa no **Chromium do Playwright** (%s)" % (nome, versao), rc == 0,
          "%d verificacoes" % n if rc == 0 else saida.strip().splitlines()[-3:][:1][0][:70] if saida.strip() else "?")

    # ---------- (2) os 2 E2E no EDGE REAL ----------
    for script, nome in ((DOM, "DOM"), (TAKEOVER, "takeover")):
        rc, saida, versao = correr(script, "msedge")
        n = len(re.findall(r"^  OK ", saida, re.M))
        T("(2) o %s passa no **Edge REAL** (%s)" % (nome, versao), rc == 0,
          "%d verificacoes -- o Edge NAO partilha o destino do Chrome" % n if rc == 0 else "falhou")

    # ---------- (3) o Chrome REAL: reportado, NAO exigido ----------
    # ***UM CRITERIO QUE EXIGE A DOENCA NAO DEIXA CURAR.*** *Se a Google voltar a permitir o `--load-extension`,
    # o Chrome passaria -- e o checker nao pode falhar por isso.* => *por isso o Chrome e' **reportado**, e o
    # criterio e' «pelo menos 2 motores reais» (Chromium + Edge), nao «o Chrome falha».
    rcC, saidaC, versaoC = correr(DOM, "chrome")
    print("  NOTA  (3) o **Chrome REAL** (%s): %s" % (versaoC,
          "carrega" if rcC == 0 else "NAO carrega o `--load-extension` (removido no Chrome 137+)"))
    T("(3) e o relatorio diz **porque** (nao e' um `skip`: e' um limite medido com versao)",
      isinstance(versaoC, str) and versaoC != "?", "versao=%s" % versaoC)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NOS MOTORES ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    if flakes:
        print("  FLAKES REGISTADOS (tolerados, com tecto de 2 tentativas):")
        for f in flakes:
            print("    - " + f)
        print("  *Um flake de ARRANQUE do browser nao e' um defeito da extensao -- e fica registado em vez de")
        print("  escondido: `o que separa um flake tolerado de um flake escondido e' o registo.`*")
        print("")
    print("  MOTORES: os 2 E2E (DOM e takeover) passam no **Chromium** e no **Edge REAL**.")
    print("  O Chrome real continua sem carregar o `--load-extension` -- e' reportado, nao exigido.")
    print("")
    print("  ***`PARTILHAR O MOTOR NAO E' PARTILHAR A POLITICA`***: *o Edge 155 manteve o flag que o Chrome 155")
    print("  removeu. O `E-1105` tinha dado o Edge por perdido; o `E-1121` mediu-o e ele funciona.*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
