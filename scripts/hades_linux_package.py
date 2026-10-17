"""Empacota o binario Linux cross-compilado num tarball verificavel -- `E-1112`.

***O QUE ISTO FAZ.*** *Pega no `_tmp/hades-linux-amd64` (ELF confirmado, **45,6 MB**, sha256 `c2d0f71f...`) e
produz `dist/hades-linux-amd64/` com:*

    - `bin/ollama` -- o binario (ELF, Linux amd64);
    - `VERSION` -- a metadata (versao, commit, arch, signed=false, **e o `BUILD_MODE`**);
    - `install.sh` / `uninstall.sh` -- o instalador local sem root (o mesmo contrato do `package_linux.sh`);
    - `PROVENIENCIA.txt` -- **o que este pacote e' e o que NAO e'**;
    - `SHA256SUMS.txt` -- o hash.

***E O QUE ELE DECLARA, porque e' o que importa.*** *O pacote e' **cross-compile** (`GOOS=linux` a partir do
Windows) com ***`CGO_ENABLED=0`*** -- *e o `scripts/package_linux.sh` do repo usa **`CGO_ENABLED=1`** => *logo
este tarball **nao** e' byte-a-byte o que o workflow produziria: e' um **artefacto real, nao-assinado, de
cross-build**, e o `PROVENIENCIA.txt` diz isso na primeira linha.* => ***`um pacote que esconde como foi feito e'
pior do que um pacote que nao existe.`***
"""
import hashlib
import io
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

RAIZ = Path(r"C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama")
BIN = RAIZ / "_tmp" / "hades-linux-amd64"
DEST = RAIZ / "dist" / "hades-linux-amd64"
STAGE = DEST / "hades-linux-amd64-UNSIGNED"


def sha256(p: Path) -> str:
    h = hashlib.sha256()
    with io.open(p, "rb") as f:
        for bloco in iter(lambda: f.read(1 << 20), b""):
            h.update(bloco)
    return h.hexdigest()


def verificar() -> int:
    """`--check`: o pacote EXISTE e e' INTEGRO (sem empacotar de novo)."""
    falhas = []
    def T(nome, ok, det=""):
        print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
        if not ok:
            falhas.append(nome)

    T("(1) o directorio do pacote existe", DEST.is_dir(), str(DEST.relative_to(RAIZ)))
    tar = DEST / "hades-linux-amd64-UNSIGNED.tar.gz"
    T("(1b) o tarball existe", tar.is_file(),
      "%.2f MB" % (tar.stat().st_size / (1 << 20)) if tar.is_file() else "AUSENTE")
    stage = STAGE
    for rel in ("bin/ollama", "VERSION", "install.sh", "uninstall.sh", "PROVENIENCIA.txt"):
        T("(2) o stage tem `%s`" % rel, (stage / rel).is_file())
    # ***O BINARIO TEM DE SER ELF/LINUX, NAO PE/WINDOWS.*** *e' o criterio que impede chamar «pacote Linux» a
    # um binario Windows renomeado.*
    b = stage / "bin" / "ollama"
    if b.is_file():
        with io.open(b, "rb") as f:
            magic = f.read(4)
        T("(2b) e o binario e' **ELF** (Linux), nao PE (Windows)", magic == b"\x7fELF",
          "magic=%s" % magic.hex().upper())
        T("(2c) e o `VERSION` declara `signed=false` e o `build_mode`",
          "signed=false" in io.open(stage / "VERSION", encoding="utf-8").read()
          and "build_mode" in io.open(stage / "VERSION", encoding="utf-8").read(),
          "a proveniencia fica DENTRO do pacote")
    # *o `SHA256SUMS.txt` tem de bater com o ficheiro*
    somas = DEST / "SHA256SUMS.txt"
    T("(3) o `SHA256SUMS.txt` existe", somas.is_file())
    if somas.is_file() and tar.is_file():
        linha = io.open(somas, encoding="utf-8").read().split("\n")[0]
        esperado = linha.split()[0] if linha.split() else ""
        real = sha256(tar)
        T("(3b) e o SHA-256 do tarball **confere**", esperado == real,
          "%s..." % real[:32] if esperado == real else "esperado=%s real=%s" % (esperado[:16], real[:16]))
    # *a matriz de plataformas cita o pacote?*
    matriz = RAIZ / "docs" / "mission" / "HADES_PLATFORM_MATRIX.md"
    if matriz.is_file():
        m = io.open(matriz, encoding="utf-8").read()
        T("(4) a matriz de plataformas cita o pacote", "hades-linux-amd64" in m,
          "o PACKAGE da matriz nao e' uma promessa")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO PACOTE LINUX ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  PACOTE LINUX: tarball com binario ELF, VERSION, install/uninstall, PROVENIENCIA e")
    print("  SHA256SUMS que confere -- e a matriz de plataformas cita-o.")
    print("")
    print("  ***O QUE ELE NAO E'.*** *NAO foi instalado nem executado (nao houve Linux); NAO e'")
    print("  byte-a-byte o artefacto do `dz23-linux-package.yaml` (que usa CGO=1); e NAO e' assinado.*")
    return 0


