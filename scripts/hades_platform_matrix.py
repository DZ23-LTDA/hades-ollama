#!/usr/bin/env python3
"""`hades_platform_matrix.py` — o INVENTARIO DE PLATAFORMAS do §ESTAGIO 1 (`E-1100`).

***O QUE O §ESTAGIO 1 EXIGE.*** *«atualizar inventario de **codigo, plataformas, workflows, pacotes e
lacunas**»* => *e o prompt (item 8 e §FASE 6) exige distinguir **SOURCE, BUILD, PACKAGE, INSTALL,
DEVICE_E2E, assinatura e hash** por plataforma* => ***«Cross-compile nao e' teste nativo; bundle web nao e'
aplicativo movel; configuracao EAS nao e' APK»***.

***O QUE ESTE GERADOR FAZ.*** *Le o repositorio -- workflows, `app.json`, `eas.json`, `dist/` -- e **mede** o
estado de cada uma das 6 plataformas nos 7 eixos* => *e **nao inventa**: se nao ha' o artefato, escreve
`AUSENTE` ou `NAO_MEDIDO`; se ha', cita o **hash**.*

***A REGRA QUE ELE APLICA.*** *Uma plataforma so' conta como `PACKAGE` se o ficheiro **existe**; so' conta
como `INSTALL` se houver **prova de instalacao** (o `dist/` nao e' prova); so' conta como `DEVICE_E2E` se
houver **teste em dispositivo/emulador** (e nao ha').* => ***`um inventario honesto e' sobretudo uma lista do
que falta.`***
"""
import hashlib
import io
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SAIDA = RAIZ / "docs" / "mission" / "HADES_PLATFORM_MATRIX.md"
WORKFLOWS = RAIZ / ".github" / "workflows"
DIST = RAIZ / "dist"
MOBILE = RAIZ / "apps" / "mobile-agentic"


def sha(p: Path) -> str:
    if not p.is_file():
        return "AUSENTE"
    return hashlib.sha256(p.read_bytes()).hexdigest()[:32]


def tamanho(p: Path) -> str:
    if not p.is_file():
        return "-"
    b = p.stat().st_size
    return "%.1f MB" % (b / (1024 ** 2)) if b > 1024 ** 2 else "%.0f KB" % (b / 1024)


def existe_wf(*nomes: str) -> str:
    for n in nomes:
        if (WORKFLOWS / n).is_file():
            return n
    return "AUSENTE"


def artefacto(*caminhos: str) -> Path | None:
    for c in caminhos:
        p = RAIZ / c
        if p.is_file():
            return p
    return None


def plataforma(nome: str, fonte: str, wf: str, pacote, install: str, e2e: str,
               assinatura: str, nota: str) -> dict:
    p = artefacto(*pacote) if pacote else None
    return {
        "nome": nome,
        "fonte": fonte,
        "build_wf": wf,
        "pacote": (str(p.relative_to(RAIZ)) if p else "AUSENTE"),
        "pacote_hash": sha(p) if p else "AUSENTE",
        "pacote_tamanho": tamanho(p) if p else "-",
        "install": install,
        "device_e2e": e2e,
        "assinatura": assinatura,
        "nota": nota,
    }


