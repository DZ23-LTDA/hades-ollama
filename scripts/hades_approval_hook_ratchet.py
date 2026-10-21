#!/usr/bin/env python3
"""A CATRACA DO ENCAIXE (`E-1044`): vigia o passo de desbloqueio que ficou escrito.

POR QUE ISTO EXISTE.

No `E-1043` o Companion passou a responder `aprovacao:consultar` com
`verificada: false, motivo: "sem_rota_autenticavel_no_runtime"`. O encaixe e' essa constante:
no dia em que o runtime expuser `FindApproved` num canal que o Companion possa autenticar,
muda-se a constante para uma chamada.

*Mas um passo de desbloqueio escrito nao e' um passo de desbloqueio vigiado.* Se a rota
aparecer e ninguem der por isso, o encaixe fica meses a responder `verificada: false` com a
extensao ja' capaz de a usar. **Este checker e' quem da' por isso.**

O QUE ELE MEDE -- o estado do mundo, nao uma promessa:

  1. `FindApproved` existe e continua a ser chamado onde sempre foi (o caminho interno das
     ferramentas, com o `scope` da organizacao);
  2. **NENHUMA rota HTTP chama `FindApproved` directamente** -- se chamar, ha' um caminho novo
     e o encaixe tem de ser revisto;
  3. **NAO existe** handler de approvals que dispense `organizationID(c)` -- um handler assim
     seria um canal sem sessao, que e' exactamente o que o Companion precisa;
  4. o encaixe do Companion continua a ser a constante `MOTIVO_SEM_ROTA`, citada e usada.

FI-- COMO ELE FALHA. Falha ALTO e diz o que fazer: nao para impedir a mudanca, mas para
obrigar quem a faz a ligar o encaixe no mesmo passo. *Uma catraca nao e' um obstaculo: e' um
lembrete que nao se esquece.*

ASCII puro, sem dependencias.
"""
import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SERVIDOR = os.path.join(RAIZ, "server")
AGENTE = os.path.join(RAIZ, "internal", "agent")
COMPANION = os.path.join(RAIZ, "apps", "hades-browser-bridge", "native-host", "companion.mjs")


def definir_raiz(nova):
    """Aponta a catraca a outra raiz. Usado pelo TESTE, que trabalha numa COPIA.

    *Um teste que muta o codigo de producao e' um risco* (`E-1036`): se for interrompido a
    meio, deixa a injecao dentro do ficheiro. Com `--raiz`, a injecao vai para uma copia
    descartavel e o repositorio nunca e' tocado.
    """
    global RAIZ, SERVIDOR, AGENTE, COMPANION
    RAIZ = os.path.abspath(nova)
    SERVIDOR = os.path.join(RAIZ, "server")
    AGENTE = os.path.join(RAIZ, "internal", "agent")
    COMPANION = os.path.join(RAIZ, "apps", "hades-browser-bridge", "native-host", "companion.mjs")


def ler(caminho):
    try:
        return io.open(caminho, encoding="utf-8").read()
    except OSError:
        return ""


def sem_comentarios_go(texto):
    """Remove comentarios Go preservando as LINHAS -- o numero tem de continuar real."""
    texto = re.sub(r"/\*.*?\*/", lambda m: "\n" * m.group(0).count("\n"), texto, flags=re.S)
    return "\n".join(re.sub(r"//.*$", "", l) for l in texto.split("\n"))


def sem_comentarios_js(texto):
    texto = re.sub(r"/\*.*?\*/", lambda m: "\n" * m.group(0).count("\n"), texto, flags=re.S)
    return "\n".join(re.sub(r"//.*$", "", l) for l in texto.split("\n"))


def ficheiros_go(pasta):
    if not os.path.isdir(pasta):
        return []
    return [
        os.path.join(pasta, n) for n in sorted(os.listdir(pasta))
        if n.endswith(".go") and not n.endswith("_test.go")
    ]


