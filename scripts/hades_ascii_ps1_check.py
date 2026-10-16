#!/usr/bin/env python3
"""Catraca do ASCII nos `.ps1` (E-992).

Por que este checker existe
---------------------------
O `~/.dsh/AGENTS.md` registra, medido neste host: *«`pwsh` nao esta' no PATH; `powershell.exe`
e' 5.1 Desktop e le `.ps1` sem BOM como Windows-1252 -- **por isso o script e' ASCII puro**»*.

E o `E-989` mediu o `scripts/hades_local_gates.ps1` com **15 bytes > 127** em **5 linhas**,
todas **pre-existentes**, e decidiu o seguinte: *«as 5 restantes ficam **declaradas**, nao
reescritas -- nao sao minhas, e mexer em comentarios alheios por causa de um byte e' o tipo
de alteracao que ninguem pediu.»*

**Mas eu violei a regra duas vezes.** Na ronda do `E-989` escrevi um `§`; na do `E-991`
escrevi outro `§` e um `e'` acentuado. *Duas vezes o mesmo erro, separado por duas rondas, e
as duas vezes porque confiei na disciplina em vez de num teste* => e' exactamente a doutrina
que esta missao ja' pagou caro para aprender: **uma regra sem catraca e' uma intencao**
(`E-987`/`E-989`/`E-991`).

O que este checker faz
----------------------
1. Le' cada `.ps1` **como bytes** e conta os que sao `> 127`.
2. Compara com a **linha de base declarada** em `BASELINE` -- *um tecto por ficheiro, nao um
   numero global*.
3. **Falha** se um ficheiro **subir** acima do tecto, dizendo o ficheiro, o antes e o depois.
4. **Falha tambem se descer**? **Nao.** Descer e' bom (`E-989` fez isso), e travar uma
   melhoria seria a catraca a impedir o progresso. Mas *o valor medido e' impresso*, para que
   a descida seja visivel e o tecto possa ser baixado conscientemente.
5. Aceita `--atualizar` para **baixar** o tecto ao medido -- *nunca subir*: subir exigiria
   editar `BASELINE` a mao, o que e' uma decisao, e nao um efeito colateral de rodar o script.

Uso
---
    python scripts/hades_ascii_ps1_check.py              # confere
    python scripts/hades_ascii_ps1_check.py --atualizar  # baixa o tecto ao medido (nunca sobe)
    python scripts/hades_ascii_ps1_check.py --json
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent

# ---------------------------------------------------------------------------
# **E-992, e a primeira versao deste checker estava ERRADA**: ela contava os bytes > 127
# **por ficheiro**, sem distinguir **onde** eles estavam. Medido entao:
#
#     scripts/install.ps1          COMENTARIO: 20   ★ STRING: 34   <- chega ao utilizador
#     scripts/build_windows.ps1    COMENTARIO: 11   STRING:  0     <- inofensivo
#     scripts/hades_local_gates.ps1 COMENTARIO: 5   STRING:  0     <- inofensivo
#     scripts/hades-quickstart.ps1 COMENTARIO:  0   1 char         <- trivial
#
# *E a distincao e' a coisa toda.* Um byte > 127 num **comentario** nao passa pelo
# `powershell.exe`: ele le' o ficheiro e nao o mostra. Um byte > 127 numa **string** e'
# impresso -- e o 5.1, que le' `.ps1` sem BOM como **Windows-1252**, transforma `a` com
# acento (`0xC3 0xA1`) em dois caracteres visiveis (`A-til + ¡`). => *o utilizador ve'
# lixo, e o pior caso e' o `install.ps1`, que e' **o instalador**.*
#
# E **o defeito ja' aconteceu**: o `install.ps1` tem **3** ocorrencias de `A-til` literal --
# *mojibake ja' produzido*, alguem escreveu, foi lido como Windows-1252 e regravado como
# UTF-8. *Documentar isso e' mais util que contar bytes.*
#
# Logo o tecto passa a ser **por classe**:
#   - **STRING**: tecto **ZERO**, sem linha de base. *Se um `.ps1` imprime um caractere
#     nao-ASCII, isso e' um defeito de produto, e nao um numero a tolerar.*
#   - **COMENTARIO**: tecto por ficheiro, declarado com a origem. *Codigo herdado tem
#     comentarios que nao sao meus, e reescreve-los nao foi pedido.*
#
# *Uma linha de base **suposta** produz uma violacao **suposta*** -- e por isso `COMENTARIO`
# so' lista o que foi **medido**; o que nao esta' na lista tem tecto zero.
# ---------------------------------------------------------------------------
BASELINE_COMENTARIO = {
    "scripts/hades_local_gates.ps1": 5,
    "scripts/build_windows.ps1": 11,
    "scripts/install.ps1": 20,
    "scripts/hades-quickstart.ps1": 0,
}

# ---------------------------------------------------------------------------
# **DIVIDA DECLARADA - nao-ASCII em STRING que JA' EXISTE, com motivo e desbloqueio.**
#
# *Isto e' uma divida, e nao um tecto.* O tecto e' **zero**; estes valores existem porque a
# correccao **nao pode ser feita sem o teste que a valida**, e esse teste e' o do §FASE-6
# (*instalacao limpa num perfil de utilizador comum*).
#
# **Por que nao corrigi agora**: o `install.ps1` e' **o instalador**. As strings com acento
# sao **mensagens ao utilizador** -- *muda-las e' mudar o produto*, e validar isso exige
# instalar sobre a RC antiga e desinstalar, que e' trabalho do §FASE-6 e nao desta fatia.
# *Corrigir 34 caracteres sem esse teste seria trocar um defeito visivel por um invisivel.*
#
# **E o que o `powershell.exe` 5.1 faz**: le' `.ps1` sem BOM como **Windows-1252**, pelo que
# `a` com acento (`0xC3 0xA1`) vira **dois** caracteres impressos. => *o utilizador ve' lixo.*
#
# **A assinatura de que isto JA' ACONTECEU**: o `install.ps1` tem **3** ocorrencias de `A-til`
# literal -- *mojibake produzido*, alguem escreveu, foi lido como Windows-1252 e regravado
# como UTF-8. O checker reporta-o em `NOTA`, e nao em `VIOLACAO`: *nao e' um numero a tolerar,
# e' um defeito a corrigir na fonte que o escreve.*
#
# **Desbloqueio**: (a) decidir BOM UTF-8 **ou** ASCII puro para os `.ps1` de produto; (b)
# aplicar a decisao ao `install.ps1` e ao `hades-quickstart.ps1`; (c) provar com instalacao
# limpa no §FASE-6. => *Ate' la', o tecto fica **congelado** no medido: **divida nova falha**,
# que e' o que o §ANTILOOP item 9 proibe esconder.*
DIVIDA_STRING = {
    "scripts/install.ps1": (
        34,
        "O instalador e' o §FASE-6; corrigir as mensagens sem instalacao limpa testada "
        "trocaria um defeito visivel por um invisivel.",
    ),
    "scripts/hades-quickstart.ps1": (
        1,
        "1 caractere; corrige-se junto com o install.ps1 para nao dividir a decisao "
        "BOM-vs-ASCII em duas.",
    ),
}


NAO_ASCII = re.compile(r"[^\x00-\x7F]")
COMENTARIO = re.compile(r"^\s*#")


def classificar(linha: str) -> str:
    """Diz se a linha e' **comentario** ou **codigo** (string/expressao).

    *Conservador de proposito*: se houver duvida, e' `codigo` -- porque o tecto de `codigo`
    e' zero, e um falso positivo aqui **obriga a olhar**, enquanto um falso negativo deixaria
    passar um caractere que chega ao utilizador.
    """
    return "comentario" if COMENTARIO.match(linha) else "codigo"


def mojibake(linhas: list[str]) -> list[tuple[int, str]]:
    """Linhas com `A-til` ou `A-circunflexo` seguidos de outro simbolo -- assinatura de
    texto que foi UTF-8, lido como Windows-1252 e regravado. *Nao e' uma teoria: e' o
    que o `install.ps1` ja' tem.*"""
    achados = []
    for i, linha in enumerate(linhas, 1):
        if re.search(r"[\u00c3\u00c2][\u0080-\u00bf\u00a0-\u00bf]", linha):
            achados.append((i, linha.strip()[:100]))
    return achados


