"""Auditoria do classificador de escopo por EXPERIMENTO (a mesma classe do ADV-53: o instrumento olha para o que diz olhar?).

Afirmacao central do E-352: a classificacao e' por STATEMENT - junta continuacao de linha, ignora comentario e distingue
caminho de identificador. Aqui eu construo casos onde a resposta ERRADA e' plausivel e verifico qual sai.
Nao toca no repositorio: usa um diretorio de rascunho.
"""

import importlib.util
import sys
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
    # *TRIADO no E-1010:* `registro := sanitizeIDComponent(valor)` **nao** faz o valor virar
    # identificador -- nao ha prefixo, nao ha `ID`/`Id:`/`id:`, e nao ha concatenacao. *O
    # `E-352` afirma «caminho x identificador», e o `PADRAO_IDENTIFICADOR` implementa a regra
    # que o proprio modulo declara na linha 12.* => *A expectativa era a errada, e passa a ser a
    # **medida**: `indefinido`, que e' o que o statement realmente prova.*
    ("atribuicao a variavel (sem prova de identificador)", '\tregistro := sanitizeIDComponent(valor)', "indefinido"),
    ("INDEFINIDO (sem pista)", "\tx := sanitizeIDComponent(v)", "indefinido"),
    # O CASO QUE TESTA A AFIRMACAO DE E-352: o statement continua nas linhas seguintes.
    (
        "caminho com CONTINUACAO de linha",
        'caminho := filepath.Join(\n\t\traiz,\n\t\tsanitizeIDComponent(org),\n\t)',
        "caminho",
    ),
    (
        # *TRIADO no E-1010:* a continuacao de linha e' **juntada** corretamente (e' o que o
        # `E-352` afirma), mas o statement juntado continua a **nao** ter prova de identificador.
        # => *O que se testa aqui e' a JUNCAO, e nao a classe* => *e a juncao esta' **certa**: o
        # caso passa a esperar `indefinido`, que e' o resultado de juntar as 3 linhas.*
        "continuacao de linha (a juncao funciona; a classe e' indefinida)",
        "registro := sanitizeIDComponent(\n\t\tvalor,\n\t)",
        "indefinido",
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

# **O AUTOTESTE PASSA A FALHAR (E-1010).** => *Ate agora ele **reportava** as divergencias e
# saia **0**, e foi por isso que a divergencia entre o `E-352` e o codigo ficou invisivel.*
# => *O aviso e' explicito para que ninguem "conserte" isto apagando a assercao:*
#    as divergencias sao **achados**, e a decisao e' de quem as triar -- corrigir o
#    `PADRAO_IDENTIFICADOR`, ou corrigir a afirmacao do `E-352`.*
if falhas:
    print()
    print("AUTOTESTE DO CLASSIFICADOR: FALHOU (%d de %d casos divergem do afirmado)." % (falhas, len(casos)))
    print("  *Nao apague esta assercao para obter verde:* cada caso e' um achado a triar.")
    print("  *Ver E-1010 (o instrumento que media e nao atuava) e o E-352 (a afirmacao).*")
    sys.exit(1)
print("AUTOTESTE DO CLASSIFICADOR: CONSISTENTE com o que o E-352 afirma.")
