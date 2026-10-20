"""Gera THIRD_PARTY_NOTICES.md + HADES_SBOM_GO.csv + HADES_SBOM_NPM.csv + HADES_SBOM_LICENCAS.csv.

Rota reproduzivel e 100% OFFLINE: `go list -m all` + a cache de modulos para o Go;
`package-lock.json` (lockfileVersion 3) para o npm. Nada e' descarregado ou instalado.

Historico das correccoes (todas achadas por LER a propria saida ou por medir o objecto certo):
  E-790  (1) o `%%` literal no texto gerado;
         (2) modulos `(nenhum)` juntavam DOIS factos: directorio ausente vs presente sem licenca.
  E-791  (3) cobertura npm: o lock sempre parseou - quem falhava era o `ConvertFrom-Json` do PS 5.1;
         (4) ENDURECIMENTO: se houver modulos no grafo e ZERO licencas, FALHA (exit 2).
  E-792  (5) classificacao por familia + terceiro CSV (triagem, nao parecer juridico);
         (6) a familia Go passou a sair do TEXTO INTEIRO do ficheiro:
             *muitos `LICENSE` comecam por `Copyright (c) ...`*;
         (7) siglas passaram a regex com `\\b`: **`MPL` casava dentro de "eXaMPLe"** e isso
             classificou **136/136** ficheiros como COPYLEFT_FRACO.
  E-793  (8) tabela de IDENTIFICADORES SPDX (0BSD, Python-2.0, CC-BY-3.0/4.0, MIT-0,
             BlueOak-1.0.0, Artistic-2.0, BSL-1.0, CC-BY-SA-4.0, ...) - era o que faltava
             para os 4 `DESCONHECIDA` do npm;
         (9) SEGUNDA FONTE para o npm: quando o lockfile nao traz `license`, o campo
             `license` do `node_modules/<pkg>/package.json` e' lido - era o que faltava
             para os 18 `SEM_TEXTO`. *Nenhuma das duas fontes sai da maquina.*

Uso:
  python scripts/hades_third_party_notices.py [--go <go>] [--raiz <dir>] [--app <dir do app web>]
"""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path

PADROES = re.compile(r"^(LICENSE|LICENCE|COPYING|NOTICE|UNLICENSE)", re.IGNORECASE)
NODE_MODULES = "node_modules/"

# Ordem IMPORTA: copyleft antes de permissiva; FRASES antes de siglas.
# *Uma sigla de 3 letras casa dentro de palavras comuns:* `MPL` casa em "eXaMPLe" -
# foi isso que classificou **TODOS** os 136 ficheiros Go como COPYLEFT_FRACO.
# Por isso: regex com `\b` (limite de palavra) e frases canonicas.
FAMILIAS = [
    (r"\bAGPL\b|AFFERO GENERAL PUBLIC", "COPYLEFT_FORTE"),
    (r"\bLGPL\b|LESSER GENERAL PUBLIC", "COPYLEFT_FRACO"),
    (r"MOZILLA PUBLIC LICENSE|\bMPL\b", "COPYLEFT_FRACO"),
    (r"ECLIPSE PUBLIC LICENSE|\bEPL\b", "COPYLEFT_FRACO"),
    (r"\bCDDL\b|COMMON DEVELOPMENT AND DISTRIBUTION", "COPYLEFT_FRACO"),
    (r"\bSSPL\b|SERVER SIDE PUBLIC", "NAO_LIVRE"),
    (r"\bBUSL\b|BUSINESS SOURCE LICENSE", "NAO_LIVRE"),
    (r"CC-BY-NC|NONCOMMERCIAL|\bPROPRIETARY\b|\bUNLICENSED\b", "NAO_LIVRE"),
    (r"\bGPL\b|GNU GENERAL PUBLIC LICENSE", "COPYLEFT_FORTE"),
    (r"APACHE LICENSE|\bAPACHE-2\.0\b", "PERMISSIVA"),
    (r"PERMISSION IS HEREBY GRANTED, FREE OF CHARGE", "PERMISSIVA"),
    (r"PERMISSION TO USE, COPY, MODIFY", "PERMISSIVA"),
    (r"REDISTRIBUTION AND USE IN SOURCE AND BINARY", "PERMISSIVA"),
    # Identificadores SPDX comuns (E-793): o `\b` e' o que os distingue de prosa.
    (r"\b0BSD\b|\bBSD-[234]-CLAUSE\b|\bMIT-0\b|\bBLUEOAK-1\.0\.0\b|\bARTISTIC-2\.0\b"
     r"|\bBSL-1\.0\b|\bCC-BY-3\.0\b|\bCC-BY-4\.0\b|\bCC-BY-SA-4\.0\b|\bPYTHON-2\.0\b"
     r"|\bPSF-2\.0\b|\bZLIB\b|\bWTFPL\b|\bUNLICENSE\b|\bCC0-1\.0\b|\bISC\b",
     "PERMISSIVA"),
    (r"\bMIT\b|\bBSD\b|PUBLIC DOMAIN|PYTHON SOFTWARE", "PERMISSIVA"),
]