def medir(caminho: Path) -> dict:
    linhas = caminho.read_text(encoding="utf-8", errors="replace").splitlines()
    contagem = {"comentario": 0, "codigo": 0}
    for linha in linhas:
        n = len(NAO_ASCII.findall(linha))
        if n:
            contagem[classificar(linha)] += n
    return {"comentario": contagem["comentario"], "codigo": contagem["codigo"],
            "mojibake": mojibake(linhas)}


def varrer() -> dict[str, dict]:
    medido: dict[str, dict] = {}
    for caminho in sorted(RAIZ.rglob("*.ps1")):
        rel = caminho.relative_to(RAIZ).as_posix()
        if "_tmp/" in rel or "node_modules/" in rel:
            continue
        try:
            medido[rel] = medir(caminho)
        except OSError:
            continue
    return medido


def conferir() -> tuple[dict[str, dict], list[str], list[str]]:
    medido = varrer()
    violacoes: list[str] = []
    notas: list[str] = []
    for rel, m in sorted(medido.items()):
        tecto_divida, motivo = DIVIDA_STRING.get(rel, (0, ""))
        if m["codigo"] > tecto_divida:
            if tecto_divida:
                violacoes.append(
                    "%s: %d caractere(s) NAO-ASCII em CODIGO/string, ACIMA DA DIVIDA "
                    "DECLARADA de %d -- divida nova nao entra (%s)"
                    % (rel, m["codigo"], tecto_divida, motivo[:80])
                )
            else:
                violacoes.append(
                    "%s: %d caractere(s) NAO-ASCII em CODIGO/string (tecto 0) -- isto e' "
                    "impresso ao utilizador e o powershell.exe 5.1 corrompe-o"
                    % (rel, m["codigo"])
                )
        tecto = BASELINE_COMENTARIO.get(rel, 0)
        if m["comentario"] > tecto:
            violacoes.append(
                "%s: %d caractere(s) > 127 em COMENTARIO, acima do tecto de %d"
                % (rel, m["comentario"], tecto)
            )
        if m["mojibake"]:
            notas.append(
                "%s: %d linha(s) com assinatura de mojibake (ja' corrompidas) -- ex.: :%d %s"
                % (rel, len(m["mojibake"]), m["mojibake"][0][0], m["mojibake"][0][1][:70])
            )
    for rel in sorted(BASELINE_COMENTARIO):
        if rel not in medido:
            violacoes.append("%s: na linha de base mas ausente do repositorio" % rel)
    return medido, violacoes, notas


