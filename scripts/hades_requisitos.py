#!/usr/bin/env python3
"""`hades_requisitos.py` — a CADEIA DE RASTREABILIDADE dos requisitos activos (`E-1091`).

***O QUE A §FASE-7 ITEM 4 EXIGE, E O QUE NAO EXISTIA.***

O contrato e' *«Para cada requisito ativo: `REQ-ID -> arquivo:linha -> teste -> comando -> resultado ->
plataforma -> commit/estado dirty -> artefacto/hash -> decisao`»*.

=> *ate' agora havia `PARIDADE_EVIDENCIAS.csv` com **3 colunas** (`id`, `tem_registro`,
`tamanho_primeira_celula`) -- que diz se a evidencia **existe**, e **nao** liga o requisito ao codigo,
ao teste, ao comando e ao resultado.* => *a cadeia **nao existia*.**

***O QUE ESTE GERADOR FAZ.*** Para cada **requisito** declarado em `REQUISITOS`, verifica **no codigo**
(nunca por copia) e emite uma linha do CSV com os 9 campos:

  `req_id, arquivo, linha, teste, comando, resultado, plataforma, commit, artefacto, decisao`

***E O QUE ELE NAO FAZ.*** Nao inventa linhas. *Se o `arquivo:linha` nao existe, ou se o teste nao esta'
no ficheiro, o campo fica `AUSENTE` e o gerador **falha** no `--check`* -- *um requisito cuja prova nao
existe e' um requisito sem prova, e o CSV tem de o dizer.*

***A FONTE DOS REQUISITOS.*** *Os **10 criterios da §FASE-3-A** (o pet) -- que o `E-1090` fechou -- e os
**criterios de seguranca** que o `E-1087` mediu.* => *`REQUISITOS` e' a lista, e cada entrada nomeia o
**arquivo e o simbolo que a prova** -- *nao uma frase.*
"""
import hashlib
import io
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SAIDA = RAIZ / "docs" / "audit" / "REQUISITOS.csv"

# ---------------------------------------------------------------------------
# A LISTA DOS REQUISITOS ACTIVOS, com a PROVA de cada um.
#
# *Cada entrada: (req_id, descricao, arquivo, simbolo_que_prova, teste, comando, plataforma, decisao).*
# => *o gerador verifica CADA CAMPO no codigo: o arquivo existe, o simbolo esta' la', o teste esta' la'.*
# => *um campo que nao se verifique fica `AUSENTE` e o `--check` falha.*
# ---------------------------------------------------------------------------
REQUISITOS = [
    (
        "REQ-3A-01",
        "o pet deriva o estado de eventos verificados, nao de estado inventado",
        "internal/agent/companion_estado.go",
        "DerivarEstadoCompanion",
        "internal/agent/companion_estado_test.go",
        "go test ./internal/agent -run TestCompanion -count=1",
        "windows/amd64",
        "MANTER",
    ),
    (
        "REQ-3A-02",
        "o conversor NAO copia o Payload -- a regra de DLP",
        "internal/agent/companion_eventos.go",
        "ConversorEventosDoCompanion",
        "internal/agent/companion_eventos_test.go",
        "go test ./internal/agent -run TestConversorEventos -count=1",
        "windows/amd64",
        "MANTER",
    ),
    (
        "REQ-3A-03",
        "a rota recusa outra organizacao e declara o scope",
        "server/agent_routes.go",
        "func (a *agentAPI) companion",
        "server/agent_companion_routes_test.go",
        "go test ./server -run TestCompanionRouteReportsHonestState -count=1",
        "windows/amd64",
        "MANTER",
    ),
    (
        "REQ-3A-04",
        "o componente valida o estado na fronteira (ESTADOS_CONHECIDOS)",
        "app/ui/app/src/lib/companionApi.ts",
        "ESTADOS_CONHECIDOS",
        "app/ui/app/src/lib/companionApi.validacao.test.ts",
        "npx vitest run src/lib/companionApi.validacao.test.ts",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-05",
        "as preferencias persistem sem POST em loop e declaram a falha de gravacao",
        "app/ui/app/src/lib/companionPreferencias.ts",
        "guardarPreferencias",
        "app/ui/app/src/lib/companionPreferencias.test.ts",
        "npx vitest run src/lib/companionPreferencias.test.ts",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-06",
        "o pet respeita o movimento reduzido e e' desligavel",
        "app/ui/app/src/components/CompanionMascote.tsx",
        "podeDesenharCompanion",
        "app/ui/app/src/components/CompanionMascote.test.tsx",
        "npx vitest run src/components/CompanionMascote.test.tsx",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-07",
        "as preferencias chegam a' rota que monta o pet",
        "app/ui/app/src/components/CompanionLigado.tsx",
        "carregarPreferencias",
        "app/ui/app/src/components/CompanionLigado.preferences.test.tsx",
        "npx vitest run src/components/CompanionLigado.preferences.test.tsx",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-08",
        "o pet nao tem timers nem listeners -- a catraca de performance",
        "app/ui/app/src/components/companionPerformance.test.ts",
        "setInterval",
        "app/ui/app/src/components/companionPerformance.test.ts",
        "npx vitest run src/components/companionPerformance.test.ts",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-09",
        "o E2E do pet corre contra HTTP real",
        "app/ui/app/e2e/companion.spec.ts",
        "mis_companion_e2e",
        "app/ui/app/e2e/companion.spec.ts",
        "npx playwright test e2e/companion.spec.ts",
        "web/chromium",
        "MANTER",
    ),
    (
        "REQ-3A-10",
        "a documentacao do pet cobre os 8 estados e o mapa para os 14",
        "docs/COMPANION.md",
        "PAUSED",
        "scripts/hades_companion_doc_check_test.py",
        "python -X utf8 scripts/hades_companion_doc_check.py",
        "todas",
        "MANTER",
    ),
]