def medir():
    """Devolve (achados, resumo). `achados` vazio = o encaixe continua por ligar, como deve."""
    achados = []
    resumo = {}

    # 1. a definicao de `FindApproved` tem de existir
    definicao = os.path.join(AGENTE, "service_agent_approvals.go")
    texto = sem_comentarios_go(ler(definicao))
    tem_def = re.search(r"func \(s \*ServiceAgentApprovalStore\) FindApproved\(", texto)
    resumo["definicao_findapproved"] = bool(tem_def)

    # 2. quem chama `FindApproved`, e onde
    chamadores = []
    for caminho in ficheiros_go(AGENTE) + ficheiros_go(SERVIDOR):
        rel = os.path.relpath(caminho, RAIZ).replace("\\", "/")
        corpo = sem_comentarios_go(ler(caminho))
        for i, linha in enumerate(corpo.split("\n"), start=1):
            if "FindApproved(" in linha and not linha.strip().startswith("func "):
                chamadores.append((rel, i, linha.strip()))
    resumo["chamadores"] = chamadores

    # 2a. uma chamada em `server/` e' um CAMINHO HTTP NOVO -- e' o sinal que interessa.
    for rel, linha, txt in chamadores:
        if rel.startswith("server/"):
            achados.append(
                "ROTA HTTP CHAMA `FindApproved`: %s:%d -> %s" % (rel, linha, txt[:80])
            )

    # 3. existe algum handler de approvals que NAO leia a SESSAO?
    #
    # Um handler assim seria um canal sem sessao -- que e' exactamente o que o Companion
    # precisa. A primeira versao procurava so' `organizationID(c)` e `agentOrganizationID(c)`
    # e deu QUATRO falsos positivos: `decideScheduleApproval` usa `a.requireContextOrganization(c)`,
    # os de deployment usam `a.builderForRequest(c)` e `a.requireApprovalApprover(c)`.
    #
    # *Um detector que conhece duas formas de uma coisa e' um detector que nao a conhece.*
    # Agora as formas estao DECLARADAS numa lista, e o relatorio diz qual delas o handler usou.
    FORMAS_DE_SESSAO = (
        "organizationID(c)", "agentOrganizationID(c)",
        "requireContextOrganization", "requireApprovalApprover",
        "builderForRequest", "authRequired", "c.Get(\"agent.membership\")",
    )
    for caminho in ficheiros_go(SERVIDOR):
        rel = os.path.relpath(caminho, RAIZ).replace("\\", "/")
        corpo = sem_comentarios_go(ler(caminho))
        linhas = corpo.split("\n")
        for i, linha in enumerate(linhas, start=1):
            m = re.match(r"func \(a \*agentAPI\) (\w*[Aa]pproval\w*)\(c \*gin\.Context\)", linha)
            if not m:
                continue
            if m.group(1) == "requireApprovalApprover":
                # e' o proprio helper de autorizacao, nao um handler
                continue
            # o corpo ate' ao proximo `func`
            fim = i
            while fim < len(linhas) and not linhas[fim].startswith("func "):
                fim += 1
            corpo_do_handler = "\n".join(linhas[i - 1:fim])
            usadas = [f for f in FORMAS_DE_SESSAO if f in corpo_do_handler]
            if not usadas:
                achados.append(
                    "HANDLER DE APPROVALS SEM SESSAO: %s:%d -> %s (nenhuma das %d formas "
                    "declaradas de ler a sessao aparece no corpo)"
                    % (rel, i, m.group(1), len(FORMAS_DE_SESSAO))
                )
            else:
                resumo.setdefault("handlers_com_sessao", []).append((rel, i, m.group(1), usadas[0]))

    # 4. o encaixe do Companion continua a existir e a ser usado
    comp = sem_comentarios_js(ler(COMPANION))
    resumo["encaixe_definido"] = bool(re.search(r'export const MOTIVO_SEM_ROTA = "', comp))
    resumo["encaixe_usado"] = len(re.findall(r"MOTIVO_SEM_ROTA", comp)) >= 2
    if not resumo["definicao_findapproved"]:
        achados.append(
            "`FindApproved` DESAPARECEU -- o encaixe aponta para uma funcao que ja' nao existe; "
            "reveja o contrato do `E-1043`"
        )
    if not resumo["encaixe_definido"]:
        achados.append(
            "O ENCAIXE `MOTIVO_SEM_ROTA` DESAPARECEU do Companion -- *o passo de desbloqueio "
            "deixou de estar escrito, e sem ele ninguem sabe o que falta*"
        )
    if not resumo["encaixe_usado"]:
        achados.append("`MOTIVO_SEM_ROTA` esta' definido mas NAO e' usado na resposta")
    return achados, resumo


def principal():
    achados, resumo = medir()
    print("  ENCAIXE da aprovacao (`E-1043`): `MOTIVO_SEM_ROTA`")
    print("    definicao de FindApproved ....... %s" % ("SIM" if resumo["definicao_findapproved"] else "NAO"))
    print("    encaixe definido no Companion ... %s" % ("SIM" if resumo["encaixe_definido"] else "NAO"))
    print("    encaixe usado na resposta ........ %s" % ("SIM" if resumo.get("encaixe_usado") else "NAO"))
    print("    chamadores de FindApproved ....... %d" % len(resumo.get("chamadores", [])))
    for rel, linha, txt in resumo.get("chamadores", []):
        print("      %-42s :%-5d %s" % (rel, linha, txt[:60]))
    # *O relatorio diz QUAL forma de sessao o handler usou* -- assim um falso positivo futuro
    # e' visivel a olho, em vez de escondido atras de um "0 achados".
    hs = resumo.get("handlers_com_sessao", [])
    print("    handlers de approvals com sessao... %d" % len(hs))
    for rel, linha, nome, forma in hs:
        print("      %-42s :%-5d %-28s via %s" % (rel, linha, nome, forma))
    if achados:
        print("")
        print("  *** O ENCAIXE ACORDOU -- LIGUE-O ***")
        for a in achados:
            print("    - %s" % a)
        print("")
        print("  *Isto nao e' um erro de codigo: e' um aviso de que o mundo mudou.*")
        print("  O runtime passou a ter um caminho para a aprovacao que o Companion pode usar.")
        print("  Ligue o encaixe: troque `MOTIVO_SEM_ROTA` pela consulta real e eleve a")
        print("  autoridade da extensao de `forma_apenas` para `verificada` -- com teste que")
        print("  discrimine (ver `E-1041`/`E-1042`). *Um encaixe ligado sem prova nao vale mais")
        print("  do que um encaixe por ligar.*")
        print("")
        print("  Se a mudanca NAO serve ao Companion (por exemplo, um handler de sessao),")
        print("  atualize este checker com o motivo escrito -- *nao o apague.*")
        return 1
    print("")
    print("  O ENCAIXE CONTINUA POR LIGAR -- e nada mudou que o devesse acordar.")
    print("  *Nao ha' rota HTTP que chame `FindApproved`, e nenhum handler de approvals")
    print("  dispensa a sessao.* O encaixe esta' no sitio certo, a espera.")
    return 0


if __name__ == "__main__":
    # `--raiz CAMINHO` permite ao teste trabalhar numa copia descartavel.
    if len(sys.argv) >= 3 and sys.argv[1] == "--raiz":
        definir_raiz(sys.argv[2])
    sys.exit(principal())
