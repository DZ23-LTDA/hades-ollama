"""`hades_native_host_roundtrip.py` — prova a **instalacao REVERSIVEL** do Native Host (`E-1119`).

***O QUE O PROMPT EXIGE.*** *«instalar/configurar o Native Host/Companion de modo **reversível**»* e *«permitir
remover extensão, host, chaves e permissões **sem apagar projetos do usuário**»*.

***E O QUE JA' EXISTE.*** *O `scripts/hades_native_host_install.ps1` (6 446 B) instala em
`%LOCALAPPDATA%\\HADES\\NativeHost\\` e registra **duas** chaves em `HKCU` (Chrome e Edge), e escreve um
`INSTALACAO.json` que lista **tudo** o que criou -- *«Nada de "apagar a pasta e rezar"»*.

***O QUE ESTE CHECKER FAZ, e por que a SANDBOX importa.*** *A bateria nao pode escrever no registo do utilizador a
cada execucao* => *logo o roundtrip corre numa **sandbox**: o `LOCALAPPDATA` e' redireccionado para uma pasta
temporaria, e o instalador e' chamado com `-Navegador ''` (nenhum navegador registado).* => *assim a prova e'
**repetivel e sem efeito no host** -- e o registo real **nao** e' tocado.*

***AS 7 VERIFICACOES.***
  1. *o estado actual do host e' **NAO INSTALADO** (limpo) -- *se estiver instalado, o cheque falha e diz por que'*;
  2. *o instalador tem `-Desinstalar` e `-Diagnostico`;
  3. *na sandbox, o `-Diagnostico` diz `NAO INSTALADO`;
  4. *instala na sandbox: o manifesto, o lancador e o `INSTALACAO.json` aparecem;
  5. *o `INSTALACAO.json` lista `chavesRegisto`, `manifesto`, `lancador`, `idDaExtensao`, `assinado`, `publicado`;
  6. *o **manifesto REAL** tem `path` **absoluto**, `type: stdio` e o **ID da extensao** em `allowed_origins`;
  7. **desinstala**: tudo o que o `INSTALACAO.json` listava **desapareceu**, e a pasta tambem.
"""
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(r"C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama")
INSTALADOR = RAIZ / "scripts" / "hades_native_host_install.ps1"
POWERSHELL = ["powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File"]

falhas: list[str] = []