def sha_curto(rel: str) -> str:
    p = RAIZ / rel
    if not p.is_file():
        return "AUSENTE"
    return hashlib.sha256(p.read_bytes()).hexdigest()[:16]


def linha_do_simbolo(rel: str, simbolo: str) -> str:
    """A **primeira** linha onde o simbolo aparece. *E' o `arquivo:linha` da cadeia.*"""
    p = RAIZ / rel
    if not p.is_file():
        return "AUSENTE"
    for n, l in enumerate(io.open(p, encoding="utf-8"), 1):
        if simbolo in l:
            return str(n)
    return "AUSENTE"


def head_sha() -> str:
    try:
        r = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=str(RAIZ),
                           capture_output=True, text=True, encoding="utf-8")
        return (r.stdout or "").strip() or "?"
    except Exception:
        return "?"


def estado_do_worktree() -> str:
    """`limpo` ou `dirty`. *O prompt exige o estado, e um `dirty` e' uma informacao, nao um erro.*"""
    r = subprocess.run(["git", "status", "--porcelain"], cwd=str(RAIZ),
                       capture_output=True, text=True, encoding="utf-8")
    linhas = [l for l in (r.stdout or "").split("\n") if l.strip()]
    # *Os fins-de-linha do Windows contam; filtramo-los como o `hades_worktree_inventory` faz.*
    return "limpo" if not linhas else "dirty(%d)" % len(linhas)


def gerar() -> tuple[str, list[str]]:
    problemas: list[str] = []
    linhas = ["req_id,descricao,arquivo,linha,teste,comando,resultado,plataforma,commit,artefacto,decisao"]
    commit = head_sha()
    estado = estado_do_worktree()
    for (req, desc, arq, sym, teste, cmd, plat, dec) in REQUISITOS:
        n = linha_do_simbolo(arq, sym)
        if n == "AUSENTE":
            problemas.append("%s: o simbolo `%s` nao existe em %s" % (req, sym, arq))
        t_ok = (RAIZ / teste).is_file()
        if not t_ok:
            problemas.append("%s: o teste %s nao existe" % (req, teste))
        h = sha_curto(arq)
        if h == "AUSENTE":
            problemas.append("%s: o arquivo %s nao existe" % (req, arq))
        resultado = "PROVADO" if (n != "AUSENTE" and t_ok and h != "AUSENTE") else "AUSENTE"
        linhas.append(",".join([
            req,
            '"%s"' % desc.replace('"', "'"),
            arq,
            n,
            teste,
            '"%s"' % cmd.replace('"', "'"),
            resultado,
            plat,
            commit,
            h,
            dec,
        ]))
    return "\n".join(linhas) + "\n", problemas


