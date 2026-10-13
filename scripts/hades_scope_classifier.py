"""E-352: a catraca passa a CLASSIFICAR cada uso de sanitizeIDComponent (caminho x identificador x comentario).

MEDIDO EM E-351: 27 arquivos usam `sanitizeIDComponent`; a catraca cobria so' os que montam CAMINHO e 5 ficaram fora -
dois deles montando IDENTIFICADOR a partir da organizacao. A classificacao foi MANUAL naquela fatia, e classificacao
manual nao escala nem protege o proximo arquivo.

***Rede que depende de alguem lembrar de olhar nao e' rede.***

CATEGORIAS:
  comentario    - a ocorrencia esta num comentario: ignorada (era o falso positivo de media_pipeline.go)
  caminho       - o valor vira caminho/arquivo: exige guarda crua ou dispensa motivada
  identificador - o valor vira identificador ("prod_"+..., "shift_"+..., ID:): exige guarda crua ou dispensa motivada
  indefinido    - nao se sabe o que e': VIOLACAO. *O desconhecido nao pode passar em silencio para virar surpresa.*
"""

import json
import re
from pathlib import Path

PALAVRAS_CAMINHO = ("filepath.Join", "path", "caminho", "lock", "Root", ".json", ".lock", "Dir")
PADRAO_IDENTIFICADOR = re.compile(r'(?:"[A-Za-z0-9_]+_"|\bID\b|Id:|id\s*[:=])\s*\+?\s*[^;\n]*sanitizeIDComponent\('
                                  r'|sanitizeIDComponent\([^)]*\)\s*\+')


def classificar_uso(linha: str) -> str:
    """Classifica UMA ocorrência de sanitizeIDComponent numa linha."""
    texto = linha.strip()
    # A DEFINIÇÃO da própria função não é uso dela (medido: supervisor_incidents.go:72 aparecia como indefinido).
    if "func sanitizeIDComponent(" in linha:
        return "definicao"
    if texto.startswith("//"):
        return "comentario"
    # Comentário no meio da linha: a ocorrência pode estar depois de `//`
    posicao = linha.find("sanitizeIDComponent(")
    if posicao > 0 and "//" in linha[:posicao]:
        return "comentario"
    if any(palavra in linha for palavra in PALAVRAS_CAMINHO):
        return "caminho"
    if PADRAO_IDENTIFICADOR.search(linha):
        return "identificador"
    return "indefinido"


def classificar_usos(diretorio: Path = None) -> dict:
    """Devolve {arquivo: {"caminho": [...], "identificador": [...], "comentario": [...], "indefinido": [...]}}."""
    raiz = diretorio or (Path(__file__).resolve().parent.parent / "internal")
    resultado: dict = {}
    if not raiz.exists():
        return resultado
    for caminho in sorted(raiz.rglob("*.go")):
        if caminho.name.endswith("_test.go"):
            continue
        try:
            linhas = caminho.read_text(encoding="utf-8", errors="replace").split("\n")
        except OSError:
            continue
        for numero, linha in enumerate(linhas, 1):
            if "sanitizeIDComponent(" not in linha:
                continue
            categoria = classificar_uso(linha)
            chave = caminho.name
            if categoria == "definicao":
                continue
            resultado.setdefault(chave, {"caminho": [], "identificador": [], "comentario": [], "indefinido": []})
            resultado[chave][categoria].append("%s:%d" % (caminho.name, numero))
    return resultado


def resumo_da_classificacao(diretorio: Path = None) -> dict:
    usos = classificar_usos(diretorio)
    total = {"caminho": 0, "identificador": 0, "comentario": 0, "indefinido": 0}
    for dados in usos.values():
        for categoria, itens in dados.items():
            total[categoria] += len(itens)
    total["arquivos"] = len(usos)
    return total


# DECLARAÇÕES do que a classificação automática não consegue decidir sozinha.
# Cada linha aqui é uma decisão HUMANA registrada, com motivo — e o instrumento REPROVA indefinido sem declaração.
DECLARACOES_INDEFINIDAS = {
    "media_pipeline.go:358": ("caminho multi-linha: o valor vira `organizacao` e é usado em `filepath.Join` na linha "
                              "seguinte, dentro de `MediaOrgRoot`, que JÁ valida a organização crua (E-336)."),
}


def indefinidos_nao_declarados(diretorio: Path = None) -> list:
    usos = classificar_usos(diretorio)
    pendentes = []
    for dados in usos.values():
        for local in dados["indefinido"]:
            if local not in DECLARACOES_INDEFINIDAS:
                pendentes.append(local)
    return pendentes

if __name__ == "__main__":
    usos = classificar_usos()
    total = resumo_da_classificacao()
    print("USOS DE sanitizeIDComponent por categoria: %s" % json.dumps(total, ensure_ascii=False))
    pendentes = indefinidos_nao_declarados()
    if pendentes:
        for local in pendentes:
            print("VIOLACAO: uso INDEFINIDO sem declaração: %s (o desconhecido não pode passar em silêncio)" % local)
        raise SystemExit(1)
    for local, motivo in sorted(DECLARACOES_INDEFINIDAS.items()):
        print("  declarado: %s — %s" % (local, motivo[:90]))
    print("  (classificacao automatica: definicao/comentario/caminho/identificador/indefinido-declarado)")
