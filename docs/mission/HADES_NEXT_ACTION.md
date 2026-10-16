# HADES — PRÓXIMA AÇÃO (V2.2 + adendo de paridade)

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
