# RELATÓRIO FINAL — SAÍDA 2 (BLOQUEIO DECLARADO)

> Formato obrigatório da missão, secção 10. Sem adjetivos. Cada afirmação tem fonte.
> Este documento **não** declara 100%: declara exactamente o que falta e porquê.

---

## 1. SHA final, WIP, commits

SHA final desta medição: **`a99d55722`** (branch `fix/checker-relogio`, PR #438).
WIP: **0** (árvore limpa; `_tmp/` é ignorado pelo git).

Commits desta sessão (ordem cronológica, 1 linha cada):

| commit | o que fez |
| --- | --- |
| `03fe1e977` | migra `BuildPlanPanel` e `AppSidebar`; reaproveita 61 chaves órfãs |
| `87088a566` | `Class A+` deixava de falhar em silêncio quando o i18n moveu um literal |
| `3a353b6b6` | catálogos i18n separados em chunk próprio, sem subir o tecto do bundle |
| `3d6930a20` | `CompanyGrowthPanel` + 8 chaves `agenticClient`; dívida de órfãs a zero |
| `9a19452f6` | catálogo de conectores passa a função (traduções deixam de congelar) |
| `14b17fda6` | prova comportamental da troca de idioma, **com mutação** |
| `42c184eb7` | `GetCPUMem` sem cgo para darwin |
| `c5b096dcf` | plataforma macOS deixa de constar como BLOCKED |
| `d657f1637` | `flowGraph`, `CodexDesktopModels`, `ClaudeDesktopModels` (89 chaves) |
| `e61888719` | **a troca de idioma passa a existir na UI** |
| `69cec0313` | `HealthCenter` + títulos de rota (UI deixa de ficar misturada) |
| `df1775d75` | banner de backend offline |
| `14cac5f7c` | `NewTaskPage` migrada (15 literais) |
| `603f4c4b3` | mensagens de erro do `agenticClient` |
| `af9da8d08` | **correcção** de uma explicação minha errada sobre o instalador |
| `67078164a` | homologação em 1 comando |
| `c64b7ca5c` | busca global no idioma escolhido |
| `a99d55722` | abas de Configurações |

---

## 2. Os 24 requisitos (situação actual)

Tabela canónica: `docs/mission/HADES_FINAL_RELEASE_GATE.md` (fonte gerada por
`scripts/hades_release_gate.py`). Contagem medida: **20 DONE · 4 BLOCKED_BY_EXTERNAL_DEPENDENCY ·
0 NAO_COMPROVADO**. `hades_release_gate.py --check` = **0**.

Os 4 não-DONE, com a causa medida:

| # | requisito | o que falta (medido, não suposto) |
| --- | --- | --- |
| 5 | APIs externas configuradas são testadas e usadas | **uma credencial de provedor usada de facto** (as 18 configuradas só tiveram chamadas de metadados) |
| 8 | Plugins têm instalação e execução verificáveis | host onde a **delegação de cgroup v2** seja permitida: `cpu.max`/`pids.max` devolvem `permission denied` neste WSL2, mesmo com raiz delegada e com cgroup2 próprio em `unshare -Cm`. `setpriv`/`setgroups` **já passam** (`DIRETO_OK`) |
| 21 | A instalação limpa está homologada | **certificado de assinatura de código** + um **perfil Windows sem ambiente de dev** |
| 23 | Os artefactos de distribuição estão disponíveis | artefacto **assinado**; hoje constrói e sai `NotSigned` (`Get-AuthenticodeSignature`) |

---

## 3. Camada B — 12 critérios

| # | critério | comando executado | saída decisiva |
| --- | --- | --- | --- |
| B1 | instalação limpa Windows | `scripts/hades_installer_windows.ps1 -Versao 0.2.0.541` | `Successful compile` · `HadesSetup-688c4b425.exe` 39,21 MB · **NÃO executado** |
| B2 | primeiro arranque sem erro | browser real em `127.0.0.1:8099` | UI carrega, título `Hades`, erro de backend **claro e traduzido**, sem stack vermelha |
| B3 | jornadas núcleo | testes de percurso + `TestAcionarPareIgnoraIdentidadeDoCorpo` | identidade vem da **sessão**, nunca do corpo |
| B4 | pare acionável | `go test ./server -run Pare -count=1 -v` | **12 PASS** · aciona, bloqueia, sobrevive a reinício, rearme registado |
| B5 | multi-OS | cross-compile WSL + CI | darwin amd64/arm64 **exit 0**; linux amd64/arm64 e windows amd64 **exit 0** |
| B6 | artefactos assinados | `Get-AuthenticodeSignature` | **`NotSigned`** — falta certificado |
| B7 | update e rollback | — | **NÃO MEDIDO** (depende de instalação) |
| B8 | desinstalação limpa | — | **NÃO MEDIDO** (o `.iss` declara `[UninstallDelete]`/`[UninstallRun]`, mas não foi executado) |
| B9 | dados e privacidade | varredura de runtime | telemetria noop; sem segredos em texto no repo |
| B10 | documentação de utilizador | `docs/GUIA_RAPIDO_USUARIO.md` | **6** problemas comuns documentados (§5.1–5.6) |
| B11 | acessibilidade e i18n | browser real, `lang=en` | controlo de idioma, persistência entre reloads, casca coerente, busca e abas traduzidas |
| B12 | performance medida | relatório de baseline | números registados em `HADES_BASELINE_REPORT.md` |

---

## 4. Camada C — homologação independente

**Não existe.** Não há nome, data nem SHA de revisor. O autor deste relatório **é** o construtor, e
por isso não pode assinar a homologação.

O que existe para tornar a homologação barata:

```
python -X utf8 scripts/hades_homologacao_verificar.py
```

Corre os 10 geradores e os 6 gates, recolhe `exit` + última linha e escreve
`_tmp/HOMOLOGACAO_<sha>.md`. Medido: `HOMOLOGACAO_VERIFICAR=PASS`, exit 0.

O que **continua** a exigir uma pessoa: **C2** (tentar falsificar a entrega — procurar testes fracos,
gates desligados, promessas sem prova) e **C3** (assinar).

---

## 5. O que NÃO foi provado (lacuna verificável)

1. **Nenhuma instalação foi executada.** O instalador constrói; ao ser executado **inicia e bloqueia**
   (`/HELP` também não termina em 15 s; `/LOG` nunca é criado). **Causa não determinada.**
2. **B7 e B8 não medidos** — dependem de instalação.
3. **Camada C inexistente.**
4. **B11 não é cobertura total**: 1602 chaves e as superfícies testadas no browser estão coerentes, mas
   a migração do resto da aplicação (~1800 ocorrências medidas por AST) continua incompleta. O que está
   provado é o que foi exercitado, não «toda a UI».
5. **#8**: a suite de plugins neste host dá **27 PASS, 2 FAIL** no cenário com raiz de cgroup
   delegada; os 2 falham por `permission denied` ao escrever `cpu.max`/`pids.max`.
6. **Bateria local completa (81 passos)** não corre neste host: o antivírus remove `go.exe` 1.26.10
   (detecção `2147738096` em `_tmp\goroot12610\go\bin\go.exe`).

---

## 6. Erros cometidos e como foram corrigidos

| erro | correcção |
| --- | --- |
| `Class A+` falhava em silêncio (abortava por `set -e` sem output) e derrubava 3 jobs | `87088a566`: helper que procura o texto no componente **ou** no dicionário |
| bundle a 2.044.042 B contra tecto de 2.000.000 | `3a353b6b6`: catálogos em chunk próprio; **tecto não foi subido** |
| `useState(t(...))` fazia o **payload** `supplier` variar com o idioma | revertido para literal; regra passou a estar nos enunciados seguintes |
| commitei com `hades_audit_package.py --check` = **1** | `ee299b442`: manifesto regenerado; ciclo dos 10 geradores passou a obrigatório |
| primeira mutação do teste i18n partiu a sintaxe e não provou nada | refeita; a tentativa falhada ficou registada no commit |
| escrevi «SmartScreen em sessão sem desktop» | `af9da8d08`: **correcção** — há desktop, e o `exit=1` era artefacto do meu `cmd /c` |
| `const { t } = useT()` não entrou por CRLF | `a99d55722`: `tsc` apanhou (`TS2304`), corrigido com edição explícita |

---

## 7. Próximo passo para humano

| # | pedido | quem decide | o que muda |
| --- | --- | --- | --- |
| 1 | homologar a Camada A/B a partir de um clone limpo e assinar | revisor que **não** seja o autor | Camada C existe → veredito pode subir |
| 2 | certificado Authenticode + chave privada | dono do produto / financeiro | B6 e #23 |
| 3 | perfil Windows real (sem ambiente de dev) | quem administra a máquina | B1, B7, B8 |
| 4 | host Linux com cgroup v2 delegável | infra | #8 |
| 5 | `HADES_PARE_AUTORIDADE` (organizações autorizadas) | governação | B4 passa de «mecanismo provado» a «activo» |
| 6 | uma chave de provedor usada de facto | decisão de custo | #5 |
| 7 | excepção de antivírus para `_tmp\goroot12610` | dono da máquina | bateria local 81/81 |

Detalhe accionável de cada um: `docs/mission/PEDIDOS_EXTERNOS.md`.

---

## 8. Limitações declaradas (o que este verde **não** cobre)

- Verde local **não** é CI verde da PR; verde de um job **não** é a PR verde.
- A Camada A em 20/24 **não** é «quase 100%»: 4 requisitos não têm evidência, e faltar um já impede o
  veredito `RELEASE_CANDIDATE`.
- O artefacto Windows é declarado **COMPOSICAO** pelo próprio script (3 de 3 alvos anteriores ao HEAD):
  não é build reproduzível da fonte.
- i18n: 1602 chaves com paridade e 0 órfãs **não** significam UI totalmente traduzida.
- Clone limpo (P2) não foi executado neste host.

---

## 9. Frase de estado

**NÃO finalizado — BLOCKED_EXTERNAL: homologação independente (Camada C); certificado de assinatura de
código; perfil Windows sem ambiente de dev; host Linux com delegação de cgroup v2; `HADES_PARE_AUTORIDADE`;
uma credencial de provedor usada de facto; excepção de antivírus para a bateria local.**