def classificar(texto):
    """TRIAGEM por FRASES e siglas COM LIMITE DE PALAVRA sobre o texto todo.

    Nao e' parecer juridico. O `\\b` e' essencial: sem ele, `MPL` casa em "example".
    """
    t = (texto or "").upper()
    if not t.strip():
        return "SEM_TEXTO"
    for padrao, familia in FAMILIAS:
        if re.search(padrao, t):
            return familia
    return "DESCONHECIDA"


def escapar_modulo(modulo):
    return re.sub(r"[A-Z]", lambda m: "!" + m.group(0).lower(), modulo)


def so_ascii(texto):
    return str(texto).encode("ascii", "replace").decode("ascii")


def texto_do_ficheiro(caminho):
    try:
        return caminho.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return ""


def primeira_linha_util(texto):
    for linha in (texto or "").splitlines():
        linha = linha.strip()
        if linha:
            return so_ascii(linha[:120])
    return "(vazio)"


def sha256(caminho):
    try:
        return hashlib.sha256(caminho.read_bytes()).hexdigest()
    except OSError:
        return ""


def licenca_do_package_json(base):
    """SEGUNDA FONTE (E-793): o `license` do package.json do proprio pacote."""
    pj = base / "package.json"
    if not pj.is_file():
        return ""
    try:
        dados = json.loads(pj.read_text(encoding="utf-8", errors="replace"))
    except Exception:
        return ""
    lic = dados.get("license") or dados.get("licenses")
    if isinstance(lic, dict):
        lic = lic.get("type")
    if isinstance(lic, list):
        partes = []
        for item in lic:
            if isinstance(item, dict):
                partes.append(str(item.get("type") or ""))
            else:
                partes.append(str(item))
        lic = " OR ".join(p for p in partes if p)
    return so_ascii(str(lic or ""))


def lista_modulos(go, raiz):
    saida = subprocess.run([go, "list", "-m", "all"], cwd=str(raiz),
                           capture_output=True, text=True, encoding="utf-8", errors="replace")
    if saida.returncode != 0:
        print("ERRO: `go list -m all` devolveu", saida.returncode, file=sys.stderr)
        print(saida.stderr[-800:], file=sys.stderr)
        raise SystemExit(1)
    modulos = []
    for linha in saida.stdout.splitlines():
        partes = linha.split()
        if len(partes) == 2:
            modulos.append((partes[0], partes[1]))
    return modulos


def inventario_go(cache, modulos):
    """linha = (modulo, versao, ficheiro, primeira_linha, bytes, sha256, familia)."""
    linhas, sem_licenca, nao_extraido = [], [], []
    for modulo, versao in modulos:
        base = cache / (escapar_modulo(modulo) + "@" + versao)
        achados = []
        if base.is_dir():
            for entrada in sorted(base.iterdir()):
                if entrada.is_file() and PADROES.match(entrada.name):
                    achados.append(entrada)
        if not achados:
            (sem_licenca if base.is_dir() else nao_extraido).append(modulo)
            linhas.append((modulo, versao, "", "", 0, "", ""))
            continue
        for f in achados:
            texto = texto_do_ficheiro(f)
            linhas.append((modulo, versao, f.name, primeira_linha_util(texto),
                           f.stat().st_size, sha256(f), classificar(texto)))
    return linhas, sem_licenca, nao_extraido


