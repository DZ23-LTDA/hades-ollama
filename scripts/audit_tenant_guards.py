#!/usr/bin/env python3
"""Auditoria de guarda de tenant nos handlers HTTP do Hades.

Método: todo handler que toca o store (`a.context.`, `a.runtime.`, `CompanyStore()`,
`scopedRuntime(`) precisa de uma guarda explícita de organização/projeto/missão ou de
uma função `require*` antes de servir dado. Este script lista os handlers que tocam o
store SEM guarda reconhecida.

Uso:
    python scripts/audit_tenant_guards.py

Leia o resultado como PISTA, não como veredito. A lista de guardas abaixo foi
calibrada contra o código real: a primeira versão, com seis guardas, acusou **49**
handlers, e a maioria era falso positivo (helpers `require*Admin`,
`companyOrganizationID(c)`, guarda manual `if a.authRequired`). Toda ocorrência
precisa ser lida no código antes de virar achado — o custo de um falso positivo é
gastar confiança, e o de um falso negativo é vazar dado.
"""

import glob
import re
import sys

# Guardas reconhecidas. `require` cobre requireSupervisorAdmin, requireWhatsAppAdmin,
# requireOrganizationAdmin, requirePluginAdmin, requireContextOrganization,
# requireApprovalApprover e futuras variações do mesmo padrão.
GUARDS = (
    "require",
    "ForRequest",
    "OrganizationID(c)",
    "organizationID(c)",
    "missionByID",
    "projectForRequestWithID",
    "scopedRuntime(",
)

# Handlers que tocam o store sem guarda de tenant e estão EXPLICADOS no código:
# diagnóstico, fluxo público autenticado por assinatura/segredo, pareamento de
# dispositivo, superfícies de compatibilidade de mídia em runtime único e backup
# recusado por `if a.authRequired`.
EXPECTED = {
    "health", "metrics", "prometheus", "tools",
    "startDevicePairing", "completeDevicePairing",
    "mediaImage", "mediaVideo", "mediaSpeech", "mediaTranscribe", "mediaVision",
    "downloadBackup",
    "whatsappWebhook", "whatsappWebhookVerify",
}

STORE = ("a.context.", "a.runtime.", "CompanyStore()", "a.store", "scopedRuntime(")
FUNC = re.compile(r"^func \(a \*agentAPI\) (\w+)\(c \*gin\.Context\) \{")


def handler_bodies(path):
    lines = open(path, encoding="utf-8").read().splitlines()
    current, start = None, 0
    for index, line in enumerate(lines):
        match = FUNC.match(line)
        if match:
            if current:
                yield current, start, lines[start:index]
            current, start = match.group(1), index
    if current:
        yield current, start, lines[start:]


def main():
    guarded, candidates = 0, []
    for path in sorted(glob.glob("server/*.go")):
        if path.endswith("_test.go"):
            continue
        for name, start, body in handler_bodies(path):
            text = "\n".join(body)
            if not any(token in text for token in STORE):
                continue
            if any(guard in text for guard in GUARDS):
                guarded += 1
                continue
            candidates.append((path, name, start + 1))

    unexplained = [c for c in candidates if c[1] not in EXPECTED]
    print(f"handlers que tocam o store COM guarda reconhecida = {guarded}")
    print(f"SEM guarda reconhecida = {len(candidates)}")
    for path, name, line in candidates:
        mark = "explicado" if name in EXPECTED else "REVISAR"
        print(f"  [{mark}] {path}:{line} {name}")
    print()
    if unexplained:
        print(f"{len(unexplained)} handler(s) precisam de leitura manual antes de virar achado.")
        return 1
    print("Nenhum candidato inexplicado: todo handler sem guarda reconhecida está documentado.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
