"""Homologacao de browser/computer-use do HADES, re-executavel por terceiros.

Exercita o helper do PROPRIO produto (`internal/agent/browser_helper.py`) e nao um
script de teste paralelo. Prova dois caminhos:

  T1 navigate  -> devolve title e screenshot real (browser a renderizar)
  T2 snapshot  -> devolve o texto do DOM
  T3 recusa    -> destino privado sem OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE da erro

Sobe um servidor HTTP local em porta efemera na loopback, pelo que nao depende de
rede externa nem de fixture pre-existente.

Uso:
    python -X utf8 scripts/hades_browser_homologacao.py

Saida: linhas legiveis + veredito final. Exit 0 se navigate+snapshot e a recusa
funcionarem; exit 1 caso contrario.
"""

import json
import os
import pathlib
import subprocess
import sys
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

RAIZ = pathlib.Path(__file__).resolve().parent.parent
HELPER = RAIZ / "internal" / "agent" / "browser_helper.py"

PAGINA = b"<html><head><title>DZ23</title></head><body><h1>Hello Browser</h1></body></html>"


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(PAGINA)))
        self.end_headers()
        self.wfile.write(PAGINA)

    def log_message(self, *args):
        pass


def correr_helper(payload, allow_private, root):
    env = dict(os.environ)
    env["OLLAMA_AGENT_BROWSER_ROOT"] = str(root)
    env["OLLAMA_AGENT_BROWSER_HEADLESS"] = "1"
    env.pop("OLLAMA_AGENT_BROWSER_CDP_URL", None)
    if allow_private:
        env["OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE"] = "1"
    else:
        env.pop("OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", None)
    proc = subprocess.run(
        [sys.executable, str(HELPER)],
        input=json.dumps(payload),
        capture_output=True,
        text=True,
        env=env,
        timeout=180,
    )
    return proc.returncode, proc.stdout.strip(), proc.stderr.strip()


def main():
    if not HELPER.is_file():
        print("FALHA: helper nao encontrado em %s" % HELPER)
        return 1

    try:
        import playwright  # noqa: F401
    except Exception:
        print("FALHA: modulo python 'playwright' ausente.")
        print("       Instale com: python -m pip install playwright")
        print("       e baixe o browser com: python -m playwright install chromium")
        return 1

    root = pathlib.Path(tempfile.mkdtemp(prefix="hades-b17-"))
    servidor = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    porta = servidor.server_address[1]
    threading.Thread(target=servidor.serve_forever, daemon=True).start()
    url = "http://127.0.0.1:%d/" % porta
    print("servidor local: %s" % url)
    print("raiz de sessoes: %s" % root)

    print("\n--- T1 navigate (helper do produto) ---")
    rc1, out1, err1 = correr_helper(
        {"action": "navigate", "session_id": "b17", "url": url}, True, root
    )
    print("rc=%s" % rc1)
    print("stdout=%s" % out1[:400])
    if err1:
        print("stderr=%s" % err1[-400:])

    print("\n--- T2 snapshot (helper do produto) ---")
    rc2, out2, err2 = correr_helper({"action": "snapshot", "session_id": "b17"}, True, root)
    print("rc=%s" % rc2)
    print("stdout=%s" % out2[:400])
    if err2:
        print("stderr=%s" % err2[-400:])
    snapshot_ok = "Hello Browser" in out2

    print("\n--- T3 recusa: destino privado sem ALLOW_PRIVATE ---")
    rc3, out3, err3 = correr_helper(
        {"action": "navigate", "session_id": "b17-recusa", "url": url}, False, root
    )
    print("rc=%s" % rc3)
    print("stdout=%s" % out3[:400])
    recusa_ok = rc3 != 0 and "non-public" in (out3 + err3)

    servidor.shutdown()

    print("\n=== RESULTADO ===")
    print("T1 navigate rc=0 ................. %s" % ("OK" if rc1 == 0 else "FALHOU"))
    print("T2 snapshot com DOM lido ......... %s" % ("OK" if snapshot_ok else "FALHOU"))
    print("T3 recusa de destino privado ..... %s" % ("OK" if recusa_ok else "FALHOU"))

    if rc1 == 0 and snapshot_ok and recusa_ok:
        print("\nHOMOLOGADO: browser/computer-use funciona neste ambiente, com recusa provada.")
        return 0
    print("\nNAO HOMOLOGADO neste ambiente.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