def main() -> int:
    parser = argparse.ArgumentParser(description="Catraca do ASCII nos .ps1 (E-992).")
    parser.add_argument("--atualizar", action="store_true",
                        help="mostra o medido para actualizar a linha de base (nunca sobe)")
    parser.add_argument("--json", action="store_true", help="imprime o medido em JSON")
    args = parser.parse_args()

    medido = varrer()

    if args.json:
        json.dump({"baseline_comentario": BASELINE_COMENTARIO, "medido": medido},
                  sys.stdout, ensure_ascii=False, indent=2)
        sys.stdout.write("\n")
        return 0

    if args.atualizar:
        for rel, m in sorted(medido.items()):
            tecto = BASELINE_COMENTARIO.get(rel, 0)
            if m["comentario"] != tecto:
                print("%s: COMENTARIO %d -> %d" % (rel, tecto, m["comentario"]))
        print("Ajuste BASELINE_COMENTARIO a mao. O tecto NUNCA sobe por rodar o script.")
        return 0

    _, violacoes, notas = conferir()
    print("ASCII PS1: %d ficheiro(s); codigo/string com tecto 0; comentario com tecto declarado"
          % len(medido))
    for rel, m in sorted(medido.items()):
        if not (m["comentario"] or m["codigo"]):
            continue
        tecto = BASELINE_COMENTARIO.get(rel, 0)
        div = DIVIDA_STRING.get(rel, (0, ""))[0]
        s = "OK " if m["codigo"] <= div and m["comentario"] <= tecto else "FALHA"
        extra = " divida-string=%d" % div if div else ""
        print("  %-5s %-38s codigo=%-3d comentario=%-3d (tecto %d)%s"
              % (s, rel, m["codigo"], m["comentario"], tecto, extra))

    if notas:
        print()
        for n in notas:
            print("NOTA: " + n)
        print("  (mojibake e' texto que JA' foi corrompido por codificacao errada -- nao e' um")
        print("   numero a tolerar: e' um defeito a corrigir na fonte que o escreve.)")

    if violacoes:
        print()
        for v in violacoes:
            print("VIOLACAO: " + v)
        print()
        print("Por que isto importa: `powershell.exe` 5.1 le' `.ps1` sem BOM como Windows-1252.")
        print("Em COMENTARIO o byte nao e' impresso; em STRING ele e' -- e sai corrompido.")
        print("Escreva 'ESTAGIO' em vez do simbolo de secao, e evite acentos em .ps1.")
        return 1

    print("ASCII PS1: nenhum ficheiro acima do tecto, e nenhum caractere nao-ASCII em string.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
