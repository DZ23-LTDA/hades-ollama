"""Regenera o `docs/audit/FINDINGS.json` a partir da API do GitHub, e o `FINDINGS.md` do JSON.

POR QUE DOIS PASSOS, E NAO UM.

Se o `FINDINGS.md` fosse gerado **directamente** da API, o `--check` precisaria de **rede** -- e um
pacote de auditoria que so' se verifica com internet nao se verifica num ambiente isolado.

=> por isso a captura (`--capturar`) escreve o `FINDINGS.json` **versionado**, com a **data** e o
**SHA** em que foi lida, e a geracao (`--check` / sem argumento) le' **so'** o JSON. *O JSON e' a
prova do que a API disse naquele dia; o MD e' a sua leitura.*

O QUE ESTE SCRIPT NAO FAZ.

- **Nao marca nenhum finding como resolvido.** *O §REVISAO HUMANA proibe: nao resolvo threads no
  GitHub sem autorizacao.* => o JSON traz o estado **como a API o devolveu**, e o MD **nao** afirma
  que algum foi corrigido.
- **Nao julga se um finding e' valido.** *O prompt diz: comentario de bot e' hipotese, nao facto.* =>
  o MD diz o que o bot disse, e o `E-1074`/`E-1070` mostram como se valida um finding contra o
  codigo.
"""
import argparse
import io
import json
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
PACOTE = RAIZ / "docs" / "audit"
JSON_PATH = PACOTE / "FINDINGS.json"
MD_PATH = PACOTE / "FINDINGS.md"

SEV = ("Critical", "Major", "Minor")


def capturar() -> int:
    """Le' os review comments da API e escreve o JSON. **Precisa de rede e de `gh`.**"""
    r = subprocess.run(
        ["gh", "api", "repos/DZ23-LTDA/hades-ollama/pulls/438/comments", "--paginate"],
        capture_output=True, text=True, encoding="utf-8", errors="replace")
    if r.returncode != 0:
        print("  ERRO: gh api falhou (exit %d)" % r.returncode)
        print("  " + (r.stderr or "")[:300])
        return 1
    arr = json.loads(r.stdout)

    # o SHA do HEAD, para o JSON dizer de que estado foi lido
    sha = subprocess.run(["git", "rev-parse", "--short", "HEAD"], cwd=str(RAIZ),
                         capture_output=True, text=True).stdout.strip()
    import datetime
    agora = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    findings = []
    for i, c in enumerate(arr, 1):
        corpo = re.sub(r"\r?\n", " ", c.get("body") or "")
        # *O `**` fica ANTES do emoji* (`**🟠 Major**`), logo `"**Major**" in corpo` **falha**.
        # => procura-se a palavra, nao a forma decorada: o emoji e' decoracao, a severidade e' a
        # palavra. ~~`"**" + s + "**" in corpo`~~ -> `s in corpo`.
        sev = "?"
        for s in SEV:
            if s in corpo:
                sev = s
                break
        # **O cabecalho tem TRES negritos de decoracao e o QUARTO e' o titulo**:
        #   **<categoria>** | **<severidade>** | **<esforco>**  **<titulo>**
        # => *a primeira versao apanhou `negritos[1]`, que e' a **severidade*** -- e o FINDINGS.md
        # saiu com `Major` como titulo. *Um indice por posicao fixa quebra quando a decoracao muda;
        # por isso o titulo e' procurado **depois** do esforco, e nao numa posicao.*
        negritos = re.findall(r"\*\*([^*]{3,160})\*\*", corpo)
        categoria = negritos[0] if negritos else "?"
        _idx_esforco = next((i for i, b in enumerate(negritos)
                             if "Quick win" in b or "Heavy lift" in b), None)
        if _idx_esforco is not None and len(negritos) > _idx_esforco + 1:
            titulo = negritos[_idx_esforco + 1]
        else:
            titulo = negritos[1] if len(negritos) > 1 else (negritos[0] if negritos else corpo[:80])
        findings.append({
            "n": i,
            "id": c.get("id"),
            "autor": (c.get("user") or {}).get("login"),
            "path": c.get("path"),
            "linha": c.get("line"),
            "criado_em": c.get("created_at"),
            "severidade": sev,
            "categoria": categoria,
            "titulo": titulo,
            "trecho": corpo[:400],
        })

    pacote = {
        "schema": "hades.findings/v1",
        "fonte": "GitHub API — pull/438/comments",
        "repo": "DZ23-LTDA/hades-ollama",
        "pr": 438,
        "sha_do_checkout": sha,
        "capturado_em": agora,
        "total": len(findings),
        "findings": findings,
    }
    PACOTE.mkdir(parents=True, exist_ok=True)
    JSON_PATH.write_text(json.dumps(pacote, ensure_ascii=False, indent=2) + "\n",
                         encoding="utf-8", newline="")
    print("  capturados %d findings -> docs/audit/FINDINGS.json (sha=%s)" % (len(findings), sha))
    return 0


