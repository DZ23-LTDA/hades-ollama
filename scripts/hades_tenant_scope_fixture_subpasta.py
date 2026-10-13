"""Fixture do ADV-56/4 (3a auditoria independente): subpasta nao pode ABORTAR a varredura nem ESCAPAR.

Antes: glob nao recursivo (caminho em subpasta escapava) + chave por basename (identificador em subpasta gerava
FileNotFoundError e o except Exception abortava o resto, escondendo a violacao REAL de outro arquivo).
"""

import importlib.util
import tempfile
from pathlib import Path


def carregar(caminho, nome):
    spec = importlib.util.spec_from_file_location(nome, caminho)
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


catraca = carregar(Path("scripts") / "hades_tenant_scope_check.py", "catraca")

with tempfile.TemporaryDirectory() as tmp:
    raiz = Path(tmp)
    # (a) ARMAZENAMENTO em SUBPASTA sem guarda: antes escapava (glob nao recursivo)
    (raiz / "sub").mkdir()
    (raiz / "sub" / "x.go").write_text(
        'package p\n\nfunc (s *Store) caminho(org string) string {\n'
        '\treturn filepath.Join(s.root, sanitizeIDComponent(org)+".json")\n}\n',
        encoding="utf-8",
    )
    # (b) IDENTIFICADOR em SUBPASTA (era o que ABORTAVA) + violacao REAL no topo, depois dele
    (raiz / "a").mkdir()
    (raiz / "a" / "util.go").write_text(
        'package p\n\nfunc g(org string) string {\n\treturn "prod_" + sanitizeIDComponent(org)\n}\n',
        encoding="utf-8",
    )
    (raiz / "z_real.go").write_text(
        'package p\n\nfunc h(org string) string {\n\treturn "brk_" + sanitizeIDComponent(org)\n}\n',
        encoding="utf-8",
    )
    violacoes, pendentes = catraca.auditar(raiz)
    print("  violacoes: %d" % len(violacoes))
    for v in violacoes:
        print("    - %s" % v[:130])
    print("  pendentes: %s" % pendentes)
    print()
    escapa = not any("sub/x.go" in v for v in violacoes)
    abortou = not any("z_real.go" in v for v in violacoes)
    print("  (a) caminho em SUBPASTA escapou?      %s  (esperado: False)" % escapa)
    print("  (b) a violacao REAL foi escondida?    %s  (esperado: False)" % abortou)
