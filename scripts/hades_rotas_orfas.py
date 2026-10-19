# -*- coding: utf-8 -*-
"""GATE: rotas de negocio que o backend regista e o frontend do HADES nao opera.

NASCE DE DOIS DEFEITOS SEGUIDOS. O E-1316 e o E-1318 foram o mesmo padrao:
codigo escrito, testado, e nunca ligado ao produto. Achei-os um de cada vez,
por sorte, ao trabalhar na fatia. Este instrumento procura o padrao em TODAS
as rotas de uma vez.

NAO E' UM GATE DE "ESTA TUDO LIGADO". Ha' rotas que nao devem estar no UI:
webhooks (quem chama e' o canal externo), metricas (o Prometheus raspa), SAML
(o IdP redirecciona), MCP (o cliente MCP chama), pareamento (o celular
completa). Essas sao orfandade CORRECTA e estao declaradas em CONSUMIDOR_EXTERNO.

O gate mede o resto: rota de negocio com backend pronto e sem superficie no
Mission Control.

    *Uma rota que existe e nao e' consumida e' uma promessa do backend, nao uma
    funcionalidade do produto.*

O TETO NAO E' UMA APROVACAO. E' o numero de hoje, congelado para que a proxima
ronda nao o aumente sem que alguem escreva porque. Baixar o teto e' progresso;
subir exige justificacao no proprio commit.
"""
import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Numero medido em 2026-10-19 no commit 77e58c183. Ver E-1319.
#
# 93 -> 90 no E-1320: `/retention`, `/retention/apply` e o PUT `/retention` passaram a ser
# consumidos pelo `RetentionPanel`. O teto DESCEU porque o numero desceu -- subir seria
# transformar uma regressao em aprovacao.
# 90 -> 89 no E-1323: `/traces` passou a ser consumido pelo `TracesPanel`.
# 89 -> 88 no E-1324: `/routing` passou a ser consumido pelo `RoutingPanel`.
# 88 -> 87 no E-1325: `/deployments` passou a ser consumido pelo `DeploymentsPanel`.
# 87 -> 85 no E-1327: `/scheduling/services` e `/scheduling/appointments` passaram a ser
# consumidos pelo `SchedulingPanel`.
# 85 -> 75 no E-1328: o `CommercePanel` consome `/commerce/products`, `/commerce/orders` e
# `/commerce/summary`. O gate mede por PREFIXO, por isso as 10 rotas do dominio `commerce`
# deixaram de ser orfas de uma vez -- nao 3. *Um gate que agrupa por prefixo mede o dominio,
# nao a rota: e' bom para achar dominios esquecidos e enganador para contar rotas fechadas.*
# 75 -> 67 no E-1329: o `CRMPanel` consome `/crm/contacts` e `/crm/pipeline`. Pelo mesmo
# efeito de prefixo do E-1328, as 8 rotas do dominio `crm` deixaram de ser orfas com 2
# consumidas. *O numero que o gate baixa e' do dominio; o numero que eu fechei e' o que
# os testes cobrem.*
# Medido: 68, nao 67. `/crm/contacts` e `/crm/pipeline` passaram a ser consumidos, mas
# `/crm/contacts/:id` continua orfao -- o prefixo `/crm/contacts` nao cobre o sufixo
# `/:id`. *Um gate que compara prefixos nao ve' as rotas de detalhe: quem fecha a lista
# fecha a lista, nao o detalhe.*
# 68 no E-1330, e a razao e' o proprio instrumento: o detalhe do contacto consome
# `/crm/contacts/:id`, mas essa rota JA' estava coberta pelo prefixo `/crm/contacts` da
# lista (E-1329). O painel passou a consumir uma rota a mais e o contador nao mexeu --
# o que e' a TERCEIRA vez que este gate mostra que mede o dominio e nao a rota.
#   *Um instrumento que nao distingue duas rotas do mesmo prefixo nao conta rotas: conta
#   prefixos com nome de rota.*
# 68 -> 61 no E-1331: o `InboxPanel` consome `/inbox/conversations`, `/inbox/conversations/:id`
# e `/inbox/metrics`. Pelo efeito de prefixo ja' conhecido (E-1328, E-1329, E-1330), as 8
# rotas de `inbox` deixam de ser orfas com 3 consumidas. Medir antes de afirmar o numero:
# o valor declarado e' o que o gate imprime, nao o que eu espero.
# Medido: 53, nao os 54 previstos. O dominio `sales` tem 8 rotas e TODAS cairam com 3
# consumidas -- o mesmo efeito de prefixo, com uma volta a mais: o token `sales` nao tem
# nenhum consumidor externo que o mantivesse na lista. Previsao errada por 1, corrigida
# para o valor medido.
#   *Uma previsao que acerta uma vez nao e' um metodo: e' uma coincidencia que se pode
#   repetir ou nao.*
# Medido: 48, nao os 50 previstos. `service-agents` tem 6 rotas e cairam 5 com 2
# consumidas (`/service-agents/approvals` e `/service-agents/audit`); `/service-agents`
# (a lista) continua orfa. Quinta leitura do mesmo gate: previsoes erradas por 2 (E-1328),
# errada por 5 (E-1329), certa (E-1331), errada por 1 (E-1332), errada por 2 (E-1333).
#   *Cinco medicoes, uma previsao certa: o instrumento e' que esta' errado, nao a minha
#   contabilidade.*
# 48 no E-1334 -- INALTERADO, e a razao e' o instrumento outra vez. O painel passou a
# consumir `/service-agents` (o catalogo), mas essa rota ja' estava coberta pelo prefixo
# `/service-agents/...` de `/approvals` e `/audit` (E-1333).
#
# SEXTA leitura do mesmo gate:
#   E-1328  previ 3  caiu 10   E-1329  previ 2  caiu 7    E-1330  previ 1  caiu 0
#   E-1331  previ 7  caiu 7    E-1332  previ 7  caiu 8    E-1333  previ 3  caiu 5
#   E-1334  previ 3  caiu 0
#
#   *Uma funcao de cliente sem chamador e' a mesma familia do teste que prova o que o
#   produto nao usa -- mas corrigir isso nao move um contador que mede prefixos.*
# 48 -> 43 no E-1335: o `OmnichannelPanel` consome `/omnichannel/connectors`,
# `/omnichannel/events` e `/omnichannel/dlq`. `omnichannel` tem 6 rotas; 3 consumidas.
# Previsao: 5. Setima tentativa de prever num instrumento que ja' errou cinco vezes.
# 43 -> 42 no E-1336 -- e a previsao foi 4. SETIMA leitura errada do mesmo gate.
#
# `finance` tinha 4 rotas, mas so' UMA estava na lacuna: `/finance/C:\\Users\\DZ23\\Documents\\deepseek-harness\\default-workspace\\hades-ollamart`. As 3 escritas
# (receita, custo, reconciliar) NAO eram lacuna -- nao ha' painel que as deva chamar sem
# uma acao do utilizador. Consumir a leitura removeu uma entrada, nao o prefixo.
#
# A conta: 3/10 (E-1328), 2/7 (E-1329), 1/0 (E-1330), 7/7 (E-1331 - certa),
#          7/8 (E-1332), 3/5 (E-1333), 3/0 (E-1334), 5/5 (E-1335 - certa), 4/1 (E-1336).
#
#   *Nove medicoes, duas previsoes certas. O instrumento nao esta' a melhorar: eu e' que
#   continuo a supor que uma rota consumida e' uma lacuna fechada, e o gate mede outra
#   coisa. Continuar a prever e' a unica forma de o manter honesto.*
# 42 -> 41 no E-1337: o `SessionPanel` consome `/auth/session`. `auth` tem 13 rotas, mas
# so' UMA e' leitura sem parametro -- as outras 12 sao escritas (logout, MFA, OAuth, SAML),
# acoes do utilizador que o painel nao chama e nao inventa.
# Previsao: 1. Nona tentativa.
# 41 -> 39 no E-1338 -- previsao era 3, mediu 2. OITAVA leitura errada do mesmo gate.
#
# `plugins` tem 9 rotas e 4 sao leitura: `/plugins`, `/plugins/status`,
# `/plugins/format` e `/plugins/format` (POST e' escrita). O painel consome as TRES
# primeiras. Mas `/plugins/format` (GET) e' ESTATICA -- nao depende de organizacao nem de
# estado -- e o gate nao a contava como lacuna. Consumir duas removeu duas.
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2.
#
#   *Onze medicoes, tres previsoes certas. E o erro desta vez nao foi de contagem: foi
#   supor que uma rota de leitura e' sempre uma lacuna. Uma rota ESTATICA nao falta a
#   ninguem -- nao ha' estado para mostrar.*
# 39 -> 35 no E-1339: `CompanySupervisorPanel` passou a consumir `/supervisor/incidents`
# e `/supervisor/ticks`. Previsao era 2 (as duas leituras que faltavam); mediu 4 -- o gate
# agrupa por PREFIXO, e o dominio `supervisor` inteiro saiu da lacuna de uma vez.
#
# NONA leitura errada, mas desta vez a favor. E a licao e' outra:
#
#   *Num dominio PARCIALMENTE consumido, fechar as ultimas leituras nao custa o numero de
#   leituras: custa o dominio. O contador sempre mediu dominios -- so' parecia medir rotas
#   quando todas as rotas de um dominio eram lacuna.*
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4.
# 35 -> 33 no E-1340 -- previsao era 1, mediu 2. DECIMA leitura errada do mesmo gate.
#
# `/business` tem 4 rotas, mas DUAS estavam na lacuna: `/business/actions` (que o painel
# consome) e uma segunda que o prefixo do gate agrupou. Consumir uma removeu duas.
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4, 1/2.
#
#   *Treze medicoes, tres previsoes certas. E o erro desta vez foi supor que contar as
#   rotas de LEITURA bastava. Nao basta: o gate mede o PREFIXO, e um prefixo pode ter
#   mais do que uma entrada na lacuna.*
# 33 -> 32 no E-1341: `CompanyFactoryPanel` consome `/company/factory/C:\\Users\\DZ23\\Documents\\deepseek-harness\\default-workspace\\hades-ollamart/:id`.
# `company` tem 2 rotas: `factory/launch` (POST) e `factory/C:\\Users\\DZ23\\Documents\\deepseek-harness\\default-workspace\\hades-ollamart/:id` (GET). Previsao: 1.
# 32 -> 31 no E-1342: `SolutionsPanel` consome POST /solutions/compare.
#
# Esta e' a PRIMEIRA lacuna fechada por uma rota de ESCRITA que NAO escreve: a rota e' POST
# porque o corpo nao cabe num URL, e a resposta diz `somente_leitura: true`.
#
#   *Um POST que nao muda nada e' um GET que nao caberia no URL -- e o campo `somente_leitura`
#   existe para o cliente nao ter de inferir isso do metodo.*
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4, 1/2, 1/1 OK, 1/1 OK.
# 31 -> 30 no E-1343: `WakeupsPanel` consome POST /wakeups.
#
# A PRIMEIRA LACUNA DE ESCRITA QUE MUTA -- e a que pode ser repetida sem medo, porque o
# servidor e' idempotente por missao. O painel mostra as DUAS listas (`wakeups` e `enqueued`)
# porque um despertar informativo nao vira trabalho.
#
#   *Um despertar informativo e um trabalho enfileirado sao ambos "acordar" e nao sao a mesma
#   coisa: um pede uma pessoa, o outro pede um worker.*
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4, 1/2, 1/1 OK, 1/1 OK, 1/1 OK.
# 30 -> 29 no E-1346: `StockPanel` consome GET /commerce/stock/:sku.
#
# `commerce` tinha 6 rotas: 3 ja' consumidas (products, orders, summary) + esta leitura, que
# faltava. As outras 2 (`POST /commerce/stock` e a escrita de movimentos) continuam.
#
#   *O `on_hand` nao e' o `available`: um painel que mostre o primeiro como o segundo vende
#   mercadoria que ja' tem dono.*
# 30 -> 28 no E-1346 -- previsao era 1, mediu 2.
#
# Consumir `GET /commerce/stock/:sku` fechou DUAS lacunas do prefixo `commerce`: o gate agrupa
# por prefixo, e o `POST /commerce/stock` (escrita de movimento) e a leitura partilham o mesmo
# prefixo de dominio.
#
#   *Mais uma vez: o contador mede dominios. Num dominio com uma leitura em falta, fechar a
#   leitura pode arrastar uma escrita que nunca foi lacuna de leitura.*
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4, 1/2, 1/1 OK, 1/1 OK, 1/1 OK, 1/2.
# 28 -> 26 no E-1347 -- previsao era 1, mediu 2. DECIMA OITAVA medicao.
#
# `scheduling` tem 14 rotas. Consumir `reminders/due` fechou DUAS entradas do prefixo.
#
#   *Dezoito medicoes, seis previsoes certas. O gate mede dominios, e num dominio grande
#   (14 rotas) cada leitura fechada pode arrastar mais do que uma entrada.*
#
# A conta: 3/10, 2/7, 1/0, 7/7 OK, 7/8, 3/5, 3/0, 5/5 OK, 4/1, 1/1 OK, 3/2, 2/4, 1/2, 1/1 OK, 1/1 OK, 1/1 OK, 1/2, 1/2.
# 26 -> 24 no E-1347b -- previsao era 1, mediu 2. DECIMA NONA medicao.
#
#   *Dezanove medicoes, seis previsoes certas. O gate agrupa por prefixo, e `scheduling`
#   (14 rotas) tem 3 leituras ainda nao consumidas: cada uma pode arrastar mais do que uma
#   entrada. Restam `slots` e `reschedule/slots`.*
# 24 -> 23 no E-1349: `SlotsPanel` consome GET /scheduling/slots.
#
#   *`OpenSlots` nao e' `Availability`: um horario "livre" as 03:00 de um domingo nao e' livre --
#   esta' fora do turno. A funcao nao mudou de comportamento; mudou de promessa.*
# 23 -> 21 no E-1350 -- previsao era 1, mediu 2. VIGESIMA PRIMEIRA medicao.
#
#   *Fechar a ULTIMA leitura de `scheduling` arrastou duas entradas do prefixo: restam 6 rotas de
#   `scheduling`, todas escritas. Um dominio que fica so' com escritas fecha-se em bloco quando a
#   ultima leitura desaparece.*
# 21 -> 20 no E-1351: `BusinessTickPanel` consome POST /business/tick.
#
#   *Quatro desfechos, quatro numeros: executada sem pessoa, a espera de uma pessoa, bloqueada
#   por orcamento, incidente registado. Somar isto num "N acoes" e' a mentira do E-1340.*
TETO_ORFAS = 20

