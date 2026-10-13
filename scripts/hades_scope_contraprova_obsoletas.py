"""Contraprova ISOLADA de declaracoes_obsoletas (o que ficou pendente em E-383).

Em E-383, ao renomear a declaracao real, quem disparava primeiro era a checagem de 'uso INDEFINIDO sem declaracao',
mascarando a de obsoleta. Aqui mantenho a declaracao REAL e adiciono uma FANTASMA: assim `pendentes` fica vazio e a
checagem de obsoleta e' a unica que pode acusar. Se ela nao acusar, nao protege nada.
"""

import importlib.util
from pathlib import Path


def carregar():
    spec = importlib.util.spec_from_file_location("c", "scripts/hades_scope_classifier.py")
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


cls = carregar()

print("  estado real: declaracoes obsoletas =", cls.declaracoes_obsoletas())

# Mantem a declaracao REAL (agent/media_pipeline.go:358) e acrescenta uma FANTASMA (linha que nao existe).
cls.DECLARACOES_INDEFINIDAS["agent/media_pipeline.go:99999"] = "fantasma para contraprova"
obsoletas = cls.declaracoes_obsoletas()
print("  com uma fantasma acrescentada: declaracoes obsoletas =", obsoletas)

acusou = "agent/media_pipeline.go:99999" in obsoletas
print()
print("  ACUSOU a fantasma? %s  (esperado: True)" % acusou)
print("  e a REAL continua valida (nao acusada)? %s  (esperado: True)"
      % ("agent/media_pipeline.go:358" not in obsoletas))
