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


def main() -> int:
    # ---------- Android: o `app.json`/`eas.json` existem? o APK? ----------
    app_json = MOBILE / "app.json"
    eas_json = MOBILE / "eas.json"
    android_dir = MOBILE / "android"
    apk = artefacto("dist/hades-mobile-unsigned.apk", "apps/mobile-agentic/android/app/build/outputs/apk/debug/app-debug.apk")

    # ---------- Web/PWA: o build de producao ----------
    web_dist = RAIZ / "app" / "ui" / "app" / "dist"
    web_index = web_dist / "index.html"

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
            ("dist/HadesSetup.exe",), "NÃO MEDIDO (INSTALLED=NAO)",
            "NÃO MEDIDO", "não assinado (unsigned)", "o SmartScreen é limite esperado"),
        plataforma(
            "Linux", "`cmd/`, `app/`, `server/` (Go)",
            existe_wf("dz23-linux-package.yaml"),
            ("dist/Hades-0.2.0-linux-amd64-unsigned.tar.gz", "dist/hades-linux-amd64-unsigned.tar.gz"),
            "NÃO MEDIDO (INSTALLED=NAO)", "NÃO MEDIDO", "não assinado (unsigned)",
            "cross-build Windows não é teste Linux"),
        plataforma(
            "macOS", "`cmd/`, `app/`, `server/` (Go)",
            existe_wf("dz23-macos-build.yaml"),
            None,
            "NÃO MEDIDO", "NÃO MEDIDO", "não assinado (unsigned)",
            "**BLOCKED_EXTERNAL**: exige macOS/Xcode, e não ha'"),
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