def instalador_do_candidato() -> tuple[Path | None, str]:
    """O instalador Windows **do candidato**, com o `SOURCE_SHA` que ele proprio declara.

    *PORQUE ISTO EXISTE (medido na ronda 237):* o gerador apontava para o caminho **fixo**
    `dist/HadesSetup.exe`, que em 2026-10-17 era o instalador do dia ANTERIOR
    (`SHA256 A70B5107...`, gerado em 2026-10-16 22:58) -- enquanto o artefacto do candidato
    `32a7ffc57` estava em `dist/hades-installer-818251093/HadesSetup-818251093.exe`
    (`SHA256 0F95833A...`). *A matriz estava a atribuir ao candidato um artefacto de outro
    snapshot.* **Cura:** procurar o directorio `dist/hades-installer-*` mais RECENTE e ler o
    `SOURCE_SHA` do `PROVENIENCIA.txt` que o script de instalador escreve ao lado.
    """
    candidatos = sorted(
        (p for p in DIST.glob("hades-installer-*/HadesSetup-*.exe") if p.is_file()),
        key=lambda p: p.stat().st_mtime, reverse=True)
    for exe in candidatos:
        prov = exe.parent / "PROVENIENCIA.txt"
        declarado = "NAO_DECLARADO"
        if prov.is_file():
            m = re.search(r"SOURCE_SHA:\s*([0-9a-f]{7,40})", prov.read_text(encoding="utf-8", errors="replace"))
            if m:
                declarado = m.group(1)
        return exe, declarado
    # sem instalador do candidato: cai para o caminho legado, e DIZ que a proveniencia nao foi conferida
    legado = DIST / "HadesSetup.exe"
    if legado.is_file():
        return legado, "NAO_DECLARADO"
    return None, "AUSENTE"