def main() -> int:
    if "--check" in sys.argv:
        return verificar()
    if not BIN.is_file():
        print("  FALHA: o binario nao existe em %s" % BIN)
        return 1

    # *confirmar que e' ELF antes de empacotar*
    with io.open(BIN, "rb") as f:
        magic = f.read(4)
    if magic != b"\x7fELF":
        print("  FALHA: o binario nao e' ELF (magic=%r) -- nao e' um binario Linux" % magic)
        return 1
    print("  binario ELF confirmado: %.1f MB" % (BIN.stat().st_size / (1 << 20)))

    # *o commit, sem inventar*
    try:
        commit = subprocess.run(["git", "rev-parse", "--short=12", "HEAD"], cwd=str(RAIZ),
                                capture_output=True, text=True, encoding="utf-8").stdout.strip()
    except OSError:
        commit = "unknown"
    if not commit:
        commit = "unknown"
    sha_bin = sha256(BIN)
    print("  commit=%s  sha256[:32]=%s" % (commit, sha_bin[:32]))

    # *o stage*
    (STAGE / "bin").mkdir(parents=True, exist_ok=True)
    destino = STAGE / "bin" / "ollama"
    io.open(destino, "wb").write(io.open(BIN, "rb").read())
    # *no Windows nao ha chmod; o `install.sh` usa `install -Dm755`, que forca o modo*
    print("  bin/ollama escrito")

    VERSION = (
        "product=HADES (Ollama Full)\n"
        "version=0.0.0-dev+%s\n"
        "commit=%s\n"
        "platform=linux\n"
        "arch=amd64\n"
        "signed=false\n"
        "build_mode=cross-compile (GOOS=linux a partir do Windows)\n"
        "cgo_enabled=0\n"
        "NOTA: o scripts/package_linux.sh do repo usa CGO_ENABLED=1; este pacote e' CGO=0.\n"
        "NOTA: por isso este tarball NAO e' byte-a-byte o que o workflow dz23-linux-package.yaml produz.\n"
    ) % (commit, commit)
    io.open(STAGE / "VERSION", "w", encoding="utf-8", newline="\n").write(VERSION)

    INSTALL = """#!/usr/bin/env bash
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX="${PREFIX:-$HOME/.local}"
install -Dm755 "${HERE}/bin/ollama" "${PREFIX}/bin/ollama"
install -Dm644 "${HERE}/VERSION" "${PREFIX}/share/hades/VERSION"
echo "HADES instalado em ${PREFIX}/bin/ollama"
echo "Garanta que ${PREFIX}/bin esta no PATH. Rode: ollama --version"
"""
    UNINSTALL = """#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-$HOME/.local}"
rm -f "${PREFIX}/bin/ollama"
rm -f "${PREFIX}/share/hades/VERSION"
echo "HADES removido de ${PREFIX}. Os modelos em ~/.ollama NAO foram tocados."
"""
    io.open(STAGE / "install.sh", "w", encoding="utf-8", newline="\n").write(INSTALL)
    io.open(STAGE / "uninstall.sh", "w", encoding="utf-8", newline="\n").write(UNINSTALL)

    PROV = (
        "PROVENIENCIA DO PACOTE -- LEIA ANTES DE USAR\n"
        "===========================================\n"
        "\n"
        "O QUE ESTE PACOTE E':\n"
        "  - um binario ELF Linux amd64, NAO ASSINADO;\n"
        "  - gerado por CROSS-COMPILE a partir do Windows (GOOS=linux GOARCH=amd64);\n"
        "  - com CGO_ENABLED=0.\n"
        "\n"
        "O QUE ESTE PACOTE NAO E':\n"
        "  - NAO foi construido em Linux nativo, e NAO foi INSTALADO nem EXECUTADO;\n"
        "  - NAO e' byte-a-byte o artefacto do workflow dz23-linux-package.yaml (que usa CGO_ENABLED=1);\n"
        "  - NAO tem smoke test (install/--version/start/stop) -- porque nao houve Linux para o correr;\n"
        "  - NAO e' publicavel: sem assinatura, sem notarizacao, sem revisao humana.\n"
        "\n"
        "SOURCE_SHA: %s\n"
        "BINARIO_SHA256: %s\n"
        "EMPACOTADO_EM_UTC: %s\n"
        "\n"
        "PARA INSTALAR (num Linux de confianca):\n"
        "  tar -xzf hades-linux-amd64-UNSIGNED.tar.gz\n"
        "  cd hades-linux-amd64-UNSIGNED && ./install.sh\n"
        "\n"
        "Para desinstalar: ./uninstall.sh   (NAO apaga os modelos em ~/.ollama)\n"
    ) % (commit, sha_bin, datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"))
    io.open(STAGE / "PROVENIENCIA.txt", "w", encoding="utf-8", newline="\n").write(PROV)

    # *o tarball -- determinista: ordem fixa e mtime fixo*
    tar = DEST / "hades-linux-amd64-UNSIGNED.tar.gz"
    if tar.exists():
        tar.unlink()
    # *usar o tar do Windows (bsdtar), com --mtime fixo para reprodutibilidade*
    r = subprocess.run(
        ["tar", "-czf", str(tar), "--mtime=2026-01-01 00:00:00", "-C", str(DEST), "hades-linux-amd64-UNSIGNED"],
        capture_output=True, text=True, encoding="utf-8", errors="replace")
    if r.returncode != 0:
        print("  FALHA no tar: %s" % (r.stderr or "")[:200])
        return 1
    print("  tarball: %s (%.2f MB)" % (tar.name, tar.stat().st_size / (1 << 20)))

    # *o SHA256SUMS*
    soma = sha256(tar)
    io.open(DEST / "SHA256SUMS.txt", "w", encoding="utf-8", newline="\n").write(
        "%s  %s\n%s  %s\n" % (soma, tar.name, sha_bin, "hades-linux-amd64-UNSIGNED/bin/ollama"))

    print("  SHA256SUMS.txt: tar=%s..." % soma[:32])
    print("")
    print("  PACOTE GERADO: dist/hades-linux-amd64/")
    print("    - PACKAGE=sim (o ficheiro existe, com tamanho e hash)")
    print("    - INSTALL=nao (nao houve Linux para instalar)")
    print("    - DEVICE_E2E=nao")
    print("    - signed=false")
    return 0


if __name__ == "__main__":
    sys.exit(main())
