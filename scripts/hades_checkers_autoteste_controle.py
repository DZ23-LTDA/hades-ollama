"""CONTROLE do teste do teste (c): o checker acusa o caso que ele FOI FEITO para pegar?

Antes de acusar o checker de nao detectar, preciso provar que ele detecta o caso positivo. Se nao detectar nem o
positivo, o defeito pode ser do MEU arquivo de teste (nome/extensao), nao do checker. *A licao do ADV-53: verificar
antes de afirmar.*
"""

import importlib.util
import tempfile
from pathlib import Path


def carregar(caminho, nome):
    spec = importlib.util.spec_from_file_location(nome, caminho)
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


tcheck = carregar(Path("scripts") / "hades_timeout_class_check.py", "hcheck")
print("  DISPENSAS declaradas: %s" % sorted(tcheck.DISPENSAS.keys()))

casos = {
    "literal inline em _test.go": (
        "caso_literal_test.go",
        "package p\n\nimport (\n\t\"context\"\n\t\"testing\"\n\t\"time\"\n)\n\n"
        "func TestCurto(t *testing.T) {\n\tctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)\n"
        "\tdefer cancel()\n\t_ = ctx\n}\n",
    ),
    "literal inline em .go comum": (
        "caso_literal.go",
        "package p\n\nimport (\n\t\"context\"\n\t\"time\"\n)\n\n"
        "func F() {\n\tctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)\n"
        "\tdefer cancel()\n\t_ = ctx\n}\n",
    ),
    "variavel em _test.go": (
        "caso_var_test.go",
        "package p\n\nimport (\n\t\"context\"\n\t\"testing\"\n\t\"time\"\n)\n\nvar orcamento = 150 * time.Millisecond\n\n"
        "func TestCurto(t *testing.T) {\n\tctx, cancel := context.WithTimeout(context.Background(), orcamento)\n"
        "\tdefer cancel()\n\t_ = ctx\n}\n",
    ),
}

for nome, (arquivo, conteudo) in casos.items():
    with tempfile.TemporaryDirectory() as tmp:
        raiz = Path(tmp)
        (raiz / arquivo).write_text(conteudo, encoding="utf-8")
        achados = tcheck.auditar(raiz)
        print("  %-32s -> acusou: %s" % (nome, "SIM (%d)" % len(achados) if achados else "NAO"))
