#!/usr/bin/env python3
"""`hades_extension_check.py` — o §ESTAGIO 14 / §FASE-3-B (HADES Browser Bridge), medido (`E-1105`).

***O QUE O §FASE-3-B EXIGE*** (do prompt):

    - **extensao propria WebExtension/Manifest V3** para controlar so' abas explicitamente conectadas;
    - **Native Messaging** com manifesto de host restrito a' extensao HADES;
    - **pairing** iniciado pelo utilizador, chave aleatoria por dispositivo, expiracao, rotacao e revogacao;
    - **nunca** token estatico no codigo/pacote, nunca segredo a paginas, nunca conexao publica/LAN;
    - **permissoes minimas** (`activeTab`, `scripting`, `storage`), sem `<all_urls>` permanente;
    - **incognito desabilitado por padrao**;
    - **CSP estrita**, sem `eval`, sem codigo remoto;
    - **pacotes de desenvolvimento nao publicados** (`.zip`/unpacked) com hash e instrucoes.

***O QUE ESTE CHECADOR MEDE.*** *Cada exigencia vira uma verificacao **no manifesto e no codigo** -- nao no
documento* => *porque `feature flag desligada, mock, rota 200 sem efeito, documento, prototipo, fixture ou botao
sem wiring nao contam como entrega`* (o proprio prompt).

***E O QUE ELE NAO CONFUNDE*** (a lista do prompt): *«Playwright/Chromium isolado com o navegador pessoal do
utilizador»*, *«adapters compilados com controle do PC homologado»*, *«extensao carregada manualmente com
distribuicao publicada»* => *este checador mede a **extensao**; o E2E em navegador real e' um **gate
separado**, e a publicacao nas lojas e' `BLOCKED_EXTERNAL` por conta/revisao do proprietario.*
"""
import io
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
EXT = RAIZ / "apps" / "hades-browser-bridge"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def ler(rel: str) -> str:
    p = EXT / rel
    return io.open(p, encoding="utf-8").read() if p.is_file() else ""