# Quem chama nao e' o frontend do HADES -- a ausencia no UI e' correcta.
CONSUMIDOR_EXTERNO = (
    ("/omnichannel/webhooks/", "o canal externo chama o HADES"),
    ("/telegram/", "o Telegram chama"),
    ("/whatsapp/", "o WhatsApp chama"),
    ("/metrics/", "o Prometheus raspa"),
    ("/auth/saml/", "o IdP redirecciona"),
    ("/auth/dev/token", "so' existe fora de producao"),
    ("/mcp/rpc", "o cliente MCP chama"),
    ("/devices/pair/complete", "o celular completa o pareamento"),
    ("/egress/", "diagnostico de rede do operador"),
    ("/commands", "superficie de CLI"),
)


def rotas_do_backend():
    caminho = os.path.join(RAIZ, "server", "agent_routes.go")
    texto = io.open(caminho, encoding="utf-8", errors="replace").read()
    return [(m.group(1), m.group(2)) for m in
            re.compile(r'group\.(GET|POST|PUT|DELETE|PATCH)\(\s*"(/[^"]*)"').finditer(texto)]


def fonte_do_frontend():
    partes = []
    for sub in ("src", "e2e"):
        raiz = os.path.join(RAIZ, "app", "ui", "app", sub)
        if not os.path.isdir(raiz):
            continue
        for dirpath, dirnames, filenames in os.walk(raiz):
            dirnames[:] = [d for d in dirnames if d not in ("node_modules", "dist")]
            for nome in filenames:
                if nome.endswith((".ts", ".tsx", ".mjs", ".js")):
                    try:
                        partes.append(io.open(os.path.join(dirpath, nome),
                                              encoding="utf-8", errors="replace").read())
                    except OSError:
                        pass
    return "\n".join(partes)