def T(nome: str, ok: bool, det: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
    if not ok:
        falhas.append(nome)


def correr(env_extra: dict, *args: str) -> tuple[int, str]:
    """Roda o instalador com um ambiente **filtrado** (o `dict(os.environ)` + o override)."""
    env = dict(os.environ)
    env.update(env_extra)
    r = subprocess.run(POWERSHELL + [str(INSTALADOR)] + list(args), capture_output=True,
                       text=True, encoding="utf-8", errors="replace", env=env, timeout=180)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def ler_json(p: Path) -> dict:
    # *o PowerShell 5.1 escreve com BOM -- e o `json.loads` cru rebenta com ele*
    return json.loads(p.read_text(encoding="utf-8-sig"))


def main() -> int:
    # ---------- (1) o estado REAL tem de estar limpo ----------
    real = Path(os.environ.get("LOCALAPPDATA", "")) / "HADES" / "NativeHost"
    T("(1) o host NAO esta' instalado no host real (estado limpo)", not real.exists(),
      str(real) + " -- se existir, rode `-Desinstalar` antes")

    # ---------- (2) o instalador tem os dois modos ----------
    txt = INSTALADOR.read_text(encoding="utf-8", errors="replace") if INSTALADOR.is_file() else ""
    T("(2) o instalador existe", bool(txt), "%.0f B" % len(txt))
    T("(2b) e tem `-Desinstalar`", "[switch]$Desinstalar" in txt)
    T("(2c) e tem `-Diagnostico`", "[switch]$Diagnostico" in txt)
    T("(2d) e escreve o `INSTALACAO.json` (o manifesto de reversibilidade)",
      "INSTALACAO.json" in txt)
    T("(2e) e so' mexe em `HKCU` (nunca `HKLM`, nunca elevacao)",
      "HKCU:" in txt and "HKLM" not in txt)

    # ---------- a SANDBOX ----------
    sandbox = Path(tempfile.mkdtemp(prefix="hades-nh-"))
    env_sb = {"LOCALAPPDATA": str(sandbox)}
    destino = sandbox / "HADES" / "NativeHost"
    print("  sandbox: %s" % sandbox)

    try:
        # ---------- (3) diagnostico na sandbox ----------
        rc, saida = correr(env_sb, "-Diagnostico")
        T("(3) o `-Diagnostico` na sandbox diz NAO INSTALADO", rc == 0 and "NAO INSTALADO" in saida,
          "exit=%d" % rc)

        # ---------- (4) instalar na sandbox, SEM registo ----------
        rc, saida = correr(env_sb, "-Navegador", "")
        T("(4) a instalacao na sandbox completa", rc == 0, "exit=%d" % rc)
        mf = destino / "com.dz23ltd.hades.bridge.json"
        lc = destino / "com.dz23ltd.hades.bridge.cmd"
        inst = destino / "INSTALACAO.json"
        T("(4b) e cria o manifesto", mf.is_file())
        T("(4c) e o **lancador** (o `.cmd`, nao o `.mjs`)", lc.is_file(),
          ".mjs directo nao e' executavel no Windows (medido no E-1032)")
        T("(4d) e o `INSTALACAO.json`", inst.is_file())

        # ---------- (5) o INSTALACAO.json lista o que criou ----------
        if inst.is_file():
            d = ler_json(inst)
            for campo in ("chavesRegisto", "manifesto", "lancador", "idDaExtensao", "assinado", "publicado"):
                T("(5) o `INSTALACAO.json` tem `%s`" % campo, campo in d,
                  str(d.get(campo))[:60] if campo in d else "AUSENTE")
            T("(5b) e declara `assinado: false` e `publicado: false`",
              d.get("assinado") is False and d.get("publicado") is False,
              "assinado=%s publicado=%s" % (d.get("assinado"), d.get("publicado")))

        # ---------- (6) o manifesto REAL ----------
        if mf.is_file():
            m = ler_json(mf)
            T("(6) o manifesto tem `path` **absoluto**",
              isinstance(m.get("path"), str) and os.path.isabs(m["path"]), str(m.get("path"))[:70])
            T("(6b) e `type: stdio`", m.get("type") == "stdio")
            orig = " ".join(m.get("allowed_origins") or [])
            T("(6c) e o `allowed_origins` tem o **ID da extensao**",
              "chrome-extension://" in orig and "REPLACE" not in orig, orig[:60])

        # ---------- (7) DESINSTALAR ----------
        rc, saida = correr(env_sb, "-Desinstalar")
        T("(7) a desinstalacao na sandbox completa", rc == 0, "exit=%d" % rc)
        T("(7b) e o manifesto **desapareceu**", not mf.exists())
        T("(7c) e o lancador tambem", not lc.exists())
        T("(7d) e o `INSTALACAO.json` tambem", not inst.exists())
        T("(7e) e a pasta `NativeHost` foi removida", not destino.exists(),
          "a pasta vazia tambem sai")
        # *e o que NAO pode ser tocado: os projetos do utilizador*
        T("(7f) e os ficheiros do PROJETO continuam intactos",
          (RAIZ / "apps" / "hades-browser-bridge" / "native-host" / "companion.mjs").is_file(),
          "o `-Desinstalar` nao toca no repo")
    finally:
        shutil.rmtree(sandbox, ignore_errors=True)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO ROUNDTRIP DO NATIVE HOST ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  NATIVE HOST: instalacao **REVERSIVEL** provada -- instala, lista tudo o que criou, e o")
    print("  `-Desinstalar` remove **exactamente** isso (inclusive a pasta vazia), sem tocar no projeto.")
    print("")
    print("  ***O QUE ELE NAO FAZ***: *nao publica, nao assina, nao mexe em `HKLM`, nao pede elevacao, e nao")
    print("  registou nenhum navegador REAL nesta prova (a sandbox corre com `-Navegador ''`).*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
