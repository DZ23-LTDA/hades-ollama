"""TESTE DO TESTE (ADV-54, triagem b/c): um checker que NAO acusa uma violacao injetada nao protege nada.

Injeta violacoes reais em diretorios de rascunho e verifica se os checkers as detectam.
Nao toca no repositorio.
"""

import importlib.util
import tempfile
from pathlib import Path


def carregar(caminho, nome):
    spec = importlib.util.spec_from_file_location(nome, caminho)
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


scripts = Path("scripts")

# ---------------------------------------------------------------- (c) timeout
print("=== (c) hades_timeout_class_check.py: orcamento em VARIAVEL escapa? ===")
try:
    tcheck = carregar(scripts / "hades_timeout_class_check.py", "hcheck")
    with tempfile.TemporaryDirectory() as tmp:
        raiz = Path(tmp)
        (raiz / "caso_literal.go").write_text(
            "package p\n\nfunc F() {\n\tctx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)\n\tdefer cancel()\n\t_ = ctx\n}\n",
            encoding="utf-8",
        )
        achados_literal = tcheck.auditar(raiz)
        print("  literal inline  -> acusou: %s" % ("SIM" if achados_literal else "NAO"))
        (raiz / "caso_literal.go").unlink()
        (raiz / "caso_variavel.go").write_text(
            "package p\n\nvar orcamento = 150 * time.Millisecond\n\nfunc F() {\n\tctx, cancel := context.WithTimeout(ctx, orcamento)\n\tdefer cancel()\n\t_ = ctx\n}\n",
            encoding="utf-8",
        )
        achados_variavel = tcheck.auditar(raiz)
        print("  orcamento em VARIAVEL -> acusou: %s" % ("SIM" if achados_variavel else "NAO"))
        (raiz / "caso_variavel.go").unlink()
        (raiz / "caso_expressao.go").write_text(
            "package p\n\nfunc F() {\n\tctx, cancel := context.WithTimeout(ctx, 0.15*time.Second)\n\tdefer cancel()\n\t_ = ctx\n}\n",
            encoding="utf-8",
        )
        achados_expressao = tcheck.auditar(raiz)
        print("  decimal 0.15*time.Second -> acusou: %s" % ("SIM" if achados_expressao else "NAO"))
except Exception as erro:  # noqa: BLE001
    print("  nao foi possivel executar: %r" % (erro,))

# ------------------------------------------------------------- (b) tenant scope
print()
print("=== (b) hades_tenant_scope_check.py: guarda mencionada em COMENTARIO passa? ===")
try:
    bcheck = carregar(scripts / "hades_tenant_scope_check.py", "bcheck")
    so_comentario = (
        "package agent\n\n"
        "// Este arquivo monta caminho por organizacao; o chamador chama requireOrganization( org ) antes.\n"
        "// requireOrganization(  <- apenas mencionado em comentario\n"
        "func caminho(org string) string { return root + org + \".json\" }\n"
    )
    print("  arquivo SEM guarda real, COM a mencao em comentario -> aprovado: %s"
          % ("SIM" if bcheck.tem_guarda_reconhecida(so_comentario) else "NAO"))
    sem_nada = "package agent\n\nfunc caminho(org string) string { return root + org + \".json\" }\n"
    print("  arquivo SEM guarda e SEM mencao            -> aprovado: %s"
          % ("SIM" if bcheck.tem_guarda_reconhecida(sem_nada) else "NAO"))
    print("  dispensas declaradas: %s" % sorted(bcheck.DISPENSAS.keys()))
    diretorio = Path("internal/agent")
    arquivos = bcheck.arquivos_de_armazenamento(diretorio)
    print("  arquivos varridos em internal/agent: %d" % len(arquivos))
    sub = sorted(p.name for p in diretorio.rglob("*.go"))
    print("  .go no diretorio e SUBDIRETORIOS: %d  (se maior que os varridos, o glob nao e' recursivo)"
          % len(sub))
except Exception as erro:  # noqa: BLE001
    print("  nao foi possivel executar: %r" % (erro,))