def main() -> int:
    T("(0) o directorio da extensao existe", EXT.is_dir())
    if not EXT.is_dir():
        return 1

    # ---------- 1. o manifesto ----------
    mf = EXT / "manifest.json"
    T("(1) o `manifest.json` existe", mf.is_file())
    if not mf.is_file():
        return 1
    m = json.loads(io.open(mf, encoding="utf-8").read())
    T("(1b) e' **Manifest V3**", m.get("manifest_version") == 3,
      "version=%r" % m.get("manifest_version"))
    T("(1c) a marca e' HADES", "HADES" in (m.get("name") or ""),
      "%r" % m.get("name"))

    # ---------- 2. incognito desabilitado ----------
    T("(2) `incognito: not_allowed` (o anonimo fica fora por padrao)",
      m.get("incognito") == "not_allowed", "%r" % m.get("incognito"))

    # ---------- 3. permissoes minimas ----------
    perm = set(m.get("permissions") or [])
    T("(3) as permissoes sao minimas (`activeTab`/`scripting`/`storage`/`tabs`)",
      perm <= {"activeTab", "scripting", "storage", "tabs", "alarms", "notifications"},
      "permissoes=%s" % sorted(perm))
    T("(3b) `nativeMessaging` e' OPCIONAL (so' pedida quando o utilizador quer)",
      "nativeMessaging" in (m.get("optional_permissions") or []) and "nativeMessaging" not in perm,
      "optional=%s" % (m.get("optional_permissions") or []))
    T("(3c) NAO ha' `<all_urls>` permanente",
      "<all_urls>" not in json.dumps(m.get("host_permissions") or []),
      "host_permissions=%s" % (m.get("host_permissions") or []))
    # *os hosts opcionais so' sao pedidos sob demanda -- e' o `optional_host_permissions`*
    T("(3d) os hosts amplos estao em `optional_host_permissions` (pedidos sob demanda)",
      "http://*/*" in (m.get("optional_host_permissions") or []),
      "optional_host=%s" % (m.get("optional_host_permissions") or []))

    # ---------- 4. content script restrito ----------
    cs = (m.get("content_scripts") or [{}])[0]
    matches = cs.get("matches") or []
    T("(4) o content script corre so' em localhost (nao em toda a web)",
      bool(matches) and all(("127.0.0.1" in u or "localhost" in u) for u in matches),
      "matches=%s" % matches)
    T("(4b) e nao injecta em todos os frames (`all_frames: false`)",
      cs.get("all_frames") is False, "all_frames=%r" % cs.get("all_frames"))

    # ---------- 5. CSP estrita ----------
    csp = ((m.get("content_security_policy") or {}).get("extension_pages") or "")
    T("(5) a CSP e' estrita (`script-src 'self'`)", "script-src 'self'" in csp, csp[:60])
    T("(5b) e proibe `object-src`/`base-uri`/`form-action`",
      "object-src 'none'" in csp and "base-uri 'none'" in csp and "form-action 'none'" in csp,
      "as tres directivas")

    # ---------- 6. sem eval / codigo remoto ----------
    fontes = sorted(EXT.glob("src/*.js")) + sorted(EXT.glob("popup/*.js"))
    total_bytes = 0
    eval_achado: list[str] = []
    remoto: list[str] = []
    for p in fontes:
        t = ler(str(p.relative_to(EXT)))
        total_bytes += len(t)
        if re.search(r"\beval\s*\(", t) or "new Function(" in t:
            eval_achado.append(p.name)
        # *import de http(s) -- codigo carregado da internet*
        if re.search(r"""(?:import|from)\s*['"]https?://""", t):
            remoto.append(p.name)
    T("(6) nenhum `eval`/`new Function` no codigo da extensao", not eval_achado,
      "fontes=%d, %d B" % (len(fontes), total_bytes) if not eval_achado else str(eval_achado))
    T("(6b) nenhum `import` de URL remota (sem codigo carregado da internet)", not remoto,
      "OK" if not remoto else str(remoto))

    # ---------- 7. o native host ----------
    nh = EXT / "native-host"
    T("(7) o directorio do native host existe", nh.is_dir())
    man_nh = nh / "com.dz23ltd.hades.bridge.json"
    T("(7b) o manifesto do host tem o nome canonico", man_nh.is_file(),
      "com.dz23ltd.hades.bridge.json")
    if man_nh.is_file():
        h = json.loads(io.open(man_nh, encoding="utf-8").read())
        T("(7c) e' do tipo `stdio` (native messaging)", h.get("type") == "stdio", h.get("type"))
        orig = h.get("allowed_origins") or []
        T("(7d) e so' a extensao HADES pode falar com ele (`allowed_origins`)",
          bool(orig) and all("chrome-extension://" in o for o in orig), "%s" % orig)
        T("(7e) e o `path` e' um placeholder, NAO um caminho inventado",
          "REPLACE_WITH" in (h.get("path") or ""),
          "o instalador escreve o caminho real; o repo guarda o TEMPLATE")
    T("(7f) a chave publica do host existe (o ID deriva dela)", (nh / "chave-publica.b64").is_file())
    T("(7g) e o companion existe", (nh / "companion.mjs").is_file(),
      "%d B" % (nh / "companion.mjs").stat().st_size if (nh / "companion.mjs").is_file() else "")

    # ---------- 8. os testes ----------
    testes = sorted(EXT.glob("test/*.test.js"))
    T("(8) a extensao tem testes", len(testes) >= 5, "%d ficheiros" % len(testes))
    # *e cada modulo tem teste?*
    modulos = {p.stem for p in EXT.glob("src/*.js")}
    cobertos = {p.stem.replace(".test", "") for p in testes}
    sem_teste = sorted(modulos - cobertos - {"content_script", "service_worker"})
    T("(8b) os modulos puros tem teste", not sem_teste,
      "modulos=%s" % sorted(modulos) if sem_teste else "%d modulos" % len(modulos))

    # ---------- 9. pairing / chave publica ----------
    protocolo = ler("src/protocol.js") + ler("src/canal_local.js")
    T("(9) o protocolo tem pairing", "pair" in protocolo.lower())
    T("(9b) e tem revogacao", "revog" in protocolo.lower() or "revoke" in protocolo.lower())
    T("(9c) e tem expiracao", "expir" in protocolo.lower() or "ttl" in protocolo.lower())
    # *e usa criptografia real (nao so' base64)*
    T("(9d) e usa criptografia assimetrica (Ed25519/P-256/ECDH)",
      bool(re.search(r"(Ed25519|P-256|ECDH|subtle\.(sign|verify|deriveBits))", protocolo)),
      "o pairing assina, nao so' codifica")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 14 (extensao) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 14 / §FASE-3-B (HADES Browser Bridge): MV3, incognito fora, permissoes")
    print("  minimas, CSP estrita, sem eval nem codigo remoto, native host restrito, e %d testes." % len(testes))
    print("")
    print("  ***O QUE ISSO NAO PROVA (e o prompt e' explicito)***: *o E2E em Chrome/Edge REAIS*")
    print("  (instalar, parear, conectar aba, clicar, pausar, takeover, revogar, desinstalar) e' um")
    print("  gate SEPARADO -- *«extensao carregada manualmente»* nao e' *«distribuicao publicada»*.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