def inventario_npm(lock, node_modules):
    dados = json.loads(lock.read_text(encoding="utf-8"))
    pacotes = dados.get("packages") or {}
    linhas, sem_licenca = [], []
    resgatados = 0
    for chave, meta in sorted(pacotes.items()):
        if not chave.startswith(NODE_MODULES) or not isinstance(meta, dict):
            continue
        nome = chave.rsplit(NODE_MODULES, 1)[-1]
        versao = str(meta.get("version") or "")
        lic = meta.get("license")
        if isinstance(lic, dict):
            lic = lic.get("type") or json.dumps(lic, ensure_ascii=False)
        lic = so_ascii(lic or "")
        base = node_modules / nome
        ficheiro, tamanho, h, familia = "", 0, "", ""
        if base.is_dir():
            for entrada in sorted(base.iterdir()):
                if entrada.is_file() and PADROES.match(entrada.name):
                    texto = texto_do_ficheiro(entrada)
                    ficheiro, tamanho, h = entrada.name, entrada.stat().st_size, sha256(entrada)
                    familia = classificar(lic or texto)
                    break
        if not lic:
            alternativa = licenca_do_package_json(base)
            if alternativa:
                lic = alternativa
                resgatados += 1
        if not familia:
            familia = classificar(lic)
        if not lic and not ficheiro:
            sem_licenca.append(nome)
        linhas.append((nome, versao, lic, ficheiro, tamanho, h, familia))
    return linhas, sem_licenca, resgatados


