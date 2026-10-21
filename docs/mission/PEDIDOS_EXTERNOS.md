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