def gerar_md() -> str:
    d = json.loads(JSON_PATH.read_text(encoding="utf-8"))
    findings = d["findings"]
    por_sev = {s: [f for f in findings if f["severidade"] == s] for s in SEV}
    outros = [f for f in findings if f["severidade"] not in SEV]

    L = [
        "# FINDINGS do PR #438 — como foram lidos, e o que NÃO se afirma",
        "",
        "**Este ficheiro é gerado de `FINDINGS.json`**, que por sua vez foi capturado da GitHub API em",
        "**%s** sobre o checkout `%s`." % (d["capturado_em"], d["sha_do_checkout"]),
        "",
        "**Total: %d** — %s." % (d["total"], " · ".join("**%d %s**" % (len(por_sev[s]), s) for s in SEV if por_sev[s])),
        "",
        "## O que este ficheiro NÃO afirma",
        "",
        "- **não afirma que algum finding foi corrigido** — nenhum foi marcado como resolvido no GitHub;",
        "- **não afirma que um finding do bot é verdadeiro** — *o prompt diz: comentário de bot é",
        "  hipótese, não facto*; validar exige confrontar com o código e criar contraprova;",
        "- **não afirma que a lista está completa** — são os **review comments** que a API devolveu",
        "  naquele instante; *uma thread sem comentário de linha não apareceria aqui.*",
        "",
        "## Como validar um finding, em vez de o aceitar",
        "",
        "O `E-1070` e o `E-1074` são o modelo: **reproduzir o defeito num checkout limpo**, medir, e só",
        "então decidir. *«Comentário de bot como hipótese não confiável: confirme no código atual, crie",
        "contraprova e só então corrija.»*",
        "",
        "## Por severidade",
        "",
    ]
    for s in SEV:
        if not por_sev[s]:
            continue
        L += ["### %s (%d)" % (s, len(por_sev[s])), "",
              "| # | Ficheiro | Título |", "| --- | --- | --- |"]
        for f in por_sev[s]:
            L.append("| %d | `%s`%s | %s |" % (
                f["n"], f["path"], (":%d" % f["linha"]) if f["linha"] else "",
                f["titulo"].replace("|", "\\|")[:150]))
        L.append("")
    if outros:
        L += ["### Sem severidade reconhecida (%d)" % len(outros), "",
              "| # | Ficheiro | Título |", "| --- | --- | --- |"]
        for f in outros:
            L.append("| %d | `%s` | %s |" % (f["n"], f["path"], f["titulo"].replace("|", "\\|")[:150]))
        L.append("")

    L += [
        "## O que fazer a seguir",
        "",
        "**Nenhum destes findings foi corrigido por este trabalho.** O `E-1076` entregou o pacote de",
        "auditoria; os findings são **entrada** para a revisão humana, não saída de uma correcção",
        "automática. *Um pacote que dissesse «corrigido» sem um teste que discrimina o defeito estaria",
        "a inventar prova.*",
        "",
    ]
    return "\n".join(L)


def main() -> int:
    ap = argparse.ArgumentParser(description="Relatorio dos findings do PR, a partir do JSON capturado.")
    ap.add_argument("--capturar", action="store_true", help="le' a API do GitHub (precisa de rede)")
    ap.add_argument("--check", action="store_true", help="compara o MD com o JSON")
    args = ap.parse_args()

    if args.capturar:
        return capturar()

    if not JSON_PATH.is_file():
        print("  ERRO: docs/audit/FINDINGS.json nao existe; rode com --capturar")
        return 2

    md = gerar_md()
    if args.check:
        atual = MD_PATH.read_text(encoding="utf-8") if MD_PATH.is_file() else ""
        if atual != md:
            print("  FINDINGS DIVERGENTE do JSON capturado")
            print("  Rode: python -X utf8 scripts/hades_findings_report.py")
            return 1
        print("  FINDINGS CONSISTENTE: o MD bate com o JSON capturado.")
        return 0

    PACOTE.mkdir(parents=True, exist_ok=True)
    MD_PATH.write_text(md, encoding="utf-8", newline="")
    print("  gerado: docs/audit/FINDINGS.md")
    return 0


if __name__ == "__main__":
    sys.exit(main())
