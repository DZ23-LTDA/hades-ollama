"""Achado do classificador de escopo: a classificacao e' TEXTUAL (lista de palavras + prefixo tipo ID).

Isso erra nas DUAS direcoes. Aqui construo os dois casos plausiveis e meço qual classe sai.
Nao toca no repositorio.
"""

import importlib.util
from pathlib import Path


def carregar(caminho, nome):
    spec = importlib.util.spec_from_file_location(nome, caminho)
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


cls = carregar(Path("scripts") / "hades_scope_classifier.py", "cls")
print("  PALAVRAS_CAMINHO    : %s" % (cls.PALAVRAS_CAMINHO,))
print("  PADRAO_IDENTIFICADOR: %s" % cls.PADRAO_IDENTIFICADOR.pattern)

casos = [
    # FALSO CAMINHO: a palavra aparece por OUTRO motivo (nome de variavel que nada tem a ver com caminho).
    ("palavra 'path' em variavel de dominio",
     '\trastreador := sanitizeIDComponent(caminhoDoEvento) // path: vem do webhook', "?"),
    ("palavra 'Dir' em contexto alheio",
     "\tnome := sanitizeIDComponent(alvo) + diretorioSufixo", "?"),
    # FALSO INDEFINIDO: identificador legitimo escrito sem prefixo tipo ID.
    ("identificador sem prefixo tipo ID",
     "\tregistro := sanitizeIDComponent(valor)", "?"),
    ("identificador com destino 'chave'",
     "\tchave := sanitizeIDComponent(organizacao)", "?"),
    # CAMINHO REAL escrito com campo minusculo (a lista so' tem 'Root' maiusculo).
    ("caminho via campo s.root (minusculo)",
     "\tdestino := filepath.Join(s.root, sanitizeIDComponent(org))", "?"),
    ("caminho montado com os.TempDir()",
     "\tdestino := filepath.Join(os.TempDir(), sanitizeIDComponent(org))", "?"),
]

print()
print("=== o que o classificador DEVOLVE ===")
for nome, trecho, _ in casos:
    print("  %-42s -> %s" % (nome, cls.classificar_uso(trecho)))

print()
print("  Comparacao com o que a LEITURA do codigo diria:")
print("  - 'path' e 'Dir' como nomes de dominio => o uso e' IDENTIFICADOR, mas a lista de palavras diz CAMINHO")
print("  - 'registro := sanitizeIDComponent(valor)' => IDENTIFICADOR legivel, mas sem prefixo ID vira INDEFINIDO")
print("  - 'os.TempDir()' NAO contem nenhuma palavra da lista => um caminho REAL pode virar indefinido")