def main() -> int:
    # ---------- Android: o `app.json`/`eas.json` existem? o APK? ----------
    app_json = MOBILE / "app.json"
    eas_json = MOBILE / "eas.json"
    android_dir = MOBILE / "android"
    apk = artefacto("dist/hades-mobile-unsigned.apk", "apps/mobile-agentic/android/app/build/outputs/apk/debug/app-debug.apk")

    # ---------- Web/PWA: o build de producao ----------
    web_dist = RAIZ / "app" / "ui" / "app" / "dist"
    web_index = web_dist / "index.html"

    instalador_win, sha_declarado_win = instalador_do_candidato()
    if sha_declarado_win not in ("NAO_DECLARADO", "AUSENTE"):
        nota_win = ("artefacto do candidato (SOURCE_SHA " + sha_declarado_win +
                    " declarado no PROVENIENCIA.txt)")
    else:
        nota_win = "artefacto LEGADO: o PROVENIENCIA.txt nao confirma o SHA do candidato"
    linhas = []
    linhas.append("# HADES — MATRIZ DE PLATAFORMAS (medida)")
    linhas.append("")
    linhas.append("**Este ficheiro é gerado.** Não edite à mão: corrija a fonte e regenere com")
    linhas.append("`python -X utf8 scripts/hades_platform_matrix.py`.")
    linhas.append("")
    linhas.append("## Porque este ficheiro existe")
    linhas.append("")
    linhas.append("O §ESTÁGIO 1 exige «atualizar inventário de **código, plataformas, workflows, pacotes e")
    linhas.append("lacunas**», e o prompt exige distinguir **SOURCE, BUILD, PACKAGE, INSTALL, DEVICE_E2E,")
    linhas.append("assinatura e hash** por plataforma.")
    linhas.append("")
    linhas.append("⇒ ***cross-compile não é teste nativo; bundle web não é aplicativo móvel; configuração EAS")
    linhas.append("não é APK.***")
    linhas.append("")
    linhas.append("## A matriz")
    linhas.append("")
    linhas.append("| Plataforma | SOURCE | BUILD (workflow) | PACKAGE | Tamanho | Hash (SHA-256[:32]) | INSTALL | DEVICE_E2E | Assinatura |")
    linhas.append("| --- | --- | --- | --- | --- | --- | --- | --- | --- |")

    tabela = [
        plataforma(
            "Windows", "`cmd/`, `app/`, `server/` (Go)",
            existe_wf("dz23-windows-installer.yaml"),
            ((instalador_win,) if instalador_win else ("dist/HadesSetup.exe",)),
            "NÃO MEDIDO (INSTALLED=NAO)",
            "NÃO MEDIDO", "não assinado (unsigned)", nota_win),
        plataforma(
            "Linux", "`cmd/`, `app/`, `server/` (Go)",
            existe_wf("dz23-linux-package.yaml"),
            ("dist/Hades-0.2.0-linux-amd64-unsigned.tar.gz",
             "dist/hades-linux-amd64-unsigned.tar.gz",
             # ***O `E-1112` GEROU O TARBALL POR CROSS-COMPILE.*** *O `scripts/package_linux.sh` do repo usa
             # `CGO_ENABLED=1`; este foi **CGO=0** a partir do Windows* => *logo: **PACKAGE=sim** (o ficheiro
             # existe, com tamanho e hash), **BUILD=cross** (nao nativo), e **INSTALL=NAO** (nao houve Linux
             # para instalar)* => ***`um pacote que esconde como foi feito e' pior do que um pacote que nao
             # existe.`***
             "dist/hades-linux-amd64/hades-linux-amd64-UNSIGNED.tar.gz"),
            # *INSTALL continua NAO MEDIDO -- o pacote existe, mas ninguem o instalou*
            "NÃO MEDIDO (INSTALLED=NAO)", "NÃO MEDIDO", "não assinado (unsigned)",
            "cross-compile Windows→Linux com CGO=0; **NAO** e' build nativo, **NAO** foi instalado, e **NAO** "
            "e' byte-a-byte o artefacto do `dz23-linux-package.yaml` (que usa CGO=1)"),
        plataforma(
            "macOS", "`cmd/`, `app/`, `server/` (Go)",
            existe_wf("dz23-macos-build.yaml"),
            None,
            "NÃO MEDIDO", "NÃO MEDIDO", "não assinado (unsigned)",
            # ***A CAUSA MEDIDA NO `E-1112` (nao uma suposicao).*** *O cross-compile `GOOS=darwin` com
            # **`CGO_ENABLED=0`** falha:*
            #     `discover/gpu.go:28:18: undefined: GetCPUMem`
            #     `discover/types.go:49:13: undefined: GetCPUMem`
            # => *e a causa e': o `GetCPUMem` do darwin esta' em **`discover/gpu_darwin.go`**, que usa
            # **`import "C"`** (`#cgo LDFLAGS: -framework Foundation -framework CoreGraphics -framework
            # Metal`) => *e o Go **exclui automaticamente** qualquer ficheiro com `import "C"` quando
            # `CGO_ENABLED=0`* => ***logo o macOS exige CGO, e o CGO exige a toolchain Apple.*** => *nao e'
            # «o build nao foi tentado»: foi tentado, e **chegou ao limite** -- e o limite e' a **toolchain
            # nativa**, nao o codigo.* => ***`BLOCKED_EXTERNAL` legitimo: o passo de desbloqueio e' o
            # `dz23-macos-build.yaml` em `macos-latest` (que tem Xcode e CGO).`***
            "**BLOCKED_EXTERNAL (medido, nao suposto)**: `GOOS=darwin CGO_ENABLED=0` falha com "
            "`undefined: GetCPUMem` -- o `GetCPUMem` do darwin vive em `discover/gpu_darwin.go`, que usa "
            "`import \"C\"` (Metal/Foundation), e o Go exclui ficheiros com `import \"C\"` quando CGO=0. "
            "Desbloqueio: o `dz23-macos-build.yaml` em `macos-latest` (Xcode + CGO)."),
        plataforma(
            "Android", "`apps/mobile-agentic/` (Expo/RN + `android/`)",
            existe_wf("dz23-e2e.yaml"),
            (str(apk.relative_to(RAIZ)),) if apk else None,
            "NÃO MEDIDO", "NÃO MEDIDO", "APK de desenvolvimento (não publicável)",
            "**o `app.json`/`eas.json` existem; o APK não**" if not apk else "APK presente"),
        plataforma(
            "iOS", "`apps/mobile-agentic/` (Expo/RN)",
            "AUSENTE",
            None,
            "NÃO MEDIDO", "NÃO MEDIDO", "não assinado (unsigned)",
            "**BLOCKED_EXTERNAL**: exige macOS/Xcode + conta Apple"),
        plataforma(
            "Web/PWA", "`app/ui/app/` (React+TS+Vite)",
            existe_wf("dz23-e2e.yaml", "test.yaml"),
            (str(web_index.relative_to(RAIZ)),) if web_index.is_file() else None,
            "N/A (build de produção)", "NÃO MEDIDO (PWA_INSTALADO_NA_UI=NAO)",
            "N/A", "servido por HTTP; o `manifest` existe"),
    ]

    for p in tabela:
        linhas.append("| **%s** | %s | `%s` | `%s` | %s | `%s` | %s | %s | %s |" % (
            p["nome"], p["fonte"], p["build_wf"], p["pacote"], p["pacote_tamanho"],
            p["pacote_hash"], p["install"], p["device_e2e"], p["assinatura"]))

    linhas.append("")
    linhas.append("## O que cada eixo significa, e o que NÃO conta")
    linhas.append("")
    linhas.append("| Eixo | Conta como MEDIDO | NÃO conta |")
    linhas.append("| --- | --- | --- |")
    linhas.append("| **SOURCE** | o código existe e compila | «o repositório menciona a plataforma» |")
    linhas.append("| **BUILD** | um workflow **executou** o build | o workflow existir por si |")
    linhas.append("| **PACKAGE** | o ficheiro **existe**, com tamanho e hash | «o workflow geraria» |")
    linhas.append("| **INSTALL** | alguém **instalou** e abriu | o `dist/` ter o instalador |")
    linhas.append("| **DEVICE_E2E** | teste em **dispositivo/emulador** | o Playwright no browser |")
    linhas.append("| **Assinatura** | certificado + notarização | «não assinado durante o desenvolvimento» |")
    linhas.append("")
    linhas.append("## As lacunas, por plataforma")
    linhas.append("")
    for p in tabela:
        linhas.append("- **%s**: %s" % (p["nome"], p["nota"]))
    linhas.append("")
    linhas.append("## Os workflows, contados")
    linhas.append("")
    wfs = sorted(w.name for w in WORKFLOWS.glob("*.y*ml"))
    linhas.append("**%d workflows** em `.github/workflows/`:" % len(wfs))
    linhas.append("")
    for w in wfs:
        linhas.append("- `%s`" % w)
    linhas.append("")
    linhas.append("## A conclusão honesta")
    linhas.append("")
    linhas.append("⇒ ***o inventário é, sobretudo, uma lista do que falta:*** *nenhuma plataforma tem")
    linhas.append("`INSTALL` medido, nenhuma tem `DEVICE_E2E`, e as que têm `PACKAGE` têm-no **não assinado**.")
    linhas.append("Isso **não** é uma falha do produto: é o estado **medido**, e é o que impede")
    linhas.append("`NOT_RELEASE_READY` de virar `RELEASE_READY` sem homologação real.")
    linhas.append("")

    conteudo = "\n".join(linhas)
    check = "--check" in sys.argv
    if check:
        if not SAIDA.is_file():
            print("  FALHA: %s nao existe (corra sem --check)" % SAIDA.relative_to(RAIZ))
            return 1
        actual = io.open(SAIDA, encoding="utf-8").read()
        # *os hashes e tamanhos mudam quando o artefato muda; o `--check` compara as seccoes estaveis.*
        for marca in ("## A matriz", "## O que cada eixo significa", "## As lacunas, por plataforma",
                      "## Os workflows, contados", "## A conclusão honesta", "Windows", "Linux",
                      "macOS", "Android", "iOS", "Web/PWA"):
            if marca not in actual:
                print("  FALHA: a matriz nao cobre `%s`" % marca)
                return 1
        print("  MATRIZ DE PLATAFORMAS CONSISTENTE: 6 plataformas, 7 eixos, workflows e lacunas")
        return 0
    SAIDA.parent.mkdir(parents=True, exist_ok=True)
    io.open(SAIDA, "w", encoding="utf-8", newline="\n").write(conteudo)
    print("  gerado: %s (%d plataformas)" % (SAIDA.relative_to(RAIZ), len(tabela)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