def gravar(caminho, conteudo, check, divergencias):
    """Grava `conteudo` em `caminho` - ou, em modo `--check`, COMPARA e nao escreve.

    E-913 (medido): o gerador escrevia os **quatro** artefactos (notices + 3 SBOM) e **nao tinha modo de
    verificacao**. *Os outros sete instrumentos do repo tem `--check`; este nao tinha* - e o §GATES pede
    «confirmacao de que os **geradores nao alteram inesperadamente a arvore**».

    Em modo `--check` a arvore **nao e' tocada**: o conteudo e' comparado com o que esta' no disco, e cada
    divergencia e' contada e nomeada. *Um `--check` que escrevesse para depois restaurar teria uma janela em
    que a arvore esta' alterada - e nada garante que a restauracao corra.*
    """
    if check:
        atual = None
        if caminho.exists():
            atual = caminho.read_text(encoding="utf-8")
        if atual != conteudo:
            divergencias.append(caminho.name)
            print("  DIVERGE: %s (disco %d bytes, gerado %d bytes)"
                  % (caminho.name, len(atual) if atual is not None else -1, len(conteudo)))
        return
    caminho.write_text(conteudo, encoding="utf-8", newline="\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--go", default=os.environ.get("HADES_GO", "go"))
    ap.add_argument("--raiz", default=".")
    ap.add_argument("--app", default=os.path.join("app", "ui", "app"))
    # E-913: sem `--check` nao havia como provar que os artefactos em disco correspondem ao gerador.
    ap.add_argument("--check", action="store_true",
                    help="nao escreve: compara o que seria gerado com o que esta' no disco e sai != 0 se divergir")
    args = ap.parse_args()
    raiz = Path(args.raiz).resolve()
    app = (raiz / args.app).resolve()

    gopath = Path(os.environ.get("GOPATH") or (Path.home() / "go"))
    cache = gopath / "pkg" / "mod"

    modulos = lista_modulos(args.go, raiz)
    linhas_go, sem_go, nao_extraido = inventario_go(cache, modulos)
    com_go = [l for l in linhas_go if l[2]]

    lock = app / "package-lock.json"
    linhas_npm, sem_npm, resgatados = ([], [], 0)
    if lock.is_file():
        try:
            linhas_npm, sem_npm, resgatados = inventario_npm(lock, app / "node_modules")
        except Exception as e:
            print("AVISO: npm nao medido:", type(e).__name__, str(e)[:160], file=sys.stderr)
    com_npm = [l for l in linhas_npm if l[2] or l[3]]

    print("GO  : grafo=%d com_licenca=%d nao_extraidos=%d extraidos_sem_licenca=%d ficheiros=%d"
          % (len(modulos), len({l[0] for l in com_go}), len(nao_extraido), len(sem_go), len(com_go)))
    print("NPM : pacotes=%d com_licenca=%d sem_licenca=%d ficheiros_no_disco=%d resgatados_do_package_json=%d"
          % (len(linhas_npm), len(com_npm), len(sem_npm), len([l for l in linhas_npm if l[3]]), resgatados))

    # ENDURECIMENTO: um documento de atribuicao sem NENHUMA licenca nao e' sucesso.
    if len(modulos) and not com_go:
        print("ERRO: %d modulos no grafo e ZERO licencas encontradas na cache (%s)." % (len(modulos), cache),
              file=sys.stderr)
        print("      Cache ausente ou GOPATH errado: um documento vazio NAO deve parecer sucesso.",
              file=sys.stderr)
        raise SystemExit(2)

    def conta(itens, idx):
        d = {}
        for it in itens:
            d[it[idx]] = d.get(it[idx], 0) + 1
        return d

    resumo_go = conta(com_go, 6)
    resumo_npm = conta(com_npm, 6)
    print("FAMILIAS GO :", json.dumps(resumo_go, sort_keys=True))
    print("FAMILIAS NPM:", json.dumps(resumo_npm, sort_keys=True))

    atencao_go = [l for l in com_go if l[6] in ("COPYLEFT_FORTE", "COPYLEFT_FRACO", "NAO_LIVRE", "DESCONHECIDA")]
    atencao_npm = [l for l in com_npm if l[6] in ("COPYLEFT_FORTE", "NAO_LIVRE", "DESCONHECIDA")]
    print("ATENCAO GO  (nao permissiva/desconhecida):", len(atencao_go))
    print("ATENCAO NPM (forte/nao livre/desconhecida) :", len(atencao_npm))

    # E-793: a linha com o `%d` tem de ser formatada FORA da tupla. Fazê-lo na linha
    # seguinte (como antes) aplicava `% resgatados` a um texto sem especificador e
    # abortava o gerador **depois** dos calculos e **antes** de escrever os ficheiros -
    # deixando os documentos antigos em disco e o bundle a ser montado com eles.
    linha_fonte_npm = (
        "A licenca vem do **campo `license` do lockfile**; quando ele falta, do **campo `license` "
        "do `package.json` do pacote** (%d pacotes obtidos por essa segunda via). O `sha256` so' "
        "aparece quando existe ficheiro de licenca em `node_modules`." % resgatados
    )
    md = [
        "# THIRD_PARTY_NOTICES",
        "",
        "Documento **gerado** por `scripts/hades_third_party_notices.py`. Nao editar a mao.",
        "**100% offline**: nada foi descarregado nem instalado.",
        "",
        "Fontes: `go list -m all` + cache `GOPATH/pkg/mod` (Go); `package-lock.json` e, quando o",
        "lockfile nao traz `license`, o `package.json` do proprio pacote (npm).",
        "",
        "## 0. Resumo por familia (TRIAGEM, nao parecer juridico)",
        "",
        "A familia sai de **frases e siglas com limite de palavra** no texto da licenca (Go:",
        "ficheiro inteiro; npm: campo `license`). **Nao** decide compatibilidade com a MIT",
        "deste projecto: **nomeia o que exige revisao humana**.",
        "",
        "| familia | Go | npm |",
        "|---|---|---|",
    ]
    for fam in sorted(set(list(resumo_go.keys()) + list(resumo_npm.keys()))):
        md.append("| %s | %d | %d |" % (fam, resumo_go.get(fam, 0), resumo_npm.get(fam, 0)))
    md += ["", "### Fora do permissivo ou nao classificada, no Go (%d)" % len(atencao_go), ""]
    md += ["- `%s@%s` - %s (%s)" % (so_ascii(l[0]), so_ascii(l[1]), so_ascii(l[3]), l[6])
           for l in atencao_go]
    md += ["", "### Forte/nao-livre/desconhecida, no npm (%d)" % len(atencao_npm), ""]
    md += ["- `%s@%s` - %s (%s)" % (so_ascii(l[0]), so_ascii(l[1]), so_ascii(l[2]), l[6])
           for l in atencao_npm]
    md += [
        "",
        "## 1. Dependencias Go",
        "",
        "| modulo | versao | ficheiro | primeira linha | familia | bytes | sha256 |",
        "|---|---|---|---|---|---|---|",
    ]
    for modulo, versao, nome, primeira, tamanho, h, familia in linhas_go:
        md.append("| `%s` | `%s` | `%s` | %s | %s | %d | `%s` |" % (
            so_ascii(modulo), so_ascii(versao), so_ascii(nome) or "(nenhum)", primeira or "-",
            familia or "-", tamanho, (h[:16] + "...") if h else ""))
    md += ["", "### Go: no grafo mas NAO extraidos na cache local (%d)" % len(nao_extraido), ""]
    md += ["*A cache contem os modulos efectivamente compilados neste host.*", ""]
    md += ["- `%s`" % so_ascii(m) for m in nao_extraido]
    md += ["", "### Go: extraidos mas SEM ficheiro de licenca (%d)" % len(sem_go), ""]
    md += ["- `%s`" % so_ascii(m) for m in sem_go]
    md += [
        "",
        "## 2. Dependencias npm (`%s`)" % so_ascii(args.app.replace(os.sep, "/")),
        "",
        linha_fonte_npm,
        "",
        "| pacote | versao | licenca | familia | ficheiro | bytes | sha256 |",
        "|---|---|---|---|---|---|---|",
    ]
    for nome, versao, lic, ficheiro, tamanho, h, familia in linhas_npm:
        md.append("| `%s` | `%s` | %s | %s | `%s` | %d | `%s` |" % (
            so_ascii(nome), so_ascii(versao), lic or "(vazio)", familia or "-",
            so_ascii(ficheiro) or "(nenhum)", tamanho, (h[:16] + "...") if h else ""))
    md += ["", "### npm: SEM licenca no lockfile, no package.json e SEM ficheiro no disco (%d)" % len(sem_npm), ""]
    md += ["- `%s`" % so_ascii(m) for m in sem_npm]
    md += [
        "",
        "## Escopo declarado",
        "",
        "* A classificacao Go usa o **texto inteiro** do ficheiro de licenca; a do npm usa o campo",
        "  `license` do lockfile ou do `package.json`. **Nao** e' auditoria SPDX nem parecer juridico.",
        "* **Nao** cobre licenca de **modelos**, de `assets/`, nem dos binarios do runner (ver `dist/`).",
        "* A decisao final de **compatibilidade com a MIT deste projecto** continua **por fazer**:",
        "  este documento diz **onde olhar**.",
        "",
        "## Reproduzir",
        "",
        "```",
        "python scripts/hades_third_party_notices.py",
        "```",
        "",
    ]
    check = args.check
    divergencias = []
    gravar(raiz / "THIRD_PARTY_NOTICES.md", "\n".join(md), check, divergencias)

    linhas = ["modulo,versao,ficheiro,bytes,sha256,familia"]
    for modulo, versao, nome, _p, tamanho, h, familia in linhas_go:
        linhas.append("%s,%s,%s,%d,%s,%s" % (modulo, versao, nome, tamanho, h, familia))
    gravar(raiz / "HADES_SBOM_GO.csv", "\n".join(linhas) + "\n", check, divergencias)

    linhas = ["pacote,versao,licenca,ficheiro,bytes,sha256,familia"]
    for nome, versao, lic, ficheiro, tamanho, h, familia in linhas_npm:
        linhas.append("%s,%s,%s,%s,%d,%s,%s" % (nome, versao, lic, ficheiro, tamanho, h, familia))
    gravar(raiz / "HADES_SBOM_NPM.csv", "\n".join(linhas) + "\n", check, divergencias)

    linhas = ["ecossistema,item,versao,licenca_texto,familia", ]
    for modulo, versao, nome, primeira, _t, _h, familia in linhas_go:
        if nome:
            linhas.append("go,%s,%s,%s,%s" % (modulo, versao, primeira.replace(",", ";"), familia))
    for nome, versao, lic, _f, _t, _h, familia in linhas_npm:
        if lic:
            linhas.append("npm,%s,%s,%s,%s" % (nome, versao, lic.replace(",", ";"), familia))
    gravar(raiz / "HADES_SBOM_LICENCAS.csv", "\n".join(linhas) + "\n", check, divergencias)

    resumo = {
        "go_modulos_no_grafo": len(modulos),
        "go_com_licenca": len({l[0] for l in com_go}),
        "go_nao_extraidos": len(nao_extraido),
        "go_extraidos_sem_licenca": len(sem_go),
        "npm_pacotes": len(linhas_npm),
        "npm_com_licenca": len(com_npm),
        "npm_sem_licenca": len(sem_npm),
        "npm_ficheiros_no_disco": len([l for l in linhas_npm if l[3]]),
        "npm_resgatados_package_json": resgatados,
        "familias_go": resumo_go,
        "familias_npm": resumo_npm,
        "atencao_go": len(atencao_go),
        "atencao_npm": len(atencao_npm),
        "notices": "THIRD_PARTY_NOTICES.md",
        "sbom_go": "HADES_SBOM_GO.csv",
        "sbom_npm": "HADES_SBOM_NPM.csv",
        "sbom_licencas": "HADES_SBOM_LICENCAS.csv",
    }
    print(json.dumps(resumo, ensure_ascii=False))
    if check:
        if divergencias:
            print("  FALHA: %d artefacto(s) DIVERGEM do gerador: %s" % (len(divergencias), ", ".join(divergencias)))
            print("         rode sem `--check` para regerar; a arvore NAO foi tocada.")
            return 1
        print("  SBOM/NOTICES: 4 artefactos reproduziveis - o disco corresponde ao gerador.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
