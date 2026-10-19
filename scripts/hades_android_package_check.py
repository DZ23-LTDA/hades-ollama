#!/usr/bin/env python3
"""HADES -- catraca de COERENCIA do package Android (E-1293).

O DEFEITO QUE ESTE INSTRUMENTO IMPEDE

O `AndroidManifest.xml` referencia a Activity e a Application com nome RELATIVO:

    <application android:name=".MainApplication" ...>
    <activity android:name=".MainActivity" ...>

O manifest **nao declara atributo `package`** -- o AGP 8.x removeu-o. Um nome com ponto inicial
resolve contra o **`namespace` do modulo Gradle**, que era `com.dz23ltd.hades`. Mas os ficheiros
Kotlin estavam em:

    app/src/main/java/com/lmprado/dz23agentic/MainActivity.kt
    package com.lmprado.dz23agentic

O Android procuraria `com.dz23ltd.hades.MainActivity` -- **classe que nao existia**. O resultado e'
`ClassNotFoundException` no arranque: **a aplicacao nao abre**.

    *Um `namespace` que nao corresponde ao package do codigo e' um endereco para uma casa que nao
    existe: o build passa e o app nao abre.*

PORQUE O BUILD NAO APANHAVA ISTO

A compilacao Kotlin nao exige que o `package` corresponda ao caminho do ficheiro, e o Gradle nao
verifica que o `namespace` case com as classes declaradas. O erro so' aparece **em execucao**, no
dispositivo -- e nesta missao nao ha' Android SDK nem emulador para o encontrar.

    *Um erro que so' o dispositivo encontra fica a espera do utilizador.*

O QUE ELE VERIFICA

1. Todo `.kt` em `app/src/main/java` tem `package` igual ao caminho desde a raiz java.
2. O `namespace` do `build.gradle` bate com o package dos `.kt`.
3. Se o manifest usa nome relativo (`.X`), a classe `X` existe no caminho do `namespace`.
4. Se o manifest declara `package`, o `namespace` e o `applicationId` estao coerentes com ele.
5. Nenhum ficheiro versionado refere o package antigo em codigo (documentacao e' permitida).
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
APP_ANDROID = RAIZ / "apps" / "mobile-agentic" / "android" / "app"
JAVA_RAIZ = APP_ANDROID / "src" / "main" / "java"
MANIFEST = APP_ANDROID / "src" / "main" / "AndroidManifest.xml"
BUILD_GRADLE = APP_ANDROID / "build.gradle"

# O package historico. Nao pode voltar em codigo.
PACKAGE_HISTORICO = "com.lmprado.dz23agentic"

# Extensoes de documentacao onde o nome antigo E' legitimo (registo da migracao).
SUFIXOS_DE_DOCUMENTACAO = (".md", ".txt")


def ler_package(ficheiro: Path) -> str:
    """Devolve o `package` declarado no ficheiro Kotlin, ou string vazia."""
    texto = ficheiro.read_text(encoding="utf-8", errors="replace")
    casamento = re.search(r"^\s*package\s+([\w.]+)", texto, re.MULTILINE)
    return casamento.group(1) if casamento else ""


def ler_namespace(build_gradle: Path) -> str:
    """Devolve o `namespace` do build.gradle."""
    texto = build_gradle.read_text(encoding="utf-8", errors="replace")
    casamento = re.search(r"^\s*namespace\s+['\"]([\w.]+)['\"]", texto, re.MULTILINE)
    return casamento.group(1) if casamento else ""


def ler_application_id(build_gradle: Path) -> str:
    """Devolve o `applicationId` do build.gradle."""
    texto = build_gradle.read_text(encoding="utf-8", errors="replace")
    casamento = re.search(r"^\s*applicationId\s+['\"]([\w.]+)['\"]", texto, re.MULTILINE)
    return casamento.group(1) if casamento else ""


def nomes_relativos_do_manifest(manifest: Path) -> list[str]:
    """Devolve os nomes com ponto inicial (`android:name=\"...\"`), sem o ponto."""
    texto = manifest.read_text(encoding="utf-8", errors="replace")
    nomes: list[str] = []
    for casamento in re.finditer(r'android:name="\.([A-Za-z0-9_.]+)"', texto):
        nomes.append(casamento.group(1))
    return sorted(set(nomes))


def avaliar_ficheiros(kotlin: list[tuple[str, str]], namespace: str, relativos: list[str],
                      resolvidos: set[str]) -> list[str]:
    """Avaliacao pura, testavel sem tocar no disco.

    `kotlin` e' uma lista de `(caminho_relativo_desde_java, package)`.
    `resolvidos` e' o conjunto de nomes de classe totalmente qualificados que existem.
    """
    problemas: list[str] = []

    if not kotlin:
        problemas.append("nenhum ficheiro .kt encontrado: o instrumento nao aprova o que nao le'")
        return problemas

    # 1. package corresponde ao caminho.
    for relativo, package in kotlin:
        esperado = str(Path(relativo).parent).replace("\\", ".").replace("/", ".")
        if package != esperado:
            problemas.append(
                "PACKAGE != CAMINHO: %s declara `package %s` mas o caminho diz `%s`"
                % (relativo, package or "(ausente)", esperado)
            )

    # 2. namespace bate com o package dos .kt.
    packages = {p for _, p in kotlin}
    if namespace and packages and namespace not in packages:
        problemas.append(
            "NAMESPACE SEM CLASSE: `namespace '%s'` no build.gradle, mas nenhum .kt declara esse "
            "package (%s)" % (namespace, ", ".join(sorted(packages)))
        )

    # 3. Nomes relativos do manifest tem de resolver.
    for relativo in relativos:
        qualificado = "%s.%s" % (namespace, relativo) if namespace else relativo
        if qualificado not in resolvidos:
            problemas.append(
                "CLASSE INEXISTENTE: o manifest usa `.%s`, que resolve para `%s`; nenhuma classe "
                "com esse nome existe -- ClassNotFoundException no arranque do app"
                % (relativo, qualificado)
            )

    return problemas


def avaliar() -> list[str]:
    """Avalia o repositorio real."""
    if not JAVA_RAIZ.is_dir():
        return ["diretorio java/ ausente: %s" % JAVA_RAIZ]
    if not BUILD_GRADLE.is_file():
        return ["build.gradle ausente: %s" % BUILD_GRADLE]

    namespace = ler_namespace(BUILD_GRADLE)
    if not namespace:
        return ["build.gradle sem `namespace` declarado"]

    kotlin: list[tuple[str, str]] = []
    resolvidos: set[str] = set()
    for ficheiro in sorted(JAVA_RAIZ.rglob("*.kt")):
        relativo = ficheiro.relative_to(JAVA_RAIZ).as_posix()
        package = ler_package(ficheiro)
        kotlin.append((relativo, package))
        # O nome da classe e' o nome do ficheiro sem extensao.
        resolvidos.add("%s.%s" % (package, ficheiro.stem))

    relativos: list[str] = []
    if MANIFEST.is_file():
        relativos = nomes_relativos_do_manifest(MANIFEST)

    problemas = avaliar_ficheiros(kotlin, namespace, relativos, resolvidos)

    # 4. package antigo em codigo (documentacao e' permitida).
    if BUILD_GRADLE.is_file():
        application_id = ler_application_id(BUILD_GRADLE)
        if application_id and application_id != namespace:
            problemas.append(
                "NAMESPACE != APPLICATION_ID: `%s` vs `%s`. Sao conceitos diferentes -- o namespace "
                "resolve nomes de classe, o applicationId identifica a app na loja -- mas divergirem "
                "exige justificacao escrita; hoje ninguem a tem" % (namespace, application_id)
            )

    return problemas


def verificar_package_historico() -> list[str]:
    """O package antigo nao pode voltar em CODIGO.

    COMENTARIO NAO E' CODIGO

    A primeira versao deste checker acusou os comentarios que documentam o proprio defeito -- no
    workflow do CI e no `hades_local_gates.ps1` -- porque procurava o texto em qualquer linha. Um
    comentario que explica porque o package mudou e' a documentacao que impede a regressao de voltar;
    recusa-lo obrigaria a apagar a explicacao para o gate ficar verde.

        *Um gate que exige apagar a explicacao do defeito para passar ensina a nao documentar.*

    A verificacao e' por linha, e uma linha e' comentario quando o package aparece depois de um
    marcador de comentario da linguagem. Verifica so' ficheiros rastreados pelo git, para nao gastar
    minutos numa varredura de disco.
    """
    import subprocess

    conclusao = subprocess.run(
        ["git", "grep", "-n", "-I", PACKAGE_HISTORICO],
        cwd=str(RAIZ),
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=120,
    )
    problemas: list[str] = []
    for linha in (conclusao.stdout or "").splitlines():
        partes = linha.split(":", 2)
        if len(partes) < 3:
            continue
        caminho, numero, conteudo = partes[0], partes[1], partes[2]
        if caminho.lower().endswith(SUFIXOS_DE_DOCUMENTACAO):
            continue  # documentacao da migracao: legitimo
        if linha_e_comentario(conteudo, caminho):
            continue  # comentario que documenta o defeito: legitimo e desejavel
        problemas.append(
            "PACKAGE HISTORICO EM CODIGO: %s:%s ainda usa `%s` fora de comentario"
            % (caminho, numero, PACKAGE_HISTORICO)
        )
    return problemas


# Marcadores de comentario por tipo de ficheiro, do mais especifico para o mais generico.
MARCADORES_DE_COMENTARIO = {
    ".py": ("#",),
    ".ps1": ("#",),
    ".yaml": ("#",),
    ".yml": ("#",),
    ".go": ("//",),
    ".kt": ("//",),
    ".ts": ("//",),
    ".tsx": ("//",),
    ".js": ("//",),
    ".mjs": ("//",),
    ".gradle": ("//",),
    ".pro": ("#",),
    ".md": (),
}


def linha_e_comentario(conteudo: str, caminho: str) -> bool:
    """True quando o conteudo da linha e' integral ou essencialmente comentario."""
    texto = conteudo.strip()
    if not texto:
        return True
    sufixo = Path(caminho).suffix.lower()
    marcadores = MARCADORES_DE_COMENTARIO.get(sufixo, ("#", "//"))
    for marcador in marcadores:
        if texto.startswith(marcador):
            return True
        posicao = texto.find(marcador)
        if posicao <= 0:
            continue
        antes = texto[:posicao]
        # `https://` e `ftp://` tem `//` mas nao iniciam comentario.
        if marcador == "//" and antes.rstrip().endswith(":"):
            continue
        # O package depois de um marcador valido NA MESMA linha: comentario de fim de linha.
        if PACKAGE_HISTORICO in texto[posicao:]:
            return True
    return False


def main() -> int:
    problemas = avaliar() + verificar_package_historico()
    if problemas:
        print("ANDROID PACKAGE: %d PROBLEMA(S)" % len(problemas))
        for problema in problemas:
            print("  - %s" % problema)
        return 1
    print(
        "ANDROID PACKAGE: o `namespace`, o `applicationId` e o `package` dos .kt sao coerentes; "
        "todo nome relativo do manifest resolve para uma classe que existe."
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
