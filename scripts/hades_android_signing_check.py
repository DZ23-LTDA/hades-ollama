#!/usr/bin/env python3
"""HADES -- catraca da assinatura de release Android (E-1290).

O DEFEITO QUE ESTE INSTRUMENTO IMPEDE

`apps/mobile-agentic/android/app/build.gradle` vinha do template do React Native com:

    release {
        // Caution! In production, you need to generate your own keystore file.
        signingConfig signingConfigs.debug
    }

A debug keystore usa a senha publica `android` e o alias `androiddebugkey`, conhecidos desde 2010. Um
APK/AAB assinado com ela e' FORJAVEL por qualquer pessoa, e o Google Play RECUSA AAB assinado com
debug key.

    *Um artefacto que passa nos testes e nasce com a chave errada nao falha onde se ve': falha onde se
    distribui.*

O QUE ELE VERIFICA

1. O bloco `release` NAO contem `signingConfigs.debug`.
2. O bloco `release` usa `hadesCredencialDeReleaseCompleta`.
3. As quatro variaveis de ambiente estao declaradas.
4. Existe um guard em `assembleRelease`/`bundleRelease` que falha sem credencial.
5. O bloco `debug` PODE usar a debug key -- isso e' correcto e nao e' sinalizado.

PORQUE ELE EXISTE SEPARADAMENTE DE UM BUILD

Nao ha' JDK 17, Android SDK nem Gradle nesta maquina, por isso o build real e' `BLOCKED_EXTERNAL`. Sem
este instrumento, a correcao nao teria nenhuma prova a correr em CI -- e voltaria em silencio no
proximo prebuild.

    *Um teste que so' existe quando a ferramenta existe nao protege nada no dia em que ela falta.*
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
BUILD_GRADLE = RAIZ / "apps" / "mobile-agentic" / "android" / "app" / "build.gradle"

# As quatro variaveis que o guard exige. A lista vive aqui e no gradle; um teste garante que as duas
# nao divergem em silencio.
VARIAVEIS_DE_CREDENCIAL = (
    "HADES_ANDROID_STORE_FILE",
    "HADES_ANDROID_STORE_PASSWORD",
    "HADES_ANDROID_KEY_ALIAS",
    "HADES_ANDROID_KEY_PASSWORD",
)

TAREFAS_DE_RELEASE = ("assembleRelease", "bundleRelease")


def extrair_bloco(texto: str, nome: str) -> str:
    """Extrai o corpo de `nome { ... }` contando chavetas.

    Nao usa regex guloso: uma regex `release \\{.*?\\}` casaria com o `}` errado assim que houvesse um
    bloco aninhado, e um instrumento que le' o bloco errado da' veredicto sobre o bloco errado.
    """
    casamento = re.search(r"\b%s\s*\{" % re.escape(nome), texto)
    if not casamento:
        return ""
    inicio = texto.index("{", casamento.start())
    profundidade = 0
    for posicao in range(inicio, len(texto)):
        caracter = texto[posicao]
        if caracter == "{":
            profundidade += 1
        elif caracter == "}":
            profundidade -= 1
            if profundidade == 0:
                return texto[inicio + 1 : posicao]
    return ""


def extrair_build_type(texto: str, nome: str) -> str:
    """Extrai um build type de DENTRO de `buildTypes`.

    PORQUE ISTO E' SEPARADO DE `extrair_bloco`

    O `build.gradle` tem DOIS blocos chamados `release`:

        signingConfigs {
            if (...) { release { storeFile ... } }   <-- o primeiro
        }
        buildTypes {
            release { signingConfig ... }            <-- o que importa
        }

    Procurar `release {` no texto inteiro encontra o de `signingConfigs` e da' veredicto sobre o bloco
    errado -- foi exactamente o bug que o teste `test_repositorio_real_passa` apanhou na primeira
    versao deste instrumento.

    *Um instrumento que le' o bloco errado nao e' rigoroso: e' seguro sobre outra coisa.*
    """
    bloco_build_types = extrair_bloco(texto, "buildTypes")
    if not bloco_build_types:
        return ""
    return extrair_bloco(bloco_build_types, nome)


def avaliar(texto: str) -> list[str]:
    """Devolve os problemas encontrados. Lista vazia = conforme."""
    problemas: list[str] = []

    if not texto.strip():
        return ["o build.gradle esta' vazio ou ausente"]

    bloco_release = extrair_build_type(texto, "release")
    if not bloco_release:
        problemas.append(
            "ATENCAO: nao foi possivel encontrar `buildTypes.release`; o instrumento nao consegue "
            "verificar o que nao le'"
        )
        return problemas

    # 1. O defeito original, literal.
    if re.search(r"signingConfig\s+signingConfigs\.debug", bloco_release):
        problemas.append(
            "P0: o bloco `release` assina com `signingConfigs.debug` -- chave PUBLICA, artefacto "
            "forjavel, AAB recusado pelo Google Play"
        )

    # 2. Tem de usar a credencial condicional.
    if "hadesCredencialDeReleaseCompleta" not in bloco_release:
        problemas.append(
            "o bloco `release` nao usa `hadesCredencialDeReleaseCompleta`; sem isso nao ha' como "
            "distinguir release com credencial de release sem credencial"
        )

    # 3. As quatro variaveis.
    for variavel in VARIAVEIS_DE_CREDENCIAL:
        if variavel not in texto:
            problemas.append("variavel de credencial ausente do build.gradle: %s" % variavel)

    # 4. O guard nas duas tarefas de release.
    for tarefa in TAREFAS_DE_RELEASE:
        if tarefa not in texto:
            problemas.append(
                "falta o guard para `%s`: sem ele o release sem credencial produziria um artefacto "
                "nao assinado em silencio" % tarefa
            )

    if "GradleException" not in texto:
        problemas.append(
            "o guard nao lanca `GradleException`: um aviso nao aborta o build, e um erro que nao "
            "aborta nao e' um erro que nao aconteceu"
        )

    # 5. O bloco `debug` PODE usar a debug key -- nao e' problema. Se o bloco debug desapareceu,
    #    e' um aviso: alguem pode ter removido a configuracao em vez de separar as duas.
    bloco_debug = extrair_build_type(texto, "debug")
    if not bloco_debug:
        problemas.append(
            "o bloco `buildTypes.debug` desapareceu: a debug key e' legitima para o build de debug e "
            "a correccao nao era remover a configuracao, era separar release de debug"
        )

    return problemas


def main() -> int:
    if not BUILD_GRADLE.is_file():
        print("ANDROID SIGNING: build.gradle AUSENTE em %s" % BUILD_GRADLE)
        return 1

    texto = BUILD_GRADLE.read_text(encoding="utf-8")
    problemas = avaliar(texto)

    if problemas:
        print("ANDROID SIGNING: %d PROBLEMA(S)" % len(problemas))
        for problema in problemas:
            print("  - %s" % problema)
        return 1

    print(
        "ANDROID SIGNING: o release NAO usa a debug key, a credencial vem do ambiente (%d variaveis) "
        "e as duas tarefas de release tem guard que falha fechado." % len(VARIAVEIS_DE_CREDENCIAL)
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
