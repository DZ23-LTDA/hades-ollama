# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

> **REMEDIACAO MAIS RECENTE: E-1123** -- *os dois instaladores do `dist/` (um sem proveniencia, e nenhum do candidato actual), e o `SOURCE_SHA: DESCONHECIDO` declarado em `docs/release/PROVENIENCIA-HadesSetup.md`.*


















> **REMEDICAO MAIS RECENTE: `E-1090`** (*ronda 173*). A cabeca do indice e' **`E-1090`**: *a documentacao do Companion (`docs/COMPANION.md`), a catraca de 47 verificacoes e as 4 injectoes que a provam.* => *a §FASE-3-A fica `COMPLETED_VERIFIED` nos 10 criterios; o que falta e' a **jornada completa** (`idle -> /goal -> ... -> success`, backend Go real + Ollama).*

> **CHECKPOINT REMEDIDO nas rodadas 181–222 (evidência mais recente `E-959`; cadeia `E-820`→`E-867`).**
>
> ⚡ **E o §18.3-25 ficou COMPLETO nas partes que faltavam (`E-867`):** **API concorrente MEDIDA** —
> **1 pedido = 370 ms**; **4 concorrentes = 187/178/168/163 ms**, **4 de 4 com sucesso**, parede **730 ms**,
> soma **696 ms** ⇒ **`wall/serial = 1,05×`**. ***«A concorrência do Ollama por omissão é uma **fila bem
> servida**: sem paralelismo real e sem degradação.»*** E a **bateria**: o WMI devolve `71582788` min =
> **`0x7FFFFFFF`** = *«desconhecido»*, `status=2` = ligado à corrente ⇒ ***`NÃO_MEDIDO` com causa***.
> ⚠️ **E a 1.ª tentativa deu um número impossível (`0,05×`)** — *e foi ele que denunciou um erro meu:
> `Start-Job` **não herda** `$body`, logo os 4 pedidos foram feitos **sem corpo**.*
> ***«Um número impossível é mais útil que um plausível: obriga a procurá-lo.»***
>
> 🧑 **E o §18.3-30 (pessoa leiga) foi inventariado (`E-866`):** o **onboarding é o maior componente de UI
> do repositório** — `Onboarding.tsx` **50 828 B** e `Onboarding.test.tsx` **45 437 B** — e há **onboarding
> de empresa** também (`company_onboarding_test.go`, `onboarding_volume_limits_test.go`,
> `onboarding_budget_test.go`, `onboarding_rollback_batch_test.go`). A **escolha do modelo** existe
> (`ModelPicker.tsx`) com erro em pt-BR que **sugere a acção**. **3 testes de onboarding passam** (`3 passed`).
> ⚠️ **Mas «instalar sem terminal» tem receita e não artefacto**: `app\ollama.iss` (**10 393 B**, Inno Setup 6)
> existe e o **Inno Setup não está instalado** ⇒ *é a decisão **(c)** do dono*. **O item fica PARCIAL** —
> *as peças existem e o **teste da pessoa** ainda não foi feito: um componente de 50 KB não substitui isso.*
>
> 🔎 **E a «via barata» do `G3` foi TENTADA (`E-865`) — e falhou, mas com prova melhor:** o `llama-server.exe`
> **existe** no Ollama instalado (`%LOCALAPPDATA%\Programs\Ollama\lib\ollama\`, com 4 DLLs e 43 ficheiros),
> e o Hades procura em **9 caminhos — todos dentro do repositório**. Copiei-o e **o erro MUDOU**: de
> `llama-server binary not found` para **`system_info line not found in output`** ⇒ *o binário corre e
> **não é compatível*** — porque o repositório **tem** `llama/server` e o fork **modificou** o motor.
> ***«Não encontro o ficheiro» e «o ficheiro corre e não serve» são dois diagnósticos diferentes.***
> E o `/api/chat` devolve **404**: é o **fail-closed** correcto — *sem motor, o Hades não finge inferência*.
> *Os 10,3 MB copiados foram **removidos**; `git status` de volta a **34**; o Ollama do utilizador intacto.*
>
> 🔥 **E O `G3` MUDOU DE CATEGORIA — de «por medir» para `BLOCKED_EXTERNAL` com causa e remédio medidos
> (`E-864`):** o Hades **compila** (*65,7 MB em 10,1 s*) e **arranca**, mas `POST /api/chat` devolve
> **404** — e o `stderr` do próprio servidor diz porquê: **`llama-server binary not found`** (*o motor de
> inferência nunca foi construído*) e dá o comando: **`cmake -S llama/server --preset cpu && cmake --build --preset cpu`**.
> *Exige `cmake` + toolchain C++, que o `E-815`/`E-859` provaram **ausentes*** ⇒ *é a decisão **(a)** do dono
> (`mingw-w64`), **agora com uma segunda razão**: não serve só para o `-race`, serve para **construir o motor**.*
> HEAD **`9f7379502`** · branch **`fix/checker-relogio`** · PR **#438** aberto, **NÃO mesclado**
> (`mergeState=CLEAN`, `reviewDecision=[]`, 1 revisor = `LMPrado-DZ23`) · **656** evidências no índice ·
> **ID mais alto `E-863`, próximo livre `864`** · **34** entradas não commitadas · **0** stashes ·
> veredicto **`NOT_RELEASE_READY`**.
>
> ⚠️ **REGRA DE MANUTENÇÃO DESTE DOCUMENTO** (*aprendida em `E-861` e reconfirmada na ronda 218*): o
> `hades_mission_check.py` exige que este ficheiro cite **evidência recente** — o piso vem do índice e
> **sobe a cada ronda**. Logo: **toda a ronda que registe uma evidência nova tem de actualizar este
> bloco na MESMA ronda**, ou o checker fica vermelho por um documento que ninguém tocou.
> *Um documento de retomada desactualiza-se **por omissão**, não por acção.*
>
> **PRÓXIMA AÇÃO (medida):** fechar os **3** itens internos do §18.3 que continuam por medir —
> **`30`** (pessoa leiga) · **`25`** (as partes que faltam: TTFT/tokens, API concorrente, bateria) e
> **`22`** (responsividade + sessão longa); e **fechar a lacuna dos headers**, que agora tem **um sítio**:
> `app/ui/ui.go` (*3 dos 9 existem; `SameSite`, `HttpOnly`, `CSP`, `HSTS`, `X-Content-Type-Options`, `csrf` ausentes*).
>
> **E o TTFT/PERF, agora MEDIDO (`E-862` + `E-863`):** há **9** modelos locais (*não* 0,92 GB — essa
> medição do `E-860` media o **directório**, não o conteúdo) e a **primeira inferência real** desta
> sessão correu: **12,07 s a frio** (*93,7% é `load_duration`*) contra **0,88 s a quente** — o
> `load_duration` caiu de **11 295,5** para **55,2 ms** (*204,6×*) e o **trabalho útil ficou idêntico**
> (*738,7 vs 783,7 ms*). **TTFT 22,36 ms/token**, **decode ~20 tokens/s**. E a **VRAM real é 8188 MiB**
> (`nvidia-smi`), o **dobro** dos 4095 que o WMI reporta — *4095 é o tecto de um `uint32` em MB*.
> ***Próximo passo de perf: API concorrente e bateria — ainda NÃO medidos.***
>
> **Estado dos gates:** `G0` **FALHADO** (*`main` vermelha em `4b3e21854`; o SHA candidato não é
> ancestral de `main`*) · `G1`/`G2` verdes (*7/7 checkers, `go build` exit 0*) · **`G2-race` =
> `NÃO_SUPORTADO`** (*`go: -race requires cgo`*) · `G12` **FALHADO** (*`THIRD_PARTY_NOTICES.md`
> 105 566 B e `HADES_SBOM_GO.csv` 21 349 B **fora do Git**; 33 vulnerabilidades*) ·
> `G10` = **`NOT_TRIGGERED`** (*existem 3 workflows de packaging — `dz23-windows-installer`,
> `dz23-linux-package`, `dz23-macos-build` — e o gatilho é `tags: v*`/`dispatch`*).
>
> **E o §18.5 continua a NÃO se aplicar:** o pacote de auditoria independente **não deve ser criado
> ainda** — faltam itens internos do §18.3.

> *A secção abaixo, medida nas rodadas 169–180, fica preservada como **HISTÓRIA**.*

> **REMEDIDO em `2026-10-15` (evidência `E-778`).** HEAD **`9f7379502`** · branch **`fix/checker-relogio`** ·
> PR **#438** aberto e **NÃO mesclado** (review humano pendente) · **5 workflows `success` no MESMO `head_sha`**,
> incluindo o agregador `pr-gate` · **20** entradas não commitadas e **0** stashes · **7/7** checkers `exit=0`.
> **A REGRA DE NUMERAÇÃO MUDOU:** o maior ID cunhado no projecto é **`E-777`** (em código e commits, **não** no
> índice) ⇒ ***a evidência nova continua do MAIOR ID cunhado, não da cabeça do índice***. E **7** das entradas
> `E-560..E-570` **colidem** com outro objecto já citado em código committado ⇒ **não reutilize IDs abaixo de `E-778`**.
> *A secção abaixo, medida na rodada 101, fica preservada como **HISTÓRIA** — citava `eaad692e0` e 547 evidências.*

> **CHECKPOINT REMEDIDO nas rodadas 169–180 (evidência `E-819`; cadeia `E-816`→`E-819`).**
> HEAD **`9f7379502`** · branch **`fix/checker-relogio`** · PR **#438** aberto, **NÃO mesclado** ·
> **612** evidências no índice · **33** entradas não commitadas · **0** stashes · **7/7** checkers `exit=0` ·
> veredicto **`NOT_RELEASE_READY`**. *A secção de baixo, medida na rodada 101, fica preservada como **HISTÓRIA**.*
>
> **O que mudou na fila desde a rodada 101:**
>
> - **`E-816`** — a camada **Web** foi medida pela **primeira vez**: **719 PASS / 2 FAIL** em 110 ficheiros
>   (`npx vitest run`, 13,55 s). *O `go test ./...` **não** corre vitest ⇒ esta camada nunca tinha sido contada.*
> - **`E-816`** — item **P0-4** do `HADES_PENDING_WORK.md` **FECHADO**: a decisão do gate de onboarding está
>   documentada em `routes/index.tsx:17-20` (*«the local-first shell must remain usable when the optional
>   preferences API is offline»*) e o teste que faltava existe: `index.onboarding-gate.test.tsx`, **3/3 PASS**.
> - **`E-818`** — **DEFEITO DE PRODUTO registado**: o «Tentar de novo» do onboarding **apaga o erro e não grava**.
>   Causa: a guarda anti-repetição (`useSettings.ts:150-152`) bloqueia o retry do utilizador, porque o payload é
>   idêntico. **NÃO corrigido de propósito**: este ponto tem **dois** incidentes de carga medidos
>   (**2 387 req/3 s** na home e **42 651 POST** idênticos no `E-782`).
> - **`E-819`** — o **contrato** da guarda: ela compara **dados** (`key` + `snapshot`), **não origem** ⇒ não pode
>   distinguir o clique de uma re-execução automática. Correção exige **contrato novo**; a alternativa segura é
>   **não limpar o erro antes de gravar** (`routes/onboarding.tsx:70-71`), que mata o silêncio sem tocar na guarda.
> - **`E-811`/`E-812`** — §18.2-A verificado **em uso real** neste host: model store em **`D:\AI\Ollama`**
>   (49 blobs, 7 manifestos), configurável pela UI **com validação e fallback** (`server.go:266-272`).
> - **`E-804`/`E-807`/`E-808`** — o `parser` corrigido (*glob sem correspondência em Windows*), o `manifest` com
>   host de registry (`:`) **registado e não corrigido** (*mudaria o formato do model store ⇒ exige ADR*), e um
>   teste que escrevia no `~/.ollama` **real** do utilizador **corrigido e verificado**.
> - **`E-815`** — este host **não tem toolchain C** nenhuma (9 compiladores e 6 directórios ausentes): o
>   `CGO_ENABLED=0` é derivado pelo próprio Go. *A via recomendada é **mingw-w64 portátil**, que não toca em código.*
>
> **Próxima ação imediata (não mudou):** **revisão humana do PR #438** (`LMPrado-DZ23`) — *não se contorna*.

**Atualizado por:** E-547 (checkpoint remedido — rodada 101) · **Evidência de criação:** E-316
**Regra:** este arquivo diz **o que fazer a seguir**, não o que já foi feito. Estado completo em
`HADES_MASTER_STATE.json`; prova em `HADES_EVIDENCE_INDEX.md`.

> ⚠️ **ESTE ARQUIVO ESTEVE PARADO 62 RODADAS** (*rodada 39 → 101*): dizia **353** evidências quando havia **547**, e a
> «fila imediata» era de uma era em que os **estágios 44–47 nem existiam**. ⇒ *Foi o `E-544` que o mediu: dos cinco
> documentos que a FASE 7 manda actualizar, **três** estavam parados há ~16 h, e este não estava na lista — *mas estava
> parado pela mesma razão*. É a **terceira** vez que um documento desta casa escreve a regra que ele próprio viola
> (*o `HANDOFF` e o `PENDING_WORK` também avisam que «os números envelhecem»*).*

---

## 📌 Checkpoint MEDIDO (rodada 101 — nada aqui é estimativa)

| Medida | Valor medido | Como foi medido |
| --- | --- | --- |
| `main` (remoto) | `4b3e21854` | `git rev-parse --short origin/main` |
| Branch do PR #438 | `eaad692e0` | `git rev-parse --short origin/fix/checker-relogio` |
| Evidências no índice | **547** | contagem de linhas `\| E-` |
| Estágios | **38** (*44–47 entregues*) | `HADES_MASTER_STATE.json` → `stages` |
| Bloqueios declarados | **17** (*17 itens / 17 ids únicos — o `B-15` duplicado foi removido no `E-545`*) | `hades_mission_check.py` |
| Conflitos com a `main` | **nenhum** | `git merge-tree --write-tree` → exit 0 |
| Credenciais expostas | **nenhuma** (*20 ocorrências de padrão `ghp_`, **20/20** em ficheiros de teste, **0** em produção*) | varrimento + classificação (`E-547`-era) |
| Veredito de release | **`NOT_RELEASE_READY`** | checker de missão |

> **Regra de honestidade sobre o SHA**: ele **envelhece a cada merge**. Este checkpoint vale para `eaad692e0`, medido em
> **2026-10-14**; qualquer trabalho posterior precisa **remedir** antes de citar.

---

## Fila imediata — REORDENADA pelas rodadas 40–101

### ✅ Já não está na fila (*entregue e testado*)
| O que era pendente | Onde ficou |
| --- | --- |
| **Estágio 44** — Cowork & Spark | `E-533`: `/spark`, `/cowork`, `/reject` na infra existente (*mesma rota de `/goal`*) |
| **Estágio 45** — Multimodal Studio | `E-529` matriz modalidade×engine · `E-535` a **recusa antes de disco/rede** · documentos PDF/DOCX/XLSX/PPTX já existiam |
| **Estágio 46** — Companion | `E-536` estados derivados de **eventos verificados** · `E-538` a11y ligada ao `prefers-reduced-motion` **existente** |
| **Estágio 47** — Artifact Drive | `E-537` busca + CAS · `E-539` comparar/restaurar **sem apagar história** · `E-542` dedup **dentro do tenant** |
| **FASE 1** — `race (macos)` | **provada**: 11/11 etapas, macOS 10 min 28 s vs ubuntu 6 min 32 s |

### ⏭️ O que resta, por ordem
| # | Ação | Critério de aceite |
| --- | --- | --- |
| 1 | **Revisão humana do PR #438** (*`LMPrado-DZ23`*) — **não** se contorna | aprovação registada; só depois o merge |
| 2 | **Contrato duplicado**: o `server` **repete** o número do `internal/agent` (*35→38 foi o sintoma*) | o `server` **lê** o valor em vez de o repetir — *dívida registada no `E-547`* |
| 3 | **P0-B / T2** harnesses externos (*Claude Code, Codex, Manus*) | `BLOCKED_BY_EXTERNAL_DEPENDENCY` até haver login ⇒ `NAO_MEDIDO` |
| 4 | **P0-A / T1** paridade Manus: pesquisa com fontes, planilha/documento, navegador autorizado | artefatos reais; navegador depende de ambiente autorizado |
| 5 | **P0-C / T3** app full-stack com auth, DB real, preview, E2E, rollback | **staging real** é dependência externa até haver alvo |
| 6 | **P1-E / T8** Creator Studio: MP4 **decodificável**, áudio, legendas, thumbnail | **MP4 real exige FFmpeg** ⇒ `BLOCKED_BY_EXTERNAL_DEPENDENCY` |
| 7 | **P1-F / T9** REA via MCP/skill governada sobre artefato próprio/autorizado | escopo, timeout e evidência; nunca burlar DRM |
| 8 | **P1-H** descoberta: README PT/EN, docs versionadas, sitemap, JSON-LD, acessibilidade | **sem** prometer ranking ou recomendação por IAs |

## Bloqueios que **não** se resolvem por código

`BLOCKED_BY_EXTERNAL_DEPENDENCY` com roteiro em `HADES_V22_IMPLEMENTATION_PLAN.md` §3: Ollama Cloud · CLIs
(Codex/Claude/Gemini) · canais Meta/Telegram · telefonia/SIP · contas de negócio · Apple/EAS · YouTube OAuth ·
assinatura Windows · licenças de ferramentas comerciais de REA · **FFmpeg** (produção real de MP4) · **TikTok**
(conector oficial) · **harnesses de paridade** (Claude Code/Codex/Manus exigem credencial).

## Válvulas de honestidade (lembrete permanente)

- Resultado ausente = **`NAO_MEDIDO`**; provedor indisponível **não** é teste aprovado.
- `existe módulo` ≠ capacidade homologada; estágio com evidência ≠ `COMPLETED_VERIFIED`.
- Nenhuma superioridade sobre Manus/Claude/Codex/Lovable/etc. é alegada **sem métrica independente**.
- O SHA medido envelhece a cada merge: registra-se **quando** e **qual revisão** foi medida.
- Chaves coladas em chat são **COMPROMETIDAS** até revogação comprovada pelo proprietário.

## Lições de processo acumuladas (para não repetir)

1. **Gerar derivados → validar → commitar** (nesta ordem).
2. **Nunca cortar branch de uma `main` sem o PR anterior mesclado** — numeração e contagens quebram.
3. **`git push -u`** em branch nova; **confirmar com `ls-remote`**; **nunca reusar nome de branch já mesclado**.
4. **Contrato testado do repositório ganha da preferência do agente** (ADV-19): revertar e **declarar**.
5. **Patcher genérico sobre código desconhecido é arma carregada**: whitelist + assinatura de uma linha + **zero certo
   por tipo**; o **compilador é o árbitro** das arities.
6. **Nunca `git reset --hard` em branch de trabalho**; reconciliar por `cherry-pick` (**sem force-push**).
7. **Hipótese não é diagnóstico; sonda é** (ADV-24): separar hipóteses por medição, não por plausibilidade.
8. **Instrumento que reconhece um único padrão transforma código certo em dívida** (ADV-21) — e o **desconhecido não
   pode passar em silêncio** (E-352).
9. **Limite que não está escrito não é limite: é surpresa** (E-353).
10. **Não declarar verificado o que não se observou**; e **não relatar artefato que não existe** (o PR precisa ser
    **criado** e a criação **confirmada**).

---

## PROXIMA ACCAO - V3.0 (RONDA 71)

> **Requisito**: o `mission_check` exige que este documento cite evidencia **recente** (>= `E-960`).
> Ele parava no `E-959`. **Isto e' o ESTAGIO 1** (*«actualizar inventario e registar divergencias»*).

### A accao unica, agora
**Fechar a QUINTA instancia do defeito «mede o disco»**: os `DIVIDA_STRING` e os tectos de
comentario do `scripts/hades_ascii_ps1_check.py` medem o **disco**, e o CI mede o **checkout**
(*`refs/pull/N/merge`*). *A cura e' a mesma do `E-997` - ler o commit - e ela ja' esta' escrita no
`rastreados_ps1()`, mas os dois dicionarios de tecto ainda sao comparados contra o resultado.*

*Alternativas, em ordem de custo:* **§ESTAGIO 21** (benchmarks reproduziveis), **§ESTAGIO 10**
(Mission Control unico), **§FASE-6** (instalador/`HadesSetup.exe`, que tambem fecha a divida dos
34 caracteres do `install.ps1`).

### O que acabou de fechar (E-960 .. E-1000)
- **§ESTAGIO 22**: `E-987` (snapshot real), `E-988` (manifesto), `E-989` (catraca), `E-993`/`E-994`/
  `E-996` (as 4 volateis e a inversao da regra), `E-997` (o inventario media o disco).
- **§FASE-5**: `E-990` (inventario da marca: 909 canonicas / 441 legadas) e `E-991` (o gerador com
  8 excecoes declaradas).
- **Instrumentos**: `E-992` (ASCII nos `.ps1`), `E-995` (`E-559` restaurado do Git),
  `E-998` (quinta instancia), `E-999` (`\n` literal), `E-1000` (evidencia sem commit).

### Estado medido
```
HEAD 31be202d6 · branch fix/checker-relogio · main 4b3e21854 · WIP 14 · stashes 0
indice 793 entradas (E-001..E-1000) · bateria 16 OK / 0 FAIL
CI: 6/6 verdes no d1f4e7915 (incluindo `test`, que era o que mais reprovava)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
```

### Bloqueios externos reais (nao mascara-los como internos)
```
race / go test -race   CGO_ENABLED=0 e mingw-w64 ausente -> w64devkit + CGO_ENABLED=1
PACKAGE / INSTALL      nenhum artefacto do candidato    -> FASE-6
revisao humana         reviewDecision vazio no #438     -> o dono designa revisor (nao LMPrado-DZ23)
```

### Divergencias factuais registadas (nao reescrever historia)
1. O indice parava em `E-905` no `ROUND 256` e esta' em `E-1000`; a faixa `E-571..E-777` (**207**
   IDs) continua **ausente e declarada** como `EVIDENCE_LOST_HISTORICAL`.
2. **`E-998` citava codigo nao commitado** - corrigido no `31be202d6` (`E-1000`).
3. O `_tmp/bateria_local.ps1` citado neste documento **nao existe no repositorio** (foi removido):
   a bateria canonica e' `scripts/hades_local_gates.ps1` mais os 4 `--check` do `E-987`..`E-997`.

---

## ACTUALIZACAO V3.0 (RONDA 86) - O QUE MUDOU DESDE O `E-1000`

> **Requisito**: a catraca `hades_handoff_window_check.py` (`E-1008`) mede a **janela** de
> citacao -- *o maior ID citado tem de estar a menos de **12** evidencias da cabeca do indice*.
> Este documento parava no `E-1000`, e a cabeca estava em `E-1008`: **8 de atraso**, medidos.
> *A correccao fecha o `atras` para **0**.*

### As 8 evidencias novas, e o que cada uma fechou
``````
E-1001  os 3 documentos de retomada paravam no E-959 (42 de atraso)   aec62f0eb
E-1002  a divida declarada A MAIS (a 5a instancia ja' estava curada)  aec62f0eb
E-1003  o E-1002 PARTIDO em 2 linhas - e NENHUM checker viu          2e3de0dc2
E-1004  o normalizar so' apanhava METADE das tabelas de contagem      55d69ea5d
E-1005  as 5 strings do FASE-5 + o marcador que mutilava `DZ23`       a10e52cdc
E-1006  o E-1005 partiu o passo 8 -> comparar CONJUNTOS, nao listas   b1f1f0d86
E-1007  a catraca da FORMA do indice (5 regras, fault injection)      dbadd69b6
E-1008  a JANELA de citacao dos documentos de retomada (janela 12)    db1ee1fae
``````

### Estado medido nesta ronda
``````
HEAD db1ee1fae · branch fix/checker-relogio · main 4b3e21854 · WIP 13
indice 801 entradas (E-001..E-1008) · bateria 19 OK / 0 FAIL
CI: b1f1f0d86 e dbadd69b6 sem failure · db1ee1fae a correr com os 3 passos novos
    (inventario, FORMA do indice, JANELA de citacao)
PR #438: OPEN · mergeState=BLOCKED · reviewDecision VAZIO (FALTA REVISAO HUMANA)
``````

### As 6 causas de CI vermelho, todas curadas e provadas
``````
run 38922978710  passo 6  gate_head/gate_branch volateis      -> E-993
run 38924032504  passo 7  source_sha/source_branch            -> E-994
run 38926745194  passo 7  host Windows vs Linux               -> E-996
run 38932562362  passo 8  inventario media o DISCO            -> E-997
run 38939790452  passo 8  normalizar so' via 2 colunas        -> E-1004
run 38941538186  passo 8  o E-1005 partiu-o (tabela aberta)   -> E-1006
``````

### Licoes medidas nesta janela
1. Um checker que conta LINHAS nao ve' uma linha que NAO E' LINHA (`E-1007`).
2. Um risco DECLARADO e' uma medicao a fazer -- e as 2 declaracoes cumpriram-se (`E-1006`).
3. Uma cura pode ter o MESMO defeito que cura (`E-1005`).
4. Sobrestimar tambem e' defeito (`E-1002`).
5. Um limiar que mede «SE HA» nao mede «QUAO RECENTE» (`E-1008`).
6. Um detector de texto nao distingue uma MENCAO de uma OCORRENCIA (`E-1007`, `E-1008`).
7. Uma catraca sem teste e' uma crenca -- o teste do `E-1007` deu 3 defeitos da propria.

### Bloqueios externos reais
``````
race / go test -race   CGO_ENABLED=0, mingw-w64 ausente
pacotes / instalador   FASE-6 nao iniciada
revisao humana         reviewDecision VAZIO no PR #438
``````

### A proxima accao unica (actualizada)

**Empacotar a extensao `.zip` nao publicado, com SHA-256 e instrucoes de instalacao/remocao** --
depois de a extensao ficar carregavel (`E-1021`, 24 testes verdes). O `.zip` e' trabalho interno;
**publicar** na Chrome Web Store / Edge Add-ons exige conta e autorizacao do proprietario.

Ordem para os passos seguintes, sem inventar atalhos:
1. `.zip` da extensao + `SHA256SUMS` + `LEIA-ME` de instalacao e remocao;
2. pairing real (native messaging + `HADES Local Companion`) -- ainda **nao existe**;
3. E2E em Chrome/Edge reais (este host consegue, com uma sessao de browser controlada);
4. `E-1021` fechou o `Browser Bridge`; o `Computer Use` continua `PARTIAL`.

### A proxima accao unica (janela `E-1022`)

**O pacote da extensao esta fechado** (`E-1022`): `.zip` nao assinado de 84,3 KB, SHA-256
`6125f72adc9812ae...`, reproduzivel byte-a-byte, com `SHA256SUMS` e `LEIA-ME` ASCII, e um
verificador (`scripts/hades_browser_bridge_package.py`) que **reprova tres defeitos reais pela
regra propria** -- incluindo o separador invertido que o .NET escreve e o Chrome recusa.

**A proxima accao unica**: **provar o handshake sem navegador** -- escrever o contrato do canal
local (schema, sequencia, expiracao, replay) e um teste de contrato que o valide, porque **o
pairing nao existe**: nao ha native messaging nem `HADES Local Companion`, e o popup mostra
`host: - (native host nao instalado)`. Isto e trabalho interno, sem credencial.

Depois, pela ordem: (2) SBOM do pacote da extensao; (3) E2E em Chrome real (este host consegue,
com sessao de browser controlada); (4) Firefox (`.xpi`) ou `ABSENT` documentado; (5) script e doc
de notarizacao/assinatura (pedido pelo dono, ainda nao escrito).

### A proxima accao unica (janela `E-1023`)

**O contrato do canal local esta fechado e testado** (`E-1023`): 23 assercoes em
`apps/hades-browser-bridge/test/canal_local.test.js` provam que o pairing futuro nao pode
nascer com token estatico, replay, ligacao LAN, instalacao desconhecida, chave expirada ou
mensagem fora de ordem. Dois desses testes apanharam defeitos reais meus: o loopback IPv6
estava recusado (o `URL` do Node devolve `[::1]` com brackets) e o detector de `Math.random`
acusava uma **string de erro** que menciona a funcao para a proibir.

**A proxima accao unica**: **acordo de chaves e transporte** -- o contrato assume que a chave
simetrica ja foi partilhada com seguranca, e isso **nao esta feito**. Escrever a derivacao
(por exemplo HKDF a partir do codigo curto, com o codigo a morrer depois de usado) e o
manifesto do native host com a allowlist de `allowed_origins` restrita a extensao HADES.
Continua a ser trabalho interno, sem credencial.

Depois, pela ordem: (2) DNS rebinding a nivel de socket; (3) SBOM do pacote da extensao;
(4) E2E em Chrome real; (5) Firefox (`.xpi`) ou `ABSENT` documentado.

### A proxima accao unica (janela `E-1024`)

**A chave do canal ja deriva do codigo curto** (`E-1024`): HKDF-SHA256 com `salt` e
`info` do contexto, codigo de uso unico, native host com allowlist restrita e
`nativeMessaging` em `optional_permissions`. 56 assercoes nos 3 ficheiros de teste, e a
discriminacao provou que o teste do contexto reprova o defeito do `E-1023`.

**A proxima accao unica**: **proteger o codigo em TRAFEGO** -- o HKDF assume que o codigo
curto atravessou um canal ja autenticado, e num canal `loopback` sem TLS o codigo de 6
digitos viaja em claro no `localhost`. Definir e testar: (a) o host recusa arrancar sem um
handshake que prova posse da chave derivada, (b) o codigo so vive em memoria no host,
(c) rate-limit por pedido e por instalacao. Tudo interno, sem credencial.

Depois: (2) o executavel do `HADES Local Companion` (stdio, com o manifesto ja escrito);
(3) DNS rebinding a nivel de socket; (4) SBOM do pacote; (5) E2E em Chrome real.

### A proxima accao unica (janela `E-1026`)

**O handshake existe e discrimina** (`E-1025`): desafio de 32 bytes com prazo de 30 s, prova de
posse, taxa de 20 tentativas por instalacao numa janela de 60 s, e o nonce morre so' depois de a
prova conferir. 67 assercoes nos 3 ficheiros de teste.

**A proxima accao unica**: **o TRANSPORTE** -- o handshake prova **identidade**, nao **sigilo**.
O codigo curto continua a atravessar o `localhost` em claro no primeiro salto. Falta escolher e
implementar: (a) native messaging por `stdio` (o manifesto ja' existe como template), com o
Companion a ler/escrever mensagens de 4 bytes de comprimento + JSON; ou (b) `loopback` com TLS e
certificado efemero. Tudo interno, sem credencial.

Antes disso, dois defeitos de higiene que o `E-1026` expos: a taxa vive **em memoria** (reiniciar
o host apaga a contagem) e nao ha rate-limit do lado do SO.

### A proxima accao unica (janela `E-1028`)

**O transporte existe e o protocolo atravessa-o** (`E-1027`): enquadramento de native messaging
com 4 bytes little-endian, erros tipados, corpo parcial que nao e' erro, e um canal em memoria
que leva pairing -> handshake -> envelope -> replay recusado, com o fluxo partido em pedacos de
1 byte. 84 assercoes nos 4 ficheiros.

**A proxima accao unica**: **o processo `HADES Local Companion`** -- o enquadramento esta' pronto e
testado, mas **nunca correu contra `process.stdin`/`stdout` de um processo real**. Fazer o
executavel minimo (Node ou Go) que: le' do `stdin` com `desenquadrar`, responde no `stdout` com
`enquadrar`, usa o `estadoInicial()` do contrato, e implementa **so'** o handshake. Depois um teste
que o **arranca como subprocesso** e fala com ele pelo fio -- sem socket, sem mock. Tudo interno,
sem credencial.

Antes disso, o que o `E-1028` expos: a taxa vive em memoria, e nao ha' rate-limit do lado do SO.

### A proxima accao unica (janela `E-1030`)

**O Companion existe como processo** (`E-1029`): o transporte e o handshake atravessam `stdin`/`stdout`
reais, com o fio partido em bytes e a correlacao por `id` a funcionar. 93 assercoes nos 5 ficheiros.

**A proxima accao unica**: **instalar o host de verdade** -- gerar o manifesto de native messaging com o
ID real da extensao e um instalador/desinstalador reversivel, e provar que o navegador arranca o
Companion. Sem isso, `allowed_origins` continua **template** e nao ha' prova de que o SO/navegador
respeita a fronteira. Tudo interno; a instalacao e' local e reversivel.

Antes disso, o que o `E-1029` expos: a taxa e o estado do pairing vivem **em memoria** -- reiniciar o
processo apaga-os. Persistir com escopo e apagar ao revogar e' o passo seguinte.

### A proxima accao unica (janela `E-1031`)

**O ID da extensao deixou de ser adivinhacao** (`E-1031`): e' derivado da chave publica
versionada (`lbndhojeaplfofjmhdmlbocabojakddb`), o `manifest.json` tem `key`, e o native host
**instala-se e desinstala-se de verdade** no registo do Windows -- medido: `registo SUJO: 0`.

**A proxima accao unica**: **exercitar o host POR UM NAVEGADOR**. Instalei-o, mas **nunca abri
o Chrome**. A prova que falta e' o E2E real: carregar a extensao em `chrome://extensions`,
confirmar que o ID mostrado e' o esperado, e falar com o Companion por native messaging a serio.
Sem isso, o que existe e' um protocolo correcto testado por subprocesso -- e nao um produto
instalado num navegador.

Antes disso: persistir a taxa e o estado do pairing (hoje vivem **em memoria**, `E-1029`).

### A proxima accao unica (janela `E-1032`)

**O protocolo atravessou um navegador a serio** (`E-1032`): o Edge real carrega a extensao,
deriva o **mesmo ID**, e a extensao falou com o Companion por native messaging -- resposta de
ida e volta, com o `id` ecoado. Exit 0.

Pelo caminho cairam **4 defeitos que nenhum teste anterior podia ver**:
- `nativeMessaging` estava declarada mas **ninguem a pedia** (e sem ela `sendNativeMessage`
  **nao e' uma funcao**);
- `chrome.permissions.request` **trava para sempre** dentro do handler (curado com prazo de 8s);
- o gesto **nao sobrevive** ao `sendMessage` -- o pedido tem de viver no popup;
- o Windows **nao executa `.mjs`** -- o `path` tem de apontar para um lancador `.cmd`.

**A proxima accao unica**: **o controle de aba a funcionar ponta a ponta**. Hoje o E2E prova o
**canal**; nao prova **observar o DOM de uma pagina real, clicar, e recusar um campo de senha**.
E' um teste com uma pagina local e a extensao a fazer `browser:navigate` -> `browser:files`
com a concessao por aba, o badge, o takeover e a revogacao. Tudo interno.

### A proxima accao unica (janela `E-1033`)

**O controle de aba funciona ponta a ponta** (`E-1033`): observar o DOM real, escrever
num campo legítimo (**o valor chega ao DOM**), **recusar senha e cartão**, badge `ON`,
revogar. Exit 0.

E caiu uma **contradição interna** que nenhum teste de unidade via: o protocolo tinha o
opt-in de loopback -- e **ninguém o chamava**; e o `content_scripts.matches` cobria
**só** `127.0.0.1`/`localhost`, as origens que o protocolo **sempre** recusava. *A única
origem onde o content script era injectado era a única que o HADES não podia usar.*

**A proxima accao unica**: **o takeover humano**. O `browser:takeover` existe no
protocolo e é o scope que falta exercitar: enquanto o utilizador digita, o agente **não**
disputa o campo; pausar; devolver a sessão; e a revogação por **mudança de origem**
(que tem código e não tem E2E). Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1034`)

**O takeover humano existe e funciona** (`E-1034`): quando o utilizador digita ou clica, o
agente para -- e **o valor do humano não é sobrescrito**. O scope `browser:takeover` estava
declarado e não tinha uma linha de implementação.

Pelo caminho caiu um achado que só o E2E podia ver: **a automação accionava o próprio
travão** -- o `preencher` do agente despacha `input`, e o listener de takeover contava isso
como gesto humano. O agente escrevia **uma vez** e perdia o controlo por sua própria acção.

**A proxima accao unica**: **a revogação por mudança de origem**. Tem código no
`tabs.onUpdated` e **não tem E2E**: navegar de `127.0.0.1` para outra origem deve revogar a
concessão, limpar o badge e recusar a acção seguinte -- e o mesmo para outra **aba** com a
mesma origem (que deve continuar concedida). Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1035`)

**A revogação por mudança de origem funciona** (`E-1035`): a aba que navega para outra
origem perde a concessão, o badge limpa e a acção é recusada -- e a **aba vizinha, na
mesma origem, continua concedida**. Voltar não restaura.

E o E2E expôs o **terceiro** local do mesmo defeito: o `tratarAccao` validava a URL do
alvo **sem o opt-in** -- `curar dois de três sítios deixa o terceiro a recusar o que os
outros aceitam`. O sintoma era incoerência: a mesma origem concedia e depois recusava.

**A proxima accao unica**: **procurar os chamadores, não o sintoma**. Fazer um teste que
**falhe se um novo local validar URL sem o opt-in** -- isto é, uma catraca sobre o
**código**, e não sobre o resultado. O defeito apareceu três vezes; a cura pontual não o
fecha. Tudo interno.

### A proxima accao unica (janela `E-1036`)

**A catraca sobre o código existe** (`E-1036`): todo o chamador de `validarUrl`/
`guardarRota` tem de passar `permitirPrivado`, e ela **discrimina** -- verde com a cura,
vermelha sem ela. O defeito do opt-in apareceu em **três** locais em três rondas, cada vez
com a cura pontual do local que falhou.

⇒ ***uma cura pontual não fecha a classe do defeito.** O gate certo não mede o sintoma:
**mede o código que o produziria.***

**A proxima accao unica**: **o `pushState`**. Ficou registado como dívida no `E-1035` e
**pode ser um buraco real**: um SPA que mude de origem por History API **não** dispara
`tabs.onUpdated` com URL novo, logo a concessão **não** seria revogada. Medir primeiro:
um teste que navegue por `pushState` e verifique se a concessão sobrevive indevidamente.
Tudo interno, na mesma página local.

### A proxima accao unica (janela `E-1037`)

**A dívida declarada no `E-1035` era um não-problema** -- `pushState` **não pode** mudar de
origem: o browser lança `SecurityError`. Mas **ao medi-la encontrei dois defeitos reais**:

1. **`onUpdated` sem URL revogava a concessão.** O browser emite `changeInfo` com **só
   `status`** (sem `url`) a cada `pushState`, `replaceState`, **reload** e até mudança de
   título. O código validava a **string vazia**, falhava, e concluía «nova origem».
   ***A navegação mais comum que existe -- recarregar -- matava a sessão.***
2. **O badge mentia.** Com a concessão viva, o Chrome limpava o badge ao navegar e ele
   ficava vazio. ***Um indicador que não reflecte o estado é pior do que indicador nenhum.***

⇒ ***O valor de medir uma dúvida não está na resposta a essa dúvida: está no que aparece
pelo caminho.***

**A proxima accao unica**: **o BFCache**. Ficou sem cobertura e **pode repor uma página
de outra origem** ao voltar atrás -- se o browser restaurar do cache sem emitir `onUpdated`
com URL, a concessão pode viver sobre conteúdo de outra origem. Medir com dois servidores
e uma navegação real seguida de `goBack`. Tudo interno.

### A proxima accao unica (janela `E-1038`)

**O BFCache não revive a concessão** (`E-1038`), e agora sei **porquê**: o regresso traz
**URL** no `onUpdated` -- `{chaves:["status","url"], ...}` -- logo o handler **avalia-o**
contra a origem concedida. E não há restauração automática: 4 passos de ida-e-volta, a
concessão cai em todos e não revive em nenhum.

⇒ ***Mas o limite é tão importante como o resultado: as marcas da página deram `pageshow`
com `persisted: false` -- o browser **recarregou**, não restaurou do cache. Esta medição
não exercitou o BFCache. Um resultado favorável não é uma cobertura.***

**A proxima accao unica**: **a familia `onUpdated` está fechada; mudar de superficie.**
O que falta e é interno e executável: **os frames e o shadow DOM** -- o contrato exige
suportá-los «com limites», e o `observar` actual só olha o documento principal. Medir
primeiro o que acontece hoje com um `iframe` de mesma origem e com um componente de
shadow DOM: o `observar` vê-os, ignora-os, ou mente sobre eles? Tudo interno.

### A proxima accao unica (janela `E-1039`)

**O `observar` devolvia uma lista parcial como se fosse completa** (`E-1039`). Medido:
uma página com **3** controlos reais devolvia **2 nós** e `total: 2`.
***Um observador que devolve parcial como completo faz o modelo agir sobre uma página
que não é a que existe.*** Não era «não atravessar frames» — era **não dizer**.

Agora **atravessa** shadow roots abertos e iframes de mesma origem (2 → **5** nós) e
**declara** `limites` com contagens, profundidades e o que ficou de fora.

E a minha primeira cura **falhou de forma instrutiva**: contava shadow roots fechados
procurando custom elements com filhos — e devolveu `0` para uma página que **tinha** um.
***Um palpite que falha é pior do que uma declaração de ignorância: dá uma contagem que
parece medida.*** Agora diz `"nao_detectaveis_por_design"`.

**A proxima accao unica**: **downloads e uploads**. É a última capacidade do escopo da
FASE-3-B sem cobertura: o contrato exige «caminhos autorizados e manifestos com
tamanho/hash», e aprovação para download executável. Medir primeiro o que existe hoje:
o `preencher` recusa `type=file`, e **nada** gere downloads. Tudo interno.

### A proxima accao unica (janela `E-1040`)

**O scope `browser:files` não tinha uma linha de implementação** — e a mensagem do
`preencher` prometia um `caminho_autorizado` que não existia. Agora **decide**: um
executável **nunca** fica «permitido» (nem com aprovação), e o **mesmo** `a.txt` inerte
é `permitido` em download e `exige_aprovacao` em upload. ***Sair do disco é sempre um acto.***

**Duas lições que valem mais do que o código:**
1. **O canal não transporta binário.** `sendMessage` serializa em JSON — um `Uint8Array`
   atravessa sem `byteLength`. O conteúdo de um ficheiro **não passa por aqui**.
2. **A lista fixa do `lerEstado` ia morder outra vez** — terceira ocorrência. Curada na
   raiz: agora preserva o que não conhece. *Curar o campo que falhou não cura a lista.*

**A proxima accao unica**: **ligar `podesTransferir` às approvals reais do Mission
Control**. Hoje a aprovação chega por `mensagem.aprovado` — um campo que **quem pede**
controla. *Quem pede não certifica o que pede.* Medir a rota de approvals existente
(`internal/agent/service_agent_approvals.go`) e ligá-la, ou declarar honestamente que a
extensão decide sem autoridade de aprovação.

### A proxima accao unica (janela `E-1041`)

**A aprovação da extensão era um campo que quem pede controla** (`aprovado: true`).
Agora é uma **prova estruturada** — `{aprovacaoId, fingerprint, aprovadoPor}` — e o
`fingerprint` tem de **casar com o pedido**. ***Um campo de permissão que quem pede
controla não é uma permissão.***

E não inventei um sistema: o backend **já tinha** `FindApproved(org, fingerprint, now)`
e `Consume` (uso único). A extensão exige a **mesma forma de prova**.

⇒ ***Mas a fronteira é honesta: a extensão **não verifica** a prova contra o store — não
tem acesso. Uma prova bem formada mas falsa passaria. A forma está implementada e
testada; a autoridade não.***

**A proxima accao unica**: **o canal autenticado extensão ↔ runtime**. É o que falta para
a autoridade existir: o `nativeMessaging` já funciona (`E-1032`) e o Companion já fala o
protocolo. Medir se o Companion pode consultar `FindApproved` e devolver a prova — ou
declarar que sem a rota no runtime a extensão decide **sem autoridade** e assumi-lo no
contrato. Tudo interno.

### A proxima accao unica (janela `E-1042`)

**O canal não existe — e agora a decisão diz o que vale.** A rota de approvals existe
(`serviceAgentApprovals`), mas é **autenticada por sessão** — e o Companion é um processo
local **sem sessão**. ***Dar-lhe a sessão do utilizador seria pior: um processo local com
a sessão do utilizador é exactamente o que não queremos.***

Em vez de inventar a integração, **pus a fronteira dentro do contrato**:
`autoridade: "forma_apenas"` acompanha **todo** o veredicto, e `avisoDeAutoridade` diz
por palavras: *«Aprovação com FORMA válida, mas NÃO verificada … uma prova bem formada
mas inexistente passaria aqui.»*

⇒ ***Um limite que não se lê não é um limite. Um campo que diz `verificada` sem ter sido
verificado é pior do que não ter campo.***

**A proxima accao unica**: **a rota no Companion**. É o passo de desbloqueio exacto: uma
rota **no Companion** (não na extensão) que use o **canal emparelhado**
(`pairing:iniciar` / `handshake:*`, que já existem do `E-1031`) para se autenticar
perante o runtime e consultar `FindApproved`. Só então `verificada` tem quem o possa
afirmar. Tudo interno.

### A proxima accao unica (janela `E-1043`)

**A rota de aprovação existe ponta a ponta — e responde que não pode verificar.**
`aprovacao:consultar` no Companion, `arquivo:consultarAutoridade` na extensão: o caminho
completo extensão → native messaging → Companion. ***O que faltava não era a verificação:
era a pergunta.***

O Companion responde `{ok: true, verificada: false, motivo:
"sem_rota_autenticavel_no_runtime", desbloqueio: "expor FindApproved ..."}`.
⇒ ***Uma rota que responde «não posso» é melhor do que uma rota que não existe: quem
pergunta fica a saber que perguntou ao sítio certo.***

E **oito asserções** provam que ele **nunca** diz `verificada: true` — incluindo quatro
tentativas de o forçar.

**A proxima accao unica**: **a catraca do encaixe**. Hoje o passo de desbloqueio é a
constante `MOTIVO_SEM_ROTA`, **escrita mas não vigiada** — se o runtime expuser a rota,
nada falha. Fazer um teste que **falhe quando `FindApproved` passar a estar acessível**
(por exemplo, quando existir um endpoint loopback de approvals), para que o encaixe não
fique esquecido. Tudo interno.

### A proxima accao unica (janela `E-1044`)

**A catraca do encaixe existe.** No `E-1043` deixei o encaixe (`MOTIVO_SEM_ROTA`) e uma
frase num documento — ***e uma frase num documento não acorda ninguém***. Agora há um
checker que **acorda** se aparecer uma rota HTTP que chame `FindApproved` ou um handler
de approvals sem sessão — o sinal de que o Companion já pode verificar.

E falha **alto, com instrução**: *«troque `MOTIVO_SEM_ROTA` pela consulta real e eleve a
autoridade de `forma_apenas` para `verificada` — com teste que discrimine»*.

⇒ ***Uma catraca não é um obstáculo: é um lembrete que não se esquece.***

**A proxima accao unica**: **a família de extensão está fechada e vigiada; mudar de eixo.**
O que falta e é interno e executável: **os quatro achados P2 de integridade do PR #438**
(`scheduling_library_declared_test.go`, `declared_categories_measured_test.go`,
comentários obsoletos em `types.go`/`runtime.go`/`step_routing.go`, e o
`HADES_ADAPTIVE_COWORK_GAP_MATRIX.md`) — são findings de revisão válidos, não expandem
escopo, e fecham dívida de rastreabilidade. Tudo interno.

### A proxima accao unica (janela `E-1045`)

**Os 12 findings do PR #438 foram medidos um a um contra o HEAD.** Dez estão corrigidos
(#2–#6 no `E-909`, #7 no `E-912`, #9 no `E-910`) ou são **falsos positivos do bot**
(#10 e #11).

⇒ ***#11 — «pipe literal numa célula Markdown» — **não se reproduz**: renderizei as 837
linhas com o `marked` e **todas produzem 5 campos**. E a minha medição dele errou **três
vezes** (767, 303, 768) antes de eu usar um renderizador. ***Não se adivinha um
renderizador: usa-se um.*** Eu ia corrigir 768 linhas correctas.***

**A proxima accao unica**: **o #8 é o único não totalmente revalidado.**
`declared_categories_measured_test.go` foi inspeccionado **por nome de teste**, não pela
alegação contra o **comportamento real** do checker. Medir o que o teste afirma, correr o
checker e confrontar os dois — e corrigir a alegação se divergir. Tudo interno.

### A proxima accao unica (janela `E-1046`)

**O #8 está fechado por medição — e os 12 findings do PR #438 estão RESOLVIDOS.**
O teste afirmava que a catraca acusa «dispensa órfã **e obsoleta**»; o comentário deliberado
do próprio checker (`hades_tenant_scope_check.py:216-219`) diz que a premissa de **obsoleta**
estava **errada** e que a acusação foi **removida de propósito**. Já estava corrigido
(`E-912`) — e a contraprova que faltava foi correr o teste: **`ok … 5.620s`**.

**Balanço**: #1 tratado antes · #2–#9 **corrigidos** · #10 e #11 **falsos positivos do bot**
· #12 **feito por construção**. E **as 14 threads do GitHub continuam abertas** — proibido
resolvê-las sem autorização.

**A proxima accao unica**: **mudar de eixo para o ESTÁGIO 21 (benchmarks)**. É o único dos 10
estágios livres que nunca foi iniciado, e o prompt exige cenários reproduzíveis com hardware,
versão, modelo, contexto, warm/cold, repetições e erro. E há um ponto de partida honesto: o
Ollama local está alinhado em 16k de contexto e há modelos medidos neste host. Definir o
protocolo e correr **três repetições** com o que existe, sem inventar benchmark de concorrente.

### A proxima accao unica (janela `E-1047`)

**O ESTÁGIO 21 não estava por iniciar — estava por medir.** A minha própria nota dizia
«não iniciado»; tinha **13+ instrumentos**, método documentado e artefacto.

**Medido**: matriz T2 real com 2 modelos locais ⇒ ***`PASS` em 599,88 s***. Resultado:
**3/8 verde · 4 trapacearam · 1 vermelho**. E os **veredictos repetiram-se um dia depois**
(o artefacto anterior era de `2026-10-15T22:07Z`; o novo, de `2026-10-16T23:41Z`) — só os
tempos mudaram. ***Reprodutibilidade não é o tempo ser igual: é a conclusão ser igual
quando o tempo não é.***

E o `HARNESS_TRAFICOU` (`harness_benchmark.go:286`) apanhou **quatro** modelos a tentar
alterar o **teste** em vez do **código**.

**A proxima accao unica**: **revalidar a repetição** -- `HADES_T2_REPETICAO.md` (`46+/0`)
é do WIP e traz números que **eu nunca corri**. O `RepetirT2Matrix` existe
(`harness_benchmark_repetition.go`); correr com `HADES_T2_MEDIR_MODELO=1` e comparar o
`46+/0` com o que sai agora. Se divergir, o artefacto está desactualizado e deve ser
regenerado; se coincidir, a repetição ganha a mesma reprodutibilidade que a matriz.

### A proxima accao unica (janela `E-1048`)

**ZERO WIP real por integrar.** As «14 entradas» que reportei durante dezenas de rondas
eram **11 `M` de fim-de-linha** + 2 não rastreados + os 2 T2 que commitei.
`git diff --stat` = **0** e `--ignore-cr-at-eol` = **0**: ***num `git status` um `M` não é
necessariamente uma mudança de conteúdo.***

⇒ ***«Reportar o número certo da coisa errada é uma forma de mentir sem dizer uma palavra
falsa.»*** *Uma contagem não é um inventário.*

**Curado**: `scripts/hades_worktree_inventory.py` classifica em três categorias e **imprime
cada caminho** — passo 31 da bateria.

**A proxima accao unica**: **revalidar a repetição T2** (`46+/0`), cuja corrida está
lançada em background com `qwen2.5:7b` e 3 repetições — o mesmo que o artefacto antigo
(`2026-10-13T13:41:29Z`), para comparação directa. Enquanto corre, **não duplicar** o
trabalho que ela já faz.

### A proxima accao unica (janela `E-1049`)

**A repetição T2 foi revalidada: o `46+/0` está confirmado.** Corrida: `PASS` em
209,78 s. Os veredictos são **idênticos três dias depois** (soma `3/3`; comparação,
caso-limite e fatiamento `0/3` com trapaça nas três) — só as durações mudaram.

⇒ *O artefacto estava desactualizado (`2026-10-13`) e eu tinha-o commitado assim, com a
dívida declarada. **Agora a data é `2026-10-16T23:52:54Z` e o número é meu.***
⇒ ***«Declarar a dívida no commit não a paga; correr o teste paga.»***

**A proxima accao unica**: **o estágio 21 tem uma lacuna que não é medição — é
integração.** O `HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem
medições **sem referenciar** a matriz T2 real e a repetição (que agora são medidas do
mesmo snapshot). Verificar se os documentos canónicos do estágio apontam para os
artefactos reais e corrigir a fonte onde divergirem. Tudo interno.

### A proxima accao unica (janela `E-1050`)

**O vermelho do `pr-gate` está explicado, e não é defeito de código.** No `55327160c`, os
gates `test` e `dz23-agentic-quality` foram **cancelados** (pelo push seguinte) e o
`pr-gate.yaml:91` ficou a **esperar uma substituta** até ao deadline de **30 min**
(`:61`) — ***«Timeout aguardando os gates profundos deste SHA»***.

**A prova são dois SHAs, os mesmos gates:** `55327160c` (23:19:46Z) ⇒ `test` e
`dz23-agentic-quality` **cancelled** ⇒ `pr-gate` **failure**. `4157a459b` (23:28:43Z) ⇒
os mesmos dois **success** ⇒ ***6 runs, 6 success***.

⇒ ***«Um push rápido garante um vermelho que não é um defeito.»*** E o `cda756f82` já
tem os dois cancelados — **vai repetir o padrão**.

**O que NÃO fiz**: aceitar `cancelled` como aprovação no `pr-gate`. O owner proibiu-o por
palavras, e **a gate está certa em reprovar**: não pode certificar um SHA cujos gates
não correram.

**A proxima accao unica**: **não fazer push enquanto as runs do SHA corrente não
terminarem** — e verificar se `ba62baf88` (HEAD, 6 runs em curso) fecha com
`failure=0`, agora que sei o que o mata. Se fechar 6/6, o mecanismo fica provado por
**três** SHAs em vez de dois.

### A proxima accao unica (janela `E-1051`)

**O mecanismo do `pr-gate` está provado com TRÊS SHAs.** `55327160c` (gates cancelados)
⇒ `pr-gate` **failure**; `4157a459b` e `ba62baf88` (gates terminados) ⇒ **6/6 success**.
A previsão do `E-1050` foi escrita **antes** do terceiro SHA fechar — e cumpriu-se.

E o comportamento mudou por causa dela: o commit `67581a993` foi feito **localmente** e só
enviado **depois** do `ba62baf88` fechar 6/6. ***A disciplina de push deixou de ser uma
frase no handoff e passou a ser um procedimento seguido.***

**A proxima accao unica**: **fechar o estágio 21 com os documentos canónicos.** O
`HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem medições **sem
referenciar** a matriz T2 real e a repetição — que agora são medições do mesmo snapshot,
verificadas. Verificar se os documentos canónicos apontam para os artefactos reais e
corrigir a fonte onde divergirem. E, antes de qualquer push, **esperar que as runs
terminem**.

### A proxima accao unica (janela `E-1052`)

**A previsão cumpriu-se.** O `E-1051` escreveu que o `cda756f82` reprovaria pelo mesmo
mecanismo; reprovou (`pr-gate` failure, `test` e `dz23-agentic-quality` cancelled).
E a disciplina de push deu **quatro SHAs 6/6 consecutivos** — sem tocar numa linha da gate.

⇒ ***«Uma previsão escrita antes é a única prova de que a explicação serve.»***
⇒ ***«O problema tinha solução procedural, e a solução foi executada.»***

**A proxima accao unica**: **fechar o estágio 21 com os documentos canónicos.** O
`HADES_BENCHMARK_METHOD.md` e o `HADES_STAGE_21_BENCHMARK.md` descrevem medições **sem
referenciar** a matriz T2 real (`E-1047`) nem a repetição (`E-1049`) — que agora são
medições verificadas do mesmo snapshot. Verificar se os documentos canónicos apontam para
os artefactos reais e corrigir a fonte onde divergirem. **E, antes de qualquer push,
esperar que as runs do SHA remoto terminem com `failure=0`.**

### A proxima accao unica (janela `E-1053`)

**O estágio 21 está ligado aos seus artefactos.** Os três documentos canónicos
descreviam as medições e **nenhum chegava ao resultado** — um auditor que lesse o estágio
não chegava ao `3/8 verde`.

⇒ ***«Um método que não aponta para o artefacto é um método que o auditor não segue.»***
⇒ ***«O trabalho estava feito; a prova não estava ligada.»***

**Curado**: `HADES_STAGE_21_BENCHMARK.md` ganhou a **secção 6** (tabela `artefacto →
comando → evidência`), e os outros dois ganharam a ligação **dentro do bloco de
citação** — a primeira tentativa criou dois blocos partidos em Markdown, e foi corrigida.

**A proxima accao unica**: **fechar a lacuna de rastreabilidade com um teste**, não com
texto. Hoje a ligação é textual: **se alguém renomear `HADES_T2_MATRIX.md`, nada acusa.**
Um checker que verifique que os documentos do estágio 21 citam os artefactos (e que os
artefactos existem) fecha isso — e é o mesmo padrão da catraca do encaixe (`E-1044`).
Tudo interno, e **esperar que as runs terminem antes de qualquer push**.

### A proxima accao unica (janela `E-1054`)

**A ligação deixou de ser texto.** No `E-1053` liguei os documentos do estágio 21 aos
artefactos — com texto — e escrevi a dívida: *«se alguém renomear `HADES_T2_MATRIX.md`,
nada acusa»*. Agora acusa: `hades_stage_artifacts_check.py` (**passos 32 e 33**),
**18 asserções**, **4 injeções** e uma contraprova, numa cópia.

⇒ ***«Ligar com texto não é ligar: é descrever a ligação.»***

**E antes de escrever, medi**: existe `hades_discoverability_check.py` — li-o e corri-o;
protege a proibição de **prometer ranking**, não a ligação. ***Reutilizar primeiro não é
uma regra de estilo: é a diferença entre acrescentar e duplicar.***

**A proxima accao unica**: **estender o mapa `LIGACOES`** — hoje vigia só o estágio 21
(3 documentos, 2 artefactos). A mesma pergunta aplica-se a **outros** estágios que
produzem artefactos: verificar quais documentos canónicos citam artefactos reais e quais
não citam, e declarar cada ligação que falta. Medir primeiro; só depois escrever.

### A proxima accao unica (janela `E-1055`)

**Fui procurar ponteiros quebrados em 131 artefactos e 143 documentos: deu ZERO defeitos
reais e QUATRO erros meus.** Comparava `basename` contra caminhos, o `fullmatch` falhava
em nomes longos, e tratava ficheiros gerados em runtime e itens «A CRIAR» como quebrados.

⇒ ***«Quando uma medição produz muitos achados de uma vez, a primeira hipótese é a
medição.»*** — terceira vez nesta janela (`E-1044`/`#11`).
⇒ ***«Um mapa com falsos positivos é pior do que nenhum mapa, porque ensina a ignorá-lo.»***

**A decisão foi NÃO alargar a lista** com os 10 falsos: o `E-1054` continua com as
mesmas 6 entradas verificadas.

**A proxima accao unica**: **medir a pergunta certa** — *quais documentos **afirmam
produzir** um artefacto e **não o citam?*** Não «quais citam zero» (113 dos 143 são
narrativos e não têm de citar). A pergunta é sobre **afirmação**, não sobre ausência:
procurar nos documentos frases que nomeiam um artefacto como produzido por eles, e
verificar se o ponteiro está lá. Tudo interno, e **esperar as runs antes de qualquer push**.

### A proxima accao unica (janela `E-1056`)

**Fiz a pergunta certa — *quais documentos **afirmam produzir** um artefacto e não o
citam?* — e deu SETE falsos positivos, todos do mesmo erro: o regex lê «está em» e «a
partir de» como se fossem «gera».**

Verificados um a um: `eas.json` **existe**; `_tmp/rel_a.md` é **gitignored**;
`HADES_MASTER_STATE.json` é a **entrada** do gerador, não a saída; e os outros quatro são
**citações**, não produções.

⇒ ***«Um detector léxico não vê a relação: vê os termos.»***
⇒ ***«O resultado de procurar e não encontrar é «não encontrei» — e não «não existe».»***

**As duas perguntas úteis foram feitas e respondidas (zero). A linha de investigação
encerra aqui.** — ***«A diligência sem hipótese vira ruído, e ruído num gate é pior do que
um gate a menos.»***

**A proxima accao unica**: **voltar a construir em vez de auditar.** A janela 142 provou
que o estágio 21 mede (3/8 verde, veredictos reproduzem) e que a ligação vigia. O trabalho
que falta é **de produto**, não de rastreabilidade: os artefactos não assinados do
candidato actual (§FASE-6). O Windows é o único que o toolchain deste host consegue
produzir ponta a ponta — e há `scripts/hades_bundle_windows.ps1` no WIP integrado.
Medir primeiro o que ele faz; depois correr. **E esperar as runs antes de qualquer push.**

### A proxima accao unica (janela `E-1057`)

**O artefacto do candidato existe.** `Hades-6093a6692-windows-amd64-portable-completo.zip`
— **48 234 478 bytes**, verificado pelo próprio script: `/api/tags` 200, `/api/chat` 200
com resposta **real** (`[Tudo certo!]`), `/api/v1/model/upstream` **501** (contrato honesto).

⇒ ***«Um artefacto que declara os seus limites vale mais do que um que os omite.»***

**E o caminho até ele revelou dois achados**: o `TMP` negado (curado **na chamada**, não no
script) e **17 processos `llama-server` órfãos** de dez verificações anteriores — o script
matava o `hades.exe` mas **não o filho**, e o filho segurava os DLLs.
⇒ ***«Um teste que arranca um filho e mata só o pai deixa o filho a segurar o diretório.»***
**Curado e provado**: `runner parado 1 proceso(s)`, e **0 órfãos** depois.

**A proxima accao unica**: **o `HadesSetup.exe` não assinado** — o `build_windows.ps1`
(1369 linhas) e o `app/ollama.iss` (223 linhas) existem, e o `ISCC.exe` **existe e
funciona**. O `LEIA-ME` do bundle já o diz: *«o que falta não é a ferramenta: é EXECUTAR o
instalador»*. **Gerar** o instalador é interno; **instalá-lo** escreve em
`%LOCALAPPDATA%\Programs\Hades` e exige decisão do dono. Medir o que o
`build_windows.ps1` faz antes de o correr. **E esperar as runs antes de qualquer push.**

### A proxima accao unica (janela `E-1058`)

**O `HadesSetup.exe` não assinado do candidato existe** — 24,26 MB, gerado pelo `ISCC.exe`
a partir de binários que **EU** construí da fonte:
`ollama.exe` 46,65 MB e `windows-ollama-app-amd64.exe` 42,66 MB, ambos `exit 0`.

⇒ ***«O bloqueio não é de permissão: é de toolchain.»*** — o `build_windows.ps1` compila o
motor C++ (`cmake`/`ninja`/`cl` **ausentes**), mas **o caminho que exigia a ferramenta não
era o único caminho**: os binários Go constroem-se, e o C++ já está pronto.

**E o compilador denunciou uma dívida**: `[UninstallRun]` sem `RunOnceId` — **curada**
(2 linhas), e o `Warning` **desapareceu** na recompilação. O hash mudou: é a prova de que
a cura entrou no artefacto.
⇒ ***«Um aviso do compilador é uma medição, não um ruído.»***

**A proxima accao unica**: **medir o artefacto gerado sem o instalar.** `INSTALLED=NAO`
continua verdadeiro e **instalar exige decisão do dono** (escreve em
`%LOCALAPPDATA%\Programs\Hades`). O que é interno e se pode fazer: **verificar o
instalador** com o `/SILENT`+`/DIR` num diretório descartável? Não — isso **é instalar**.
O que resta sem instalar: **extrair o conteúdo do Setup** (o Inno tem `/EXTRACT` em
`innounp`-like tools, mas não há ferramenta) **ou** documentar os hashes e entregar. Medir
primeiro o que é possível sem tocar em `%LOCALAPPDATA%`.

### A proxima accao unica (janela `E-1059`)

**O tarball Linux instalável existe.** O que havia era **um tarball, não um tarball
instalável**: 9 entradas, **sem `install.sh`, sem `uninstall.sh`, sem `VERSION`**.

⇒ ***«A diferença entre os dois estava num ficheiro de 400 bytes.»***

**E o binário Linux é de outro candidato** — medido no próprio ELF: `0.2.0-rc.2.0.
20261015171943-9f7379502b1d+dirty` ⇒ commit **`9f7379502b1d`**, **não** o HEAD. E o
`VERSION` do pacote **diz isso**: ***`build_reproduzivel=NAO`.***

**Sem compilador C e sem bash real** (`gcc`/`clang` ausentes; o `bash` do sistema é o do
WSL e responde `execvpe(/bin/bash) failed`), o `package_linux.sh` **não corre** — e no
entanto o pacote foi produzido. ⇒ ***«O caminho que exigia a ferramenta não era o único
caminho — outra vez.»***

**A proxima accao unica**: **macOS e Web/PWA são os próximos tecnicamente possíveis.**
macOS exige macOS/Xcode (`BLOCKED_EXTERNAL` pela máquina), mas o **Web/PWA** é build
local: `app/ui/app` tem `vite build` e o prompt pede «build de produção + manifest +
ícones + PWA». Verificar se existe script de build de produção e correr. **E esperar as
runs antes de qualquer push.**

### A proxima accao unica (janela `E-1060`)

**O build Web/PWA passa o orçamento**: `✓ built in 15,68s`, entry `index-CWgSFuKK.js`
**raw 1 509 050 / 2 000 000** e **gzip 412 527 / 600 000** ⇒ **`BUNDLE_BUDGET=PASS`**.

**E encontrei um defeito real que já estava no HEAD**: os **três** ícones do manifest
apontavam para `/icons/…` e **o diretório `icons/` nunca existiu** — os PNGs estão na
**raiz** de `public/`. `git show HEAD:…` confirmou: **já era assim no HEAD**.

⇒ ***«Um caminho escrito à mão é uma promessa; um caminho medido é um facto.»***
**Curado**: 0 de 3 ícones resolviam → **3 de 3**.

**E o que falta no PWA é medido, não suposto**: **zero service workers**
(`ocorrencias em src/: 0`; `dist/sw.js` e `dist/service-worker.js` ausentes) ⇒ **sem cache,
sem offline, sem instalação completa**. ⇒ **`PWA_INSTALAVEL=PARCIAL`**.

**A proxima accao unica**: **o service worker** é o que falta para o PWA ser instalável e
offline. Mas **offline honesto** exige decidir o que cachear (e o que **não** cachear: o
chat e as aprovações não podem vir de cache). Desenhar o escopo mínimo — shell + assets
estáticos, **nunca** respostas de API — e implementar. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1061`)

**O service worker existe** — e a **API nunca entra no cache**, por desenho e por teste.

⇒ ***«Um cache que responde por uma aprovação é um aprovador a mais.»***

`/api/*` é **network-only, sem excepção**: servir `/api/chat` do cache devolveria uma
resposta de outro momento como se fosse agora; uma aprovação do cache poderia autorizar
uma acção que já não está pendente.

**8 testes** e **contraprova com 3 defeitos injetados**: remover a guarda de `/api/` ⇒
**vermelho**; remover a guarda de método ⇒ **vermelho**; remover o `caches.delete` ⇒
**vermelho**; restaurar ⇒ **verde**.
⇒ ***«A contraprova é o que separa um teste de uma decoração.»***

**Suite completa**: **111 ficheiros, 738 testes, 0 falhas.**

**A proxima accao unica**: **E2E num browser real** — o service worker existe mas
**nunca foi instalado num browser**. Isso exige um servidor de produção (`vite preview`)
e um browser real; o Playwright está instalado. E **esperar as runs antes do push.**

### A proxima accao unica (janela `E-1062`)

**O PWA foi instalado e testado num Chromium real: 8/8.** O `E-1061` testou o `sw.js`
com `caches` **simulado** — *«prova a lógica; não prova a instalação»*. Este fecha isso.

| verificação | resultado |
|---|---|
| app abre | **200** |
| **service worker ACTIVO** | `script=…/sw.js`, `scope=/` |
| manifest | `name=Hades`, `display=standalone` |
| **3 ícones** | **200 · 200 · 200** |
| **`/api/*` fora do cache** | **404 basic** — chegou à rede |
| cache do shell | `hades-sw-v1-shell`, 2 caches |
| **offline** | **504 honesto** — não inventou conteúdo |
| pedidos falhados | 0 inesperados |

**E duas falhas iniciais eram do meu teste** (6/8): o SW **estava** registado e eu li o
`state` cedo demais; e contei como falha um pedido que **o próprio teste** provocou.
⇒ ***«Medir cedo demais é uma forma de inventar um defeito.»***

**A proxima accao unica**: **Firefox e WebKit** — o E2E só correu em Chromium. O Playwright
tem `webkit-2311` instalado; **Firefox não**. Instalar o Firefox do Playwright e correr os
mesmos 8 pontos. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1063`)

**O PWA passa 8/8 nos tres motores**: `chromium` **8/8**, `firefox` **8/8**, `webkit` **8/8**.
O `E-1062` só tinha corrido em Chromium.

**Faltavam dois dos três motores**: o `playwright-core` 1.64.0 espera `firefox-1555` e
`webkit-2370`; só existia `webkit-2311` (de versão anterior). `npx playwright install
firefox webkit` resolveu em **27,7 s**.
⇒ ***«Duas ferramentas instaláveis não são dois bloqueios externos.»***

**E a diferença offline prova que o teste mede o motor**: Chromium devolve **504**;
**Firefox lança `TypeError: NetworkError`**; WebKit devolve **504**. E o teste aceita
ambos porque **ambos são honestos** — nenhum inventa conteúdo.
⇒ ***«Se o teste medisse a si mesmo, os três dariam o mesmo resultado.»***

**A proxima accao unica**: **`PWA_INSTALADO_NA_UI` continua `NAO`** — o gesto de instalar
é humano. O que é interno e resta: **medir recursos do service worker** (memória, CPU,
tempo de `install`/`activate`, tamanho do cache) num browser real, e **testar upgrade
entre versões reais do SW** (publicar v2, activar, confirmar que a v1 é apagada).
**E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1064`)

**Os recursos do service worker medidos e o upgrade v1→v2 provado num browser real: 9/9.**

| medição | orçamento (declarado **antes**) | medido |
|---|---|---|
| entradas no cache | ≤ 400 | **9** |
| bytes no cache | ≤ 64 MB | **0,16 MB** |
| activação | < 20 s | **448 ms** |

⇒ ***«Um limite que se ajusta ao resultado não é um limite.»***

**E o upgrade**: com o servidor a servir `hades-sw-v2`, o browser **aceitou a v2**, o
`activate` **apagou a v1** («nenhum cache `hades-sw-v1-*`»), e o **`/api/*` continuou a ir
à rede** depois do upgrade.
⇒ ***«Um service worker que nunca é substituído congela o produto no dia em que é
instalado.»***

**A proxima accao unica**: **Firefox e WebKit não foram medidos** — o `E-1063` correu os
8 pontos nos três motores, mas **recursos e upgrade só em Chromium**. Correr o mesmo
`pwa-recursos-e-upgrade.mjs` nos três motores (o script já abre um servidor próprio na
porta 5180; basta parametrizar o motor como no `E-1063`). **E esperar as runs antes do
push.**

### A proxima accao unica (janela `E-1065`)

**Os recursos e o upgrade do SW passam 9/9 nos tres motores** -- e a latencia de activacao difere:

| motor | activacao | resultado |
|---|---|---|
| Chromium | **423 ms** | 9/9 |
| WebKit | **1041 ms** | 9/9 |
| Firefox | **1646 ms** | 9/9 |

Mesmo comando, mesma maquina, mesmo `dist/`, mesmo orcamento de 20 s.
=> ***«Se o instrumento se medisse a si mesmo, os tres dariam o mesmo numero.»***

E os tres, em todos os pontos: SW `activated`; `hades-sw-v1-shell`(3) + `hades-sw-v1-assets`(6);
**9 entradas, 160,6 KB, 0,16 MB** (identico ao byte); **upgrade v1->v2 aceite**; **v1 APAGADA**;
e **`/api/*` continua a ir a rede** depois do upgrade.

=> ***«Um numero nao e' uma explicacao.»*** -- o Firefox e' **3,9x** mais lento a activar que o
Chromium, e **eu nao medi por que**.

**A proxima accao unica**: **investigar o +1,2 s do Firefox na activacao do SW** -- e' um numero
medido e nao explicado. Instrumentar os marcos (`install`, `waiting`, `activate`) com
`performance.now()` no proprio `sw.js` e comparar. Se for estrutural do motor, registar como
limite; se for do nosso `addAll`, corrigir. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1066`)

**O tempo da activacao do SW esta' no `install` -- e o numero de 1646 ms do `E-1065` NAO era o SW.**

| motor | install | activate | fora | pagina (ready->activated) |
|---|---|---|---|---|
| Chromium | **18,2 ms** | **0,1 ms** | 1,0 ms | **73,2 ms** |
| Firefox | **132,0 ms** | **0,0 ms** | 3,0 ms | **262,0 ms** |
| WebKit | **72,0 ms** | **0,0 ms** | 12,0 ms | **308,0 ms** |

**H1 ACEITE** nos tres: o custo e' o `addAll` (shell + 6 icones). O `activate` custa **0 ms**.

**E a correcao da minha propria medicao**: o `E-1065` media desde **abrir a pagina**; este mede
desde `navigator.serviceWorker.ready`. O numero grande era **o browser a arrancar**, nao o SW.
=> ***«O cronometro estava certo; a etiqueta no cronometro e' que estava errada.»***

**E o veredito sobre optimizar**: `install` do Firefox **132 ms** contra orcamento de **20 s** =
**~150x de folga**. Nao se optimiza. E trocar o `addAll` atomico para ganhar 100 ms poria em risco
a garantia do `E-1061`: *«preferimos um SW que nao instala a um SW que instala meio shell»*.
=> ***«Nao se optimiza o que ja' esta' 150x dentro do orcamento.»***

**A proxima accao unica**: o perfil do SW esta' fechado. **Web/PWA tem agora 8 pontos (3 motores),
9 pontos de recursos/upgrade (3 motores) e o perfil do install/activate.** O que resta no Web e'
**`PWA_INSTALADO_NA_UI`**, que e' humano. *Avancar para uma plataforma por produzir* -- e a
**Android** e' a proxima tecnicamente possivel (Gradle local, APK nao assinado). Medir primeiro o
que o `apps/mobile-agentic` exige. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1067`)

**O APK Android: o projecto esta' completo, os tres requisitos estao' presentes, e o Gradle 8.13
nao carrega a `native-platform.dll` neste host.**

**Medido**: `apps/mobile-agentic/android/` **nativo completo** (`gradlew`, `gradlew.bat`,
`build.gradle`, `settings.gradle`, `local.properties`, `gradle-wrapper`); **`_tmp\android-sdk`
com 394,2 MB** (`android-34`, `build-tools 34.0.0`, `adb`); **`_tmp\jdk17` = openjdk 17.0.20.1**;
**Gradle 8.13** ja' extraido. Os tres requisitos estavam la'.

**3 tentativas, 3 hipoteses**: wrapper (`FileNotFoundException` no zip apagado) -> gradle directo
(`Failed to load native library`) -> `-Djava.io.tmpdir` explicito (mesmo erro). O `--stacktrace`
**nomeou a causa**: `NativeLibraryLocator.java:46` -> `createFileExclusively` ->
***«O sistema nao pode encontrar o caminho especificado»***.
=> ***«O protocolo antiloop manda parar a' terceira e registar -- e' o que faco.»***

**E isto NAO e' `BLOCKED_EXTERNAL`**: nao falta credencial, conta nem dispositivo. E' uma **falha
de ambiente reproduzivel**, com causa nomeada e passo de desbloqueio concreto: (1) confirmar o
`java.io.tmpdir` que o processo java ve; (2) criar esse directorio exactamente; (3) se continuar,
extrair a `native-platform.dll` do jar a' mao e passar `-Dorg.gradle.native.dir`.

**A proxima accao unica**: **fazer o passo (1)** -- `java -XshowSettings:properties -version` com o
JDK 17 e o `TMP` que o Gradle herda, para ver **que caminho** o `NativeLibraryLocator` tenta criar.
E' uma medicao de 1 comando, com hipotese nova -- nao e' a quarta tentativa cega. **E esperar as
runs antes do push.**

### A proxima accao unica (janela `E-1068`)

**A marca Android migrada em três categorias, e elas não são a mesma coisa**:

| categoria | acção |
|---|---|
| **UI visível** | muda — `DZ23 AGENTIC` → **`HADES`** |
| **identificadores de package** | mudam — `com.lmprado.dz23agentic` → **`com.dz23ltd.hades`** |
| **chaves de storage** (`dz23.agent.*`) | **FICAM** (4 + 2 ocorrências preservadas) |

⇒ ***«Uma migração de marca que apaga a sessão de quem já a usava não é uma migração: é uma
perda.»***

**E o `android/` não se commitou**: é **saída do `expo prebuild`** (35 ficheiros não
rastreados), e o `app.json` é a fonte — o `build.gradle` deriva do `app.json.android.package`.
A migração **correcta** é no `app.json`, e está feita.

**O plano está escrito onde se lê**: `apps/mobile-agentic/MIGRACAO_MARCA.md` — com a tabela
antes/depois, o que **não** mudou e por quê, e o que acontece a quem tem a versão antiga (no
Android o `applicationId` **é a identidade** — a versão nova é **outra app**; e a mitigação
honesta é que **não existe APK com o id antigo**, logo não há ninguém a quebrar).

**E quatro ficheiros que eu não toquei apareceram como `M`**: medido com
`git diff --ignore-cr-at-eol --numstat` → **vazio** ⇒ **só fim-de-linha**.

**A proxima accao unica**: **o inventário de marca mostra 231 `DZ23 Agentic` e 49
`Ollama DZ23` ainda no monorepo** — a migração do mobile reduziu, mas falta o resto. **A
proxima fatia é classificar as 231 ocorrências** por categoria (UI visível / nome técnico /
excepção) usando o próprio `HADES_BRAND_INVENTORY.md`, que já as classifica. **E esperar as
runs antes do push.**

### A proxima accao unica (janela `E-1069`)

**As 231 ocorrências de `DZ23 Agentic` classificadas NO SÍTIO delas: ~250 são nome técnico
e 19 são histórico — não há marca visível por migrar.**

**Como foi medido**: carreguei o **próprio gerador** por `importlib`, chamei `inventariar()`, e
para **cada** ocorrência olhei **50 caracteres de contexto** em cada lado, classificando contra
**10 padrões técnicos declarados antes**.

**As 19 «marca», uma a uma**: **3** em `MIGRACAO_MARCA.md` (o documento **que eu criei**, e que
**cita os nomes antigos de propósito** — é uma tabela antes/depois) · **14** no
`HADES_EVIDENCE_INDEX.md` (**histórico**, e o §FASE-7 proíbe reescrever evidência) · **1+1** no
`HADES_HANDOFF.md` e `HADES_PENDING_WORK.md` (documentos que eu escrevi, a citar o passado).
⇒ ***«Nenhuma é texto de interface de utilizador.»***

**E as ~250 técnicas**: `DZ23-LTDA` é o **owner do GitHub** · `dz23-agentic-quality` é o **nome
do workflow de CI** · `com.lmprado.dz23agentic` é **bundle id** · `dz23.agent.token` é **chave
de storage** · `Ollama Full`/`Ollama Classe A+`/`Ollama DZ23` são **nomes históricos de
produto** em documentos datados.

**A correção da minha hipótese**: eu esperava um erro de **contagem**; o erro estava na
**granularidade da categoria**. ⇒ ***«Um ficheiro migrável não faz das suas ocorrências marca
visível.»*** ⇒ ***«Quem procura um erro no número pode não ver o erro na pergunta.»***

**A proxima accao unica**: o **§FASE-5 item 2** está cumprido (mobile migrado) e **não tem mais
nada para migrar**. O item 3 («manter, e dizer por que») resta — e o inventário já o diz para
as **8 exceções**. ⇒ **fechar o §FASE-5** e **voltar ao bloqueio do Android** com a última
hipótese não testada: extrair a `native-platform.dll` do jar **à mão** e passar
`-Dorg.gradle.native.dir`. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1072`)

**O Gradle arranca com a DLL extraída à mão — mas o daemon não a herda.**

**O ganho, medido**: extraí a `native-platform.dll` (**141 312 B**) do
`native-platform-windows-amd64-0.22-milestone-28.jar`, e
`gradle --version -Dorg.gradle.native.dir=…` deu **exit 0** — `Gradle 8.13`, `Launcher JVM:
17.0.20.1`. **Antes disto, o Gradle nem `--version` fazia.**

**O bloqueio**: `assembleDebug` arranca um **daemon**, e o daemon falha com
***`NativeServices.initializeOnDaemon` → `createFileExclusively` → «O sistema não pode
encontrar o caminho especificado»***. **6 hipóteses, 10 tentativas**, todas falharam no daemon.

**E a pista que o log do daemon deu — e que diz que já funcionou**: em **2026-10-16T06:42:43**
o daemon **executou uma build** e falhou com ***`No Java compiler found`*** — **não** com a DLL.
⇒ ***«Um log antigo que ninguém lê é uma pista que ninguém usa.»***

**A próxima accao unica**: a **6ª hipótese falhou** (o `gradle.properties`), e o `android/` é
**gerado** — testar mais variantes ali é **trabalho descartável**. O caminho real é **o
`JAVA_HOME` do daemon**: o log diz que o daemon já correu, e o erro de então era `No Java
compiler found` — **a JVM do daemon não está a ver o JDK 17**. Verificar o `org.gradle.java.home`
e o `daemon JVM` que o `--version` reporta. **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1073`)

**O daemon do Gradle: 12 tentativas, 8 hipoteses, `BLOCKED` — e mudanca de trabalho.**

**As 8 hipoteses, todas medidas**: (1) `native.dir` no launcher → `--version` exit 0,
`assembleDebug` exit 1 · (2) `native=false` → o daemon **precisa** das native services ·
(3) `java.io.tmpdir` curto → **medi** que o efectivo é `AppData\Local\Temp` (o Java ignora o
`TMP` que eu defini) · (4) `GRADLE_OPTS`/`JAVA_OPTS` → não chegam ao daemon · (5) layout
alternativo → mesmo erro · (6) `gradle.properties` → mesmo erro · (7) **criar
`~/.gradle/native/`** (que **não existia**) → falhou igual · (8) `--info` → a única linha é
`Failed to load native library`, sem caminho.

**E a prova de que o daemon já carregou**: `daemon-3984.out.log` (10-16 06:42:43) tem
`[DEBUG] [NativeServices] Native-plat...` seguida de `Executing build` e de
`GeneratedClassCompilationException: No Java compiler found` ⇒ **o `NativeServices`
inicializou**; a falha de então era `javac` fora do `PATH`.

**Pelo protocolo antiloop, item 5**: *«se a mesma causa bloquear três tentativas, registe
BLOCKED com evidência e avance para trabalho independente»* — 12 tentativas é **quatro vezes**
o limite. ⇒ ***«Repetir com variação não é investigar — é insistir com disfarce.»***

**A proxima accao unica**: **mudar de trabalho** — não insistir no Gradle. O caminho interno
sem bloqueio é o **§FASE-7 item 3**: classificar os **73 IDs órfãos** por requisito/call site.
O desbloqueio do Android é **decisão do dono** (outro Gradle 8.14+/9.x, ou outro host/runner).
**E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1074`)

**Os 74 IDs orfaos classificados: todos dentro da faixa declarada, zero fora.**

**Como foi medido**: varri o repositorio por `E-(\d{3,})` e comparei com as **866** entradas
do indice. Resultado: **77 IDs citados sem entrada** — e desses, **3 são falsos positivos**
com prova: o `E-000` (sentinela), o **ano** de um aviso CVE, e a **cabeça de teste** do
proprio `hades_index_shape_check.py` ⇒ **74 orfaos reais**.

**O split exacto**: a faixa `E-571..E-777` tem **207** IDs ⇒ **74 citados + 133 não
citados = 207**. O prompt dizia **73 + 134** ⇒ ***a diferença é 1, e eu medi o número, não o
copiei.***

**E os 74 estão TODOS dentro da faixa; `FORA da faixa: 0`** ⇒ a catraca (`E-778`) **já os
permite** porque a faixa é **dívida declarada**. O §FASE-7 item 6 está cumprido.

**Onde os 74 vivem**: **63 em código** e **11 só em documentos**, em **88 ficheiros de
código**. Os que mais citam: `server/agent_routes.go` (18), `agenticClient.ts` (9),
`hades_local_gates.ps1` (7). A lista completa está em `_tmp/orfaos_tabela.md`.

**E a ironia que isto produziu**: ao **citar** os dois falsos positivos no texto do `E-1074`,
criei **dois novos** — porque o `CITACAO` é `\bE-(\d{3,})\b` e o texto tinha a forma exacta
de uma citação. ⇒ ***«O instrumento que mede o defeito participa do defeito.»***

**A proxima accao unica**: o **pacote de auditoria independente** (§FASE-7 item 8) —
`README_AUDITOR.md`, `PARIDADE_EVIDENCIAS.csv`, `GATES.md`, `LIMITACOES.md`,
`ARTEFATOS.sha256` — usando a classificação dos 74 como **alcance declarado**. **E esperar as
runs antes do push.**

### A cura do `E-1075` (a mesma janela)

**O `E-1074` expôs um defeito de paridade**: o `hades_handoff_window_check.py` **não** tinha o
limite inferior que o `hades_index_shape_check.py` já tinha ⇒ o `E-000` (que vira `0` no `int()`)
entrava como *«citado sem entrada»* em **qualquer** documento que o mencionasse.

**Cura**: a **mesma** regra (`_abaixo = min(registados)` e `c >= _abaixo`). ⇒ ***«Dois checkers, a
mesma regra, e um deles ficou para trás.»***

**Resultado**: `JANELA DE CITACAO - fault injection: 20 OK, 0 FAIL` (era 18/2), e a bateria voltou a
**33 OK / 0 FAIL**. ⇒ ***«O documento que documenta o falso positivo é o teste que o checker não
esperava.»***

### A proxima accao unica (janela `E-1076`)

**O pacote de auditoria independente (§FASE-7 item 8) existe — e é gerado, não escrito.**

**5 artefactos**, cada um função de uma fonte medível: `README_AUDITOR.md` (as secções, os
hashes, e **o que o auditor NÃO vai encontrar**) · `PARIDADE_EVIDENCIAS.csv` (as 869 entradas
do índice) · `GATES.md` (os **36** gates declarados, um por linha) · `LIMITACOES.md` (os **17**
blockers, a faixa `571..777`, os artefactos ausentes, a matriz por plataforma) ·
`ARTEFATOS.sha256` (SHA-256 dos artefactos que **existem**).

**Por que é gerado**: um pacote escrito à mão **envelhece no dia seguinte** — é o `E-1071`
outra vez. `--check` verifica que bate com o repositório.

**E o gerador tinha dois defeitos, ambos medidos**: (1) listava um zip que **já não existe** no
`dist/`; (2) separava os blockers resolvidos dos abertos por `"RESOLVIDO" in descricao` — e
**mediu mal**: dava 5 abertos e 12 resolvidos, quando há **8 abertos e 9 resolvidos**.
⇒ ***«É o `E-83` no meu próprio gerador: um detector lexical vê termos, não relações.»*** A
cura foi separar pelo **campo `tipo`**.

**24 verificações, 4 defeitos injetados**: artefacto apagado → exit 1; linha a mais → exit 1;
índice que muda → exit 1; e o **`E-83`**: um blocker com `tipo=RESOLVIDO` e a palavra «ABERTO»
na descrição fica em **Resolvidos**, e um com `tipo=EXTERNO` e a palavra «RESOLVIDO» fica em
**Abertos** — ambos verificados.

**A proxima accao unica**: o **`E-1076` fecha os 5 artefactos nomeados**; falta o **estado do
CI do mesmo snapshot** e a **lista de findings abertos/fechados** que o item 8 também pede.
E o **Android** continua `BLOCKED` (decisão do dono). **E esperar as runs antes do push.**

### A proxima accao unica (janela `E-1077`)

**A lista de findings do PR #438: 14 capturados da API, e o MD gerado do JSON — o `--check`
não precisa de rede.**

**O que foi medido**: `gh api …/pulls/438/comments` → **14 review comments**, todos de
`coderabbitai[bot]` → **5 Major** e **9 Minor**. O `FINDINGS.json` versionado guarda id, autor,
path, linha, data, severidade, categoria, título e trecho, mais o **SHA do checkout**
(`093390737`) e a **data da captura**.

**E os findings batem com o que o prompt descreve** — o que valida a captura: o #2
«Classifique cada uso de relógio separadamente» (item 2), o #4 «Continue a busca após o
primeiro `.Add(`» (item 3), o #5 «Vincule a triagem ao conteúdo» (item 4), o **#6 Major
«Protect every per-organization approval write»** (o **P0**, item 1), o #14 «Do not classify
partially closed findings as closed» (item 8).

**A decisão de desenho, por causa da rede**: dois passos. A captura (`--capturar`) escreve o
**JSON versionado** com data e SHA; a geração lê **só** o JSON. ⇒ ***«Um pacote de auditoria
que só se verifica com internet não se verifica num ambiente isolado.»***

**E os três defeitos que o gerador tinha, todos medidos**: (1) o `Total: 14 — .` saiu com a
lista vazia porque o regex procurava `"**Major**"` e o corpo tem `"**🟠 Major**"` — **o emoji
fica DENTRO do negrito**; (2) o **título** saiu como `🟠 Major` porque apanhava `negritos[1]`,
que é a severidade — o título é o **quarto**; (3) o ID injectado do DEFEITO 3 foi construído em runtime, e não escrito por extenso.
⇒ ***«Um índice por posição fixa quebra quando a decoração muda.»***

**O `FINDINGS.md` diz o que NÃO afirma**: não afirma que algum finding foi corrigido (nenhum
foi marcado no GitHub — proibido), não afirma que o bot está certo, não afirma que a lista é
completa.

**A proxima accao unica**: **os 14 findings ainda não foram validados contra o código**. O
`E-1070` mostrou o método (reproduzir num worktree limpo). O **#6 (P0)** e o **#2/#4/#5** são
os primeiros. **E esperar as runs antes do push.**