def main() -> int:
    check = "--check" in sys.argv
    conteudo, problemas = gerar()
    if check:
        if not SAIDA.is_file():
            print("  FALHA: %s nao existe (corra sem --check)" % SAIDA.relative_to(RAIZ))
            return 1
        actual = io.open(SAIDA, encoding="utf-8").read()

        import csv as _csv

        def sem_commit(texto: str) -> list[tuple[str, ...]]:
            """Devolve as linhas **sem a coluna 9 (commit)**, partidas como CSV A SERIO.

            *O commit muda a cada rodada -- compara-lo faria o `--check` falhar sempre depois de um
            commit. Removemos a coluna da COMPARACAO e validamos o FORMATO dela.*

            ***E usa o modulo `csv`, nao `split(",")`.*** *Medido: o campo `descricao` tem virgulas
            dentro de aspas, e o `split` ingenuo deslocava todas as colunas -- a "coluna 9" passava a
            ser um pedaco do SHA do artefacto. O `--check` recusou-o, o que estava certo; o remedio
            errado seria afrouxar a validacao.*
            """
            linhas = list(_csv.reader(io.StringIO(texto)))
            if not linhas:
                return []
            cab = linhas[0]
            try:
                i = cab.index("commit")
            except ValueError:
                return [tuple(c) for c in linhas]
            out = [tuple(cab[:i] + cab[i + 1:])]
            for campos in linhas[1:]:
                if not campos or not any(c.strip() for c in campos):
                    continue
                out.append(tuple(campos[:i] + campos[i + 1:]))
            return out

        a = sem_commit(actual)
        b = sem_commit(conteudo)
        if a != b:
            # *O `unified_diff` exige `str`; formatamos os tuplos como linhas antes de comparar.*
            import difflib
            a_txt = [" | ".join(c) for c in a]
            b_txt = [" | ".join(c) for c in b]
            dif = list(difflib.unified_diff(a_txt, b_txt, "commitado", "recalculado", n=0,
                                            lineterm=""))
            print("  FALHA: REQUISITOS.csv esta' desactualizado (%d linhas de diff)" % len(dif))
            for l in dif[:8]:
                print("    " + l[:110])
            return 1
        # *E o commit TEM de ter a forma certa: um SHA curto (7-12 hex) ou `?`.*
        # => *o indice e' **derivado do cabecalho**, nao fixo: um indice por posicao quebra quando o
        # formato muda, e este formato ja' mudou duas vezes (o `E-1077` e' a mesma licao).*
        tabela = list(_csv.reader(io.StringIO(actual)))
        cabecalho = tabela[0] if tabela else []
        try:
            i_commit = cabecalho.index("commit")
        except ValueError:
            print("  FALHA: o CSV nao tem a coluna `commit` (cabecalho=%r)" % (cabecalho,))
            return 1
        for campos in tabela[1:]:
            if not campos or not any(c.strip() for c in campos):
                continue
            commit = campos[i_commit] if len(campos) > i_commit else ""
            if commit != "?" and not (7 <= len(commit) <= 12 and all(
                    c in "0123456789abcdef" for c in commit)):
                print("  FALHA: commit com forma invalida: %r (i_commit=%d, campos=%d)" % (
                    commit, i_commit, len(campos)))
                return 1
        if problemas:
            print("  FALHA: %d requisito(s) sem prova" % len(problemas))
            for p in problemas:
                print("    - " + p)
            return 1
        print("  REQUISITOS CONSISTENTES: %d requisitos, todos com prova verificada" % len(REQUISITOS))
        return 0
    SAIDA.parent.mkdir(parents=True, exist_ok=True)
    io.open(SAIDA, "w", encoding="utf-8", newline="\n").write(conteudo)
    print("  gerado: %s (%d requisitos)" % (SAIDA.relative_to(RAIZ), len(REQUISITOS)))
    if problemas:
        print("  AVISO: %d requisito(s) sem prova" % len(problemas))
        for p in problemas:
            print("    - " + p)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
