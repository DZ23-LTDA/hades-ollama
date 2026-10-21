# HADES — pedidos externos (o handoff acionável)

> **Para que serve este ficheiro:** um item só é `BLOCKED_EXTERNAL` se estiver **aqui**, com pedido
> exato, destinatário e verificação. «Falta de margem» não é bloqueio; «não tenho o certificado» é.
> Nenhum valor de chave, token ou certificado é escrito neste ficheiro.
>
> SHA de referência: `78d5e3a8f` (branch `fix/checker-relogio`, PR #438) · 2026-10-21.

---

## 1. Chave de provedor usada de facto (requisito #5 do gate)

**O que falta:** provar que uma API externa é **usada**, não apenas autenticada. O requisito diz
«configuradas são testadas **e usadas**».

**Por que é externo:** «usada» implica inferência real, que consome créditos. A autorização dada até
agora foi apenas para chamadas de metadados (custo zero).

**O que já foi feito (não é bloqueio de configuração):** 18 de 24 provedores autenticam com HTTP 200
a partir desta máquina (OpenAI 133 modelos, Anthropic 14, Gemini 50, Mistral 53, NVIDIA 83, Ollama 23,
Fireworks 22, SambaNova 6, Mimo 6, DeepSeek 2, além de OpenRouter/Apify/Brave/Cloudflare/Deepgram/HF/
Pollinations). 5 recusaram (Cerebras, Groq, Novita, Together = 403; ElevenLabs = 400) e 6 não têm
endpoint de metadados público.

**Pedido exato ao humano:** autorizar **duas** inferências mínimas (ex.: `deepseek-chat` e
`gpt-4o-mini`) com um a cinco tokens cada, e o teto de custo aceitável.

**Quem decide/fornece:** o dono das chaves (utilizador).

**Como entregar com segurança:** já estão em variáveis de ambiente de **utilizador** do Windows
(`HKCU\Environment`), fora do repositório. Não colar em chat, issue nem ficheiro do repo. Recomenda-se
**rotação** das chaves que responderam 200, por terem passado por um histórico de conversa.

**O que muda no gate quando chegar:** `#5` passa a `DONE` com evidência do tipo
`POST /v1/chat/completions -> 200, modelo=X, tokens=Y`, sem imprimir a chave.

**Como verificar o desbloqueio:** repetir a sonda `_tmp/providers_probe.py` (só metadados) e executar
a inferência mínima autorizada, colando códigos HTTP — nunca o valor da chave.

---

## 2. Homologação de plugins com instalação real (requisito #8)

**O que falta:** uma sessão que instale um plugin **real** com verificação de assinatura, num ambiente
que não seja o de desenvolvimento.

**Por que é externo:** exige um plugin assinado por chave que não é do autor e uma máquina/ambiente de
homologação. O código já está provado: **30 testes PASS** (instalação com verificação de assinatura,
rejeição de conteúdo adulterado, rollback a restaurar a versão anterior uma única vez, isolamento por
organização, sete caminhos de recusa).

**Pedido exato ao humano:** disponibilizar (a) um bundle de plugin assinado de terceiro e (b) um
ambiente de homologação limpo ou autorização para usar este PC com perfil novo.

**Quem decide/fornece:** utilizador (ambiente) + fornecedor do plugin (bundle assinado).

**Como entregar com segurança:** caminho local do bundle; nunca no git. A chave privada de assinatura
**não** é pedida — só o bundle já assinado.

**O que muda no gate quando chegar:** `#8` -> `DONE` com a transcrição da instalação (aceite), da
rejeição de um bundle adulterado (recusa) e do rollback.

**Como verificar o desbloqueio:** `go test ./internal/agent -run Plugin -count=1` continua a passar, e
a sessão real produz a transcrição acima.

---

## 3. Certificado de assinatura + runners reais de SO (requisitos #21 e #23)

**O que falta:** um certificado de assinatura de código (Authenticode) e execução de builds em runners
Windows, macOS e Linux reais para homologar **instalação limpa**.

**Por que é externo:** o certificado é um recurso comprado/emitido por terceiro e a chave privada não
pode estar neste ambiente. `Inno Setup` não está instalado nesta máquina (`ISCC.exe` ausente), logo o
instalador também não é construído aqui.

**Estado medido do que existe:** release `v0.2.0-rc.2` publicada com `HadesSetup.exe` e SHA-256
documentado — **porém não assinada** e **apenas Windows**.

**Pedido exato ao humano:** (a) certificado de assinatura e instruções de uso em CI; (b) runners macOS
e Linux reais (ou autorização para usar os do GitHub Actions e o visto de homologação).

**Quem decide/fornece:** utilizador (certificado) + infraestrutura de CI.

**Como entregar com segurança:** o certificado entra como *secret* do repositório/CI, nunca em ficheiro
versionado nem em chat. Aqui só se regista o nome do secret.

**O que muda no gate quando chegar:** `#21` -> `DONE` com a transcrição da instalação limpa nos 3 SO;
`#23` -> `DONE` com o artefacto assinado e a verificação de assinatura (`signtool verify` / `codesign
-dvv`) colada.

**Como verificar o desbloqueio:** `powershell scripts\verify-linux-parity.ps1` e o job de instalador
no CI, com a saída de verificação de assinatura.

---

## 4. Autoridade do gatilho do pare (B4 — autoridade, não mecanismo)

**O que falta:** decidir **quem** pode acionar o pare em produção, com que credencial e com que
registo de auditoria.

**Por que é externo:** é uma decisão de governança, não de código. O **mecanismo** está implementado e
provado (`POST /build/pare`, `POST /build/pare/rearmar`, 11 testes PASS, persistência a sobreviver ao
reinício, identidade só da sessão). O que não é meu para decidir é a lista de quem.

**Estado atual, explícito:** sem `HADES_PARE_AUTORIDADE` definida, ambas as rotas respondem
**403** com o motivo (`o gatilho do pare existe, mas a autoridade não está configurada`). Lista vazia
significa **ninguém** — nunca «todos».

**Pedido exato ao humano:** preencher `HADES_PARE_AUTORIDADE` com as organizações autorizadas e
registar a decisão (quem, desde quando, com que processo de auditoria dos acionamentos).

**Quem decide/fornece:** o responsável operacional da plataforma.

**Como entregar com segurança:** variável de ambiente no host de produção; a lista contém
**identificadores**, não segredos.

**O que muda no gate quando chegar:** B4-autoridade -> `DONE`; o pare passa de «observável e
acionável» a «governado».

**Como verificar o desbloqueio:**
`curl -X POST .../build/pare -H "Cookie: <sessão autorizada>" -d '{"motivo":"teste"}'` -> `200` com
`pare.acionado=true`, seguido de `rearmar` e do reinício para provar a durabilidade.

---

## 5. Homologador independente (Camada C)

**O que falta:** uma pessoa (ou agente de contexto limpo) que **não** seja o autor do diff executar as
Camadas A e B a partir de um clone limpo, tentar **falsificar** a entrega e assinar.

**Por que é externo:** por definição. Um agente que assina o próprio trabalho não homologa nada.

**Pedido exato ao humano:** indicar o revisor (nome/identificador) e a data; entregar-lhe o ficheiro
`docs/mission/HOMOLOGACAO_INDEPENDENTE_RUNBOOK.md` e o SHA a homologar.

**Quem decide/fornece:** o dono do produto.

**Como entregar com segurança:** o revisor recebe acesso de leitura ao repositório; nenhuma credencial
deste ambiente lhe é transmitida.

**O que muda no gate quando chegar:** a Camada C fica assinada em
`docs/mission/HOMOLOGACAO_INDEPENDENTE.md` e **só então** se pode escrever
`release_state = COMPLETED_VERIFIED`.

**Como verificar o desbloqueio:** o bloco de assinatura preenchido com nome, data, SHA e veredito, mais
os achados da tentativa de falsificação.

---

## Resumo do que está do lado de dentro e já fechado

| item interno | estado |
| --- | --- |
| GO-2026-6809 (crypto/x509) | **fechado** — `go.mod` 1.26.10; `govulncheck` = `No vulnerabilities found` |
| vermelho de código no CI | **zero** (`fail 0` na tabela da PR) |
| pare **acionável** (mecanismo + durabilidade) | **fechado** — 11 testes PASS |
| identidade nunca vinda do corpo | **fechado** — teste dedicado |
| acessibilidade WCAG desktop+mobile | **fechado** — axe gate PASS |
| primeira execução e jornadas núcleo | **fechado** — E2E PASS |
| comentário do `concurrency` a contradizer a config | **fechado** — corrigido nos 2 workflows, com a medição citada |

---

## ATUALIZAÇÃO 2026-10-21 — o pedido 3 ficou mais ESTREITO (medido, não estimado)

Quando este ficheiro foi escrito, o requisito #23 dizia «artefacto existe mas não assinado, só
Windows» e a bateria local reportava `BLOCKED_BY_EXTERNAL_DEPENDENCY: sem ISCC.exe`. **Medido agora,
o `ISCC.exe` existia** — `%LOCALAPPDATA%\Programs\Inno Setup 6\ISCC.exe` (6.7.3), que é exactamente
o caminho por omissão de `scripts/hades_installer_windows.ps1`. Estava fora do sítio onde a bateria
o procurava, e isso fazia o bloqueio parecer maior do que era.

**O que passou a estar provado do nosso lado:**

```
scripts/hades_installer_windows.ps1 -Versao 0.2.0.541 -Iscc <ISCC>
  Successful compile (44,016 sec)
  HadesSetup-688c4b425.exe  39,21 MB
  SHA256 243C52BF6E875549D6C42E61E8BB8DE5AC8326F451C4E248AB8F8B7BCE00B75D
  versao PE 0.2.0.541 | produto 'Hades'
  Get-AuthenticodeSignature -> Status = NotSigned
```

**Portanto o pedido 3 deixa de ser «faltam ferramentas» e passa a ser duas coisas, ambas mesmo externas:**

| # | O que falta | Por que é externo | Quem decide | O que muda no gate quando chegar |
|---|---|---|---|---|
| 3a | Certificado Authenticode + chave privada para assinar `HadesSetup-*.exe` | É emitido/comprovado por terceiro; a chave privada nunca entra no repositório | Dono do produto / financeiro | `Get-AuthenticodeSignature` passa de `NotSigned` a `Valid`; **B6** e **#23** deixam de estar bloqueados |
| 3b | Um perfil Windows **sem o ambiente de dev** para correr o instalador | Este host tem toolchain, caches e variáveis de dev; **B1** exige explicitamente «máquina/perfil sem o ambiente de dev» | Quem administra a máquina de teste | **B1** passa a ter prova de instalação limpa, não apenas de compilação |

**O que continua NÃO provado, e está escrito no dossiê:** o instalador **não foi executado**. Construir
não é instalar. E o próprio script classificou o artefacto como **COMPOSICAO** (3 de 3 alvos em
`dist\` são de 16–18/10, anteriores ao HEAD), logo **não** é um build reproduzível desta fonte — o que
reforça o pedido 3b: quem testar deve saber que está a testar uma composição declarada como tal.
---

## 4. Fila de runners do GitHub Actions (infraestrutura)

**Estado:** o job `Web and mobile quality` (`.github/workflows/dz23-agentic-quality.yaml`) falhou no SHA
`c4b93e006` com uma causa **medida e não relacionada com código**:

```
failed to load config from .../app/ui/app/vitest.config.ts
Error: Cannot find module '../lightningcss.linux-x64-gnu.node'
##[error]Process completed with exit code 1.
```

**O que já foi descartado (medido):**
- o lockfile **está completo**: `lightningcss-linux-x64-gnu | v=1.32.0 | resolved=true | integrity=true | optional=true | os=linux` (lido com `node`, não com PowerShell);
- o workflow instala com `npm ci --no-audit --no-fund`, **sem** `--omit=optional`.

**Hipótese declarada:** instalação incompleta no runner (cache npm / fetch do binário por plataforma).

**Tentativa de fechar:** `gh run rerun 39618635386 --failed` (job `119291858912`). Ele foi **criado e
iniciado às 14:50:15Z** e permanece `queued` — a fila de runners não o pegou.

**Pedido externo (ou acção de infra):** destravar a fila de runners (ou dar capacidade) para que este job
volte a correr. Se ele **falhar de novo**, o passo seguinte dentro do repositório é forçar os opcionais
(`--include=optional`) **citando no workflow as linhas de falha acima** — nunca desligar o gate.

**Excepção de processo registada:** com o gate vermelho e a fila travada, o trabalho de código retoma
(I1b) com esta excepção **escrita** — para não confundir «bloqueado por infra» com «parado por opção».
O verde continua a ser exigido para marcar qualquer item como DONE.
---

## 5. Homologacao independente (Camada C)

**O que e:** uma pessoa que NAO seja o autor executa o roteiro de homologacao, tenta falsificar a
entrega e assina com nome, data e SHA.

**Como:** `python -X utf8 scripts/hades_homologacao_verificar.py` no checkout, no SHA a homologar.

**Porque bloqueia:** um autor nao se homologa a si proprio. Sem esta assinatura nenhuma afirmacao de
"pronto" e' legitima, por muitos testes verdes que existam.

---

## 6. Instalador assinado (Authenticode)

**O que e:** certificado de assinatura de codigo + chave, e o passo de assinatura no empacotamento.

**Estado medido:** o instalador e' gerado e NAO assinado (o proprio script avisa "nao assinado, nao
publicado"). Sem assinatura o Windows mostra aviso e o SmartScreen pode bloquear.

---

## 7. Runners multi-OS / desinstalacao limpa

**O que e:** execucao real em macOS e Linux (build e testes), e uma bateria de instalacao/desinstalacao
em perfil Windows limpo.

**Estado medido:** a propria UI reporta "5 de 8 alvos bloqueados" (ios, linux, macos, windows,
windows-installer) e 3 incompletos (android, oci, web). A desinstalacao foi medida UMA vez neste host
(exit 0, registo e PATH limpos) -- nao e' bateria.

---

## 8. Instancia real do Evolution (WhatsApp)

**O que e:** URL e instancia de um servidor Evolution real, para provar o QR contra servidor verdadeiro.

**Estado medido:** o caminho do QR esta' provado com um servidor falso (httptest) e testes que garantem
que a api key e o webhook secret NAO sao devolvidos. Contra servidor real, nao.

---

## 9. Decisao de exposicao de rede (OLLAMA_HOST)

**O que e:** decisao escrita do dono sobre expor o motor para outras maquinas da rede.

**Estado:** o motor escuta apenas em 127.0.0.1 (loopback). A recusa por omissao esta' deliberada e
mantem-se: uma API local sem autenticacao exposta na LAN e' um risco que ninguem pediu. Para habilitar,
e' preciso decisao escrita + autenticacao ANTES da exposicao.
---

## 10. E2E de painel da publicacao (BLOCKED por pre-requisito interno)

**O que falta:** uma MISSAO REAL com ARTEFACTO no armazem, para poder clicar em Publicar e abrir o URL.

**Medido agora:** a app esta de pe (janela "Hades", `/api/tags` 200, pids activos) mas a pasta de
configuracao so' tem `dz23-providers.json` -- nenhum artefacto de missao disponivel para publicar.

**Como destravar (executavel):** correr uma missao que produza um artefacto (pela UI ou pela API com o
token da sessao), e entao: abrir o painel Studio, separador de construcao ao vivo, clicar Publicar, e
PROVAR com a resposta HTTP do URL publicado (200 + conteudo), nao com print.

**Porque nao foi feito ja:** produzir o artefacto exige exercitar o pipeline de agente com modelo, que e'
trabalho de outra dimensao e nao cabe numa verificacao rapida; declarar a publicacao como provada sem
este passo seria exactamente a mentira que o contrato proibe.