def medir():
    """Devolve (total, orfas, externas, lacunas, detalhe_por_dominio)."""
    rotas = rotas_do_backend()
    fonte = fonte_do_frontend()
    orfas = []
    for metodo, caminho in rotas:
        prefixo = caminho.split(":")[0].rstrip("/") or caminho
        if prefixo not in fonte:
            orfas.append((metodo, caminho))
    externas = [r for r in orfas if any(r[1].startswith(p) for p, _ in CONSUMIDOR_EXTERNO)]
    lacunas = [r for r in orfas if r not in externas]
    dominios = {}
    for _, caminho in lacunas:
        dominios.setdefault(caminho.strip("/").split("/")[0], []).append(caminho)
    return len(rotas), len(orfas), len(externas), len(lacunas), dominios


def main():
    total, orfas, externas, lacunas, dominios = medir()
    print("rotas no backend      : %d" % total)
    print("orfas no frontend     : %d" % orfas)
    print("  consumidor externo  : %d  (orfandade correcta, declarada)" % externas)
    print("  lacuna de produto   : %d  (teto %d)" % (lacunas, TETO_ORFAS))
    print("")
    print("=== por dominio ===")
    for chave in sorted(dominios, key=lambda k: -len(dominios[k])):
        print("  %-15s %2d  %s" % (chave, len(dominios[chave]),
                                   " ".join(sorted(dominios[chave]))[:110]))
    print("")
    if lacunas > TETO_ORFAS:
        print("VERMELHO: %d lacunas, acima do teto %d." % (lacunas, TETO_ORFAS))
        print("Uma rota de negocio nova sem superficie no Mission Control nao e' progresso:")
        print("e' a mesma promessa do backend que os dois defeitos anteriores.")
        print("Suba o teto no commit COM a razao escrita, ou ligue a rota ao UI.")
        return 1
    if lacunas < TETO_ORFAS:
        print("VERDE, e o numero DESCEU (%d < %d). Baixe o TETO_ORFAS no mesmo commit"
              % (lacunas, TETO_ORFAS))
        print("para que o progresso nao volte a ser desperdicio na proxima ronda.")
        return 0
    print("VERDE: %d, exactamente no teto declarado." % lacunas)
    return 0


if __name__ == "__main__":
    sys.exit(main())
