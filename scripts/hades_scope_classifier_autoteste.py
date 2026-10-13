"""Auditoria do classificador de escopo por EXPERIMENTO (a mesma classe do ADV-53: o instrumento olha para o que diz olhar?).

Afirmacao central do E-352: a classificacao e' por STATEMENT - junta continuacao de linha, ignora comentario e distingue
caminho de identificador. Aqui eu construo casos onde a resposta ERRADA e' plausivel e verifico qual sai.
Nao toca no repositorio: usa um diretorio de rascunho.
"""

import importlib.util
import tempfile
from pathlib import Path


def carregar(caminho, nome):
    spec = importlib.util.spec_from_file_location(nome, caminho)
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


classificador = carregar(Path("scripts") / "hades_scope_classifier.py", "cls")

casos = [
    ("comentario de linha", '// sanitizeIDComponent("") devolve fallback', "comentario"),
    ("comentario apos codigo", '\tvalor := "x" // usa sanitizeIDComponent(v)', "comentario"),
    ("definicao da funcao", "func sanitizeIDComponent(value string) string {", "definicao"),
    ("caminho inline", 'caminho := filepath.Join(raiz, sanitizeIDComponent(org)+".json")', "caminho"),
    ("identificador inline", '\tregistro := sanitizeIDComponent(valor)', "identificador"),
    ("INDEFINIDO (sem pista)", "\tx := sanitizeIDComponent(v)", "indefinido"),
    # O CASO QUE TESTA A AFIRMACAO DE E-352: o statement continua nas linhas seguintes.
    (
        "caminho com CONTINUACAO de linha",
        'caminho := filepath.Join(\n\t\traiz,\n\t\tsanitizeIDComponent(org),\n\t)',
        "caminho",
    ),
    (
        "identificador com CONTINUACAO de linha",
        "registro := sanitizeIDComponent(\n\t\tvalor,\n\t)",
        "identificador",
    ),
]

print("=== classificacao por caso (a coluna 'esperado' vem do que o E-352 AFIRMA) ===")
falhas = 0
for nome, trecho, esperado in casos:
    obtido = classificador.classificar_uso(trecho)
    ok = "OK " if obtido == esperado else "FALHOU"
    if obtido != esperado:
        falhas += 1
    print("  [%s] %-38s esperado=%-14s obtido=%s" % (ok, nome, esperado, obtido))

print()
print("=== varredura em diretorio de rascunho (a contagem reage ao conteudo?) ===")
with tempfile.TemporaryDirectory() as tmp:
    raiz = Path(tmp)
    (raiz / "caminho.go").write_text(
        'package p\n\nfunc a(org string) string {\n\treturn filepath.Join(raiz, sanitizeIDComponent(org)+".json")\n}\n',
        encoding="utf-8",
    )
    (raiz / "ident.go").write_text(
        "package p\n\nfunc b(v string) string {\n\treturn sanitizeIDComponent(v)\n}\n",
        encoding="utf-8",
    )
    (raiz / "ignorado_test.go").write_text(
        'package p\n\nfunc c() string { return sanitizeIDComponent("x") }\n',
        encoding="utf-8",
    )
    usos = classificador.classificar_usos(raiz)
    print("  arquivos classificados: %s" % sorted(usos.keys()))
    print("  caminho.go  -> %s" % {k: len(v) for k, v in usos.get("caminho.go", {}).items()})
    print("  ident.go    -> %s" % {k: len(v) for k, v in usos.get("ident.go", {}).items()})
    print("  _test.go aparece? %s (deve ser NÃO)" % ("ignorado_test.go" in usos))

print()
print("  casos com resultado diferente do afirmado: %d de %d" % (falhas, len(casos)))
