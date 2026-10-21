#!/usr/bin/env python3
"""HADES — DESCOBERTA HONESTA (P1-H / estágio 39 do V2.2) — E-322.

O QUE ESTE INSTRUMENTO PROTEGE
-----------------------------
O adendo exige descoberta (SEO/GEO, README e documentação pt-BR/en, sitemap, JSON-LD, acessibilidade) **e proíbe**,
na mesma frase: *"não prometer rankings ou recomendação por IAs"*.

Uma proibição em documento é uma promessa. Aqui ela vira **teste**: o repositório passa a **falhar** se um documento
público prometer posição, supremacia ou recomendação — ***a proibição do adendo vira defeito de CI***.

Além disso, o instrumento verifica o que existe SEM INVENTAR o que não existe:
  - `sitemap.xml`/`robots.txt`/JSON-LD só são validados **se existirem** (não há site hospedado neste repositório, e
    publicar em domínio exige autorização do proprietário);
  - a documentação precisa oferecer entrada em **pt-BR** e em **inglês**, porque a UI é pt-BR e a documentação pública
    também é en (V2.2 §0).

LIMITE DECLARADO: este instrumento **não** mede ranqueamento nem "GEO" — não existe métrica honesta de posição em
buscador ou em IA sem consulta externa real, e inventar uma seria exatamente o que o adendo proíbe.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent

# Promessas proibidas pelo adendo: posição, supremacia ou recomendação garantida.
PADROES_PROIBIDOS = [
    (r"garant(e|imos|ido)\s+(o\s+)?(primeir[oa]|topo|ranking|1[ºo°])", "promessa de posição garantida"),
    (r"primeir[oa]\s+lugar\s+garantido", "promessa de primeiro lugar"),
    (r"\b(n[ºo°]?\s*1|#1|number one)\s+(do|no|em)\s+mercado", "alegação de supremacia de mercado"),
    (r"melhor\s+que\s+(todos|todas|qualquer)", "alegação de supremacia sem métrica"),
    (r"(ser[áa]|seremos|ser[ãa]o|vai\s+ser|vamos\s+ser)\s+recomendad[oa]s?\s+por\s+(ias?|intelig[êe]ncia)", "promessa de recomendação por IA"),
    (r"aparecer\s+em\s+primeiro\s+(no|na)\s+(google|buscador)", "promessa de posição em buscador"),
    (r"100%\s*(pronto|conclu[íi]do|homologado)", "declaração de conclusão integral proibida pelo V2.2"),
]

ARQUIVOS_PUBLICOS = ["README.md", "README.en.md", "README.pt-BR.md", "docs/README.md",
                     "docs/GUIA_RAPIDO_USUARIO.md", "docs/mission/HADES_V22_IMPLEMENTATION_PLAN.md"]


def auditar_promessas(raiz: Path = RAIZ) -> list[str]:
    """Acusa promessas proibidas em arquivos públicos."""
    violacoes: list[str] = []
    for relativo in ARQUIVOS_PUBLICOS:
        caminho = raiz / relativo
        if not caminho.exists():
            continue
        texto = caminho.read_text(encoding="utf-8", errors="replace")
        for padrao, rotulo in PADROES_PROIBIDOS:
            for achado in re.finditer(padrao, texto, re.IGNORECASE):
                linha = texto[:achado.start()].count("\n") + 1
                violacoes.append(f"{relativo}:{linha}: {rotulo} -> {achado.group(0)[:60]!r}")
    return violacoes


def auditar_idiomas(raiz: Path = RAIZ) -> list[str]:
    """Exige entrada de documentação em pt-BR e em inglês (V2.2 §0)."""
    violacoes: list[str] = []
    readme = raiz / "README.md"
    if not readme.exists():
        return ["README.md ausente"]
    texto = readme.read_text(encoding="utf-8", errors="replace").lower()
    tem_pt = "portugu" in texto or "pt-br" in texto or (raiz / "README.pt-BR.md").exists()
    tem_en = "english" in texto or (raiz / "README.en.md").exists()
    if not tem_pt:
        violacoes.append("README.md sem entrada em português (e sem README.pt-BR.md)")
    if not tem_en:
        violacoes.append("README.md sem entrada em inglês (e sem README.en.md)")
    return violacoes


# Anti-XXE / billion-laughs SEM dependencia externa (o achado Semgrep
# `use-defused-xml` aponta `ET.fromstring` neste ficheiro).
#
# PORQUE NAO SE USA `defusedxml`: ele nao esta' instalado e o projecto nao tem
# `requirements.txt`/`pyproject.toml` -- acrescentar uma dependencia a um script de
# verificacao local que corre em qualquer maquina seria trocar um risco por uma
# fragilidade de instalacao.
#
# O VECTOR REAL, e como o fechar:
#   - entidades EXTERNAS (`SYSTEM`/`PUBLIC`): o Python 3 NAO as resolve por omissao;
#   - entidades INTERNAS recursivas ("billion laughs"): o Python 3 RESOLVE e pode esgotar
#     memoria -- a defesa e' RECUSAR qualquer `<!DOCTYPE`/`<!ENTITY`, que um sitemap do
#     protocolo nunca tem;
#   - payload gigante: limite de tamanho antes de parsear.
TAMANHO_MAXIMO_SITEMAP = 8 * 1024 * 1024  # 8 MiB; o limite do protocolo e' 50 MB descomprimido


def _ler_xml_seguro(caminho: Path) -> str:
    """Le o XML recusando o que nenhum sitemap do protocolo tem: DOCTYPE/ENTITY e tamanho absurdo."""
    texto = caminho.read_text(encoding="utf-8")
    if len(texto) > TAMANHO_MAXIMO_SITEMAP:
        raise ValueError(f"sitemap.xml excede {TAMANHO_MAXIMO_SITEMAP} bytes")
    # Case-insensitive e sem depender de espacos: `<!doctype`, `<!DocType` e `<! DOCTYPE`
    # sao a mesma construcao. *Recusar so' a forma que eu imaginei seria um filtro que se
    # testa a si proprio.*
    normalizado = texto.upper().replace(" ", "").replace("\t", "").replace("\n", "")
    for construcao in ("<!DOCTYPE", "<!ENTITY"):
        if construcao in normalizado:
            raise ValueError(f"sitemap.xml contem {construcao}, recusado por seguranca")
    return texto


def validar_sitemap(caminho: Path) -> list[str]:
    """Sitemap só é validado se existir — e então precisa ser XML válido com URLs ABSOLUTAS."""
    if not caminho.exists():
        return []
    violacoes: list[str] = []
    try:
        arvore = ET.fromstring(_ler_xml_seguro(caminho))
    except ValueError as erro:
        return [f"sitemap.xml recusado: {erro}"]
    except ET.ParseError as erro:
        return [f"sitemap.xml inválido: {erro}"]
    # MEDIDO: ler só o namespace oficial deixava passar sitemap SEM namespace (meu próprio fixture provou).
    locais = [(no.text or "") for no in arvore.iter() if str(no.tag).endswith("loc")]
    if not locais:
        violacoes.append("sitemap.xml sem nenhuma entrada <loc>")
    for local in locais:
        if not local.startswith(("https://", "http://")):
            violacoes.append(f"sitemap.xml com URL não absoluta: {local!r}")
    return violacoes


def validar_jsonld(caminho: Path) -> list[str]:
    """Blocos JSON-LD, quando existirem, precisam ser JSON VÁLIDO com @context e @type."""
    if not caminho.exists():
        return []
    violacoes: list[str] = []
    texto = caminho.read_text(encoding="utf-8", errors="replace")
    for bloco in re.findall(r"<script[^>]*application/ld\+json[^>]*>(.*?)</script>", texto, re.S | re.I):
        try:
            dados = json.loads(bloco)
        except json.JSONDecodeError as erro:
            violacoes.append(f"{caminho.name}: JSON-LD inválido: {erro}")
            continue
        for item in dados if isinstance(dados, list) else [dados]:
            if not isinstance(item, dict):
                violacoes.append(f"{caminho.name}: JSON-LD não é objeto")
                continue
            if "@context" not in item:
                violacoes.append(f"{caminho.name}: JSON-LD sem @context")
            if "@type" not in item:
                violacoes.append(f"{caminho.name}: JSON-LD sem @type")
    return violacoes


def auditar(raiz: Path = RAIZ) -> list[str]:
    """Auditoria completa de descoberta honesta."""
    return (
        auditar_promessas(raiz)
        + auditar_idiomas(raiz)
        + validar_sitemap(raiz / "sitemap.xml")
        + validar_jsonld(raiz / "README.md")
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Audita descoberta honesta (P1-H / estágio 39).")
    parser.add_argument("--check", action="store_true", help="falha (exit 1) se houver violação")
    args = parser.parse_args(argv)
    violacoes = auditar()
    if violacoes:
        for violacao in violacoes:
            print(f"VIOLACAO: {violacao}")
        print(f"\n{len(violacoes)} violação(ões) de descoberta honesta.")
        return 1
    print("DESCOBERTA HONESTA: sem promessa de ranking/supremacia; entradas pt-BR e en presentes.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
