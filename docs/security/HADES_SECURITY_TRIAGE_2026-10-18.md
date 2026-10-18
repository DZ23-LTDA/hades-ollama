# Triagem de Segurança — Relatório GitGuard de 2026-10-18

**Estado:** triagem em curso · **Veredito global:** `NOT_RELEASE_READY`
**Data:** 2026-10-18

---

## 1. Baseline (medido, não narrado)

| item | valor |
|---|---|
| repositório | `DZ23-LTDA/hades-ollama` |
| branch | `fix/checker-relogio` |
| HEAD no início da triagem | `81e7293c5226fe9840a60746b5101abf2c1881ed` |
| upstream | `origin/fix/checker-relogio` |
| `origin/main` | `4b3e2185422c24d5a05fd46a343953ad6f41d1e5` |
| merge-base (HEAD, origin/main) | `4b3e21854…` — **`origin/main` É ancestral de HEAD** |
| ahead / behind vs main | 603 / 0 |
| PR | **#438, OPEN**, `main ← fix/checker-relogio`, `mergeable_state = blocked` |
| WIP no início | **árvore limpa** |

**O relatório GitGuard analisou a `main` no commit `4b3e21854`** — que é, hoje, exactamente o `origin/main`.
O PR está **603 commits à frente** desse ponto. *Comparar contagens de achados entre a `main` e o HEAD do PR sem dizer isto seria fingir que são o mesmo código.*

### O que este documento NÃO faz

- Não replica a instrução do relatório ("aplique as correções diretamente nesse checkout"): a autorização vigente permite escrita **apenas** em `fix/checker-relogio`.
- Não declara "tudo corrigido" por redução de contagem.
- Não revela valores de segredos: os achados Gitleaks são tratados por **caminho + linha + natureza**.

---

## 2. Resumo do relatório original

**Scan** `cmvcz9czn00dond5btxcrch49` · MANUAL · branch `main` · commit `4b3e2185422c`
**Total: 339 findings** — HIGH 84 · MEDIUM 249 · LOW 6 · SEMGREP 326 · GITLEAKS 13

| regra | n | sev | superfície dominante |
|---|---|---|---|
| `unsafe.use-of-unsafe-block` | 198 | MED | **upstream / camada nativa** (app/, mlx/, app/wintray/, app/dialog/) |
| `dangerous-exec-command` | 65 | HIGH | HADES (agent, cmd/launch, app/server, discover) |
| `crypto.math_random.math-random-used` | 9 | MED | misto |
| `unsafe-formatstring` (JS) | 6 | MED | UI |
| `use-of-md5` | 6 | MED | misto |
| `path-join-resolve-traversal` (JS) | 5 | MED | UI/build |
| `import-text-template` | 5 | MED | misto |
| `node_insecure_random_generator` | 4 | MED | mobile/UI |
| `dependabot-missing-cooldown` | 4 | MED | governança CI |
| `potential-dos-via-decompression-bomb` | 2 | MED | HADES |
| `unsafe-reflect-by-name` | 2 | MED | misto |
| `invalid-usage-of-modified-variable` | 2 | — | misto |
| `missing-unlock-before-return` | 1 | **HIGH** | **HADES — `server/sched.go:609`** |
| `missing-user` + `missing-user-entrypoint` | 2 | **HIGH** | **HADES — `Dockerfile`** |
| `use-defused-xml` | 1 | **HIGH** | **HADES — script de verificação** |
| `string-formatted-query` | 1 | — | HADES |
| `use-tls` | 1 | **HIGH** | misto |
| `reverseproxy-director` | 1 | — | HADES |
| `use-of-sha1` | 1 | MED | HADES (`deploy.go`) |
| `github-actions-mutable-action-tag` | 1 | **HIGH** | supply chain CI |
| `react-insecure-request` | 1 | **HIGH** | UI/e2e |
| `hardcoded_secrets.node_secret`/`node_username` | 2 | **HIGH** | UI |
| `dynamic-urllib-use-detected` | 1 | — | scripts |
| `incomplete-sanitization` | 1 | — | JS |
| `regex_dos` | 1 | — | JS |
| `no-direct-write-to-responsewriter`/`no-fprintf-…` | 2 | — | HADES |
| `insecure-use-string-copy-fn` (C) | 1 | — | nativo upstream |
---

## 3. Achados CONFIRMADOS e corrigidos nesta fatia

### A. `Dockerfile` — processo final como root (HIGH, 2 achados)

**Antes:** `FROM ubuntu:24.04` … `ENTRYPOINT ["/bin/ollama"]`, `CMD ["serve"]`, **sem nenhuma directiva `USER`** — e `ENV OLLAMA_HOST=0.0.0.0:11434`, ou seja, a nascer **root e exposto**.

**Corrigido:**
- utilizador/grupo não privilegiado com **uid/gid 1000 fixos** (números, não nome: *um volume montado fora precisa de um número estável para o `chown` casar*);
- `chown` **apenas** de `/usr/share/ollama` (o único directório gravável) e `chmod 0755`;
- `ENV OLLAMA_MODELS` apontado para esse directório; `VOLUME` declarado;
- **`USER 1000:1000`**;
- `OLLAMA_HOST` passou a **`127.0.0.1:11434`** (item C: o padrão é loopback);
- `HEALTHCHECK` com `["/bin/ollama", "list"]` (sem shell).

**GPU não quebra por uid:** o acesso a `/dev/nvidia*` e `/dev/dri` vem do **runtime** (NVIDIA Container Toolkit), não do uid; o runtime acrescenta os grupos suplementares ao processo já sem root. **Não medido neste host** (sem GPU no container).

**Estado:** `CORRIGIDO_NAO_VALIDADO_EM_HARDWARE`.

### B. `server/sched.go:609` — `panic` com lock preso (HIGH)

**Antes:**
```go
s.loadedMu.Lock()            // linha 522
...
if s.activeLoading.ModelPath() != wantPath {
    panic(fmt.Errorf("attempting to load different model after eviction …"))  // linha 609
}
...
s.loadedMu.Unlock()          // linha 613
```
O `panic` está **88 linhas depois do `Lock`** e é o **único dos 6 caminhos de saída que não liberta o lock** (os outros 5 nas linhas 536, 564, 598, 613, 652).

**Porque é confirmado e não "defensivo inalcançável":** *um lock preso por `panic` é um deadlock que sobrevive à causa.* Mesmo que o caminho seja raro, o efeito é **indisponibilidade total do servidor** em vez de uma requisição falhada — e o `recover` de um handler HTTP não liberta um `sync.Mutex` alheio.

**Corrigido:** `panic` → erro controlado no **mesmo padrão das linhas 595-600** (`slog.Error` + `req.errCh <- err` + `s.loadedMu.Unlock()` + `return false`). A invariante continua a ser verificada; muda o que acontece quando é violada.

**Nenhum teste esperava o `panic`** (`git grep 'attempting to load different model'` só encontra a própria linha).
**Estado:** `CORRIGIDO` · `go test ./server` **PASS 25,077 s**.

### B2. `internal/agent/builder_staging.go` — PATH e ambiente herdados (HIGH)

**Dois defeitos reais que o scanner não nomeia:**

1. `exec.CommandContext(ctx, "go", "build", …)` resolvia **`go` pelo PATH do operador**. Um `go` plantado no PATH executaria código arbitrário **no caminho de deploy**.
2. `compilar.Env = os.Environ()` e `processo.Env = append(os.Environ(), …)` entregavam ao filho **todas as variáveis do operador**, incluindo tokens — que o filho nunca pediu.

**Corrigido:**
- `go` resolvido por `defaultGovernedCLIResolver().resolve("go")` — o **mesmo resolvedor governado** que o `cli_harness` já usava;
- ambiente do build = `governedCLIEnv` (lista de permissão + marcadores que **removem** `_TOKEN`, `_API_KEY`, `GITHUB_`, `AWS_`, …);
- **`GOCACHE` próprio** dentro do temporário (também impede poluir o cache do operador);
- **`GOTOOLCHAIN=local`** — um build reprodutível não escolhe o próprio compilador por omissão de rede;
- `GOROOT`/`GOPATH` do operador **deliberadamente não herdados**;
- o filho do staging (a aplicação gerada) recebe `governedCLIEnv` + **apenas** as 3 variáveis que usa.

**Teste discriminador:** `TestAmbienteGoNaoHerdaCredenciaisDoOperador` define `GITHUB_TOKEN`, `OPENAI_API_KEY`, `*_TOKEN`, `*_API_KEY`, `*_PASSWORD` no ambiente e **exige que nenhum chegue ao filho**, ao mesmo tempo que exige que `GOCACHE` e uma variável inofensiva (`LANG`) **cheguem**. `TestAmbienteGoNaoHerdaGOROOTNemGOPATH` cobre o resto.
**Estado:** `CORRIGIDO` · `go test ./internal/agent` **PASS 308,194 s**.

### C. `scripts/hades_discoverability_check.py:28` — `xml.etree.ElementTree` (HIGH)

**Superfície real:** `validar_sitemap` faz `ET.fromstring(caminho.read_text())`. O Python 3 **não** resolve entidades externas por omissão, mas **resolve entidades internas** (billion laughs) e não tem limite de tamanho.

**Corrigido sem dependência externa** (não há `requirements.txt`/`pyproject.toml`; acrescentar `defusedxml` a um script que corre em qualquer máquina trocaria um risco por uma fragilidade de instalação):
- recusa de `<!DOCTYPE` e `<!ENTITY` (após normalizar maiúsculas e espaços — *recusar só a forma que eu imaginei seria um filtro que se testa a si próprio*);
- limite de 8 MiB antes de parsear.

**Testes acrescentados (3):** DOCTYPE recusado em três grafias, ENTITY recusado, e **sitemap normal continua a passar** — *um filtro que recusa tudo também passa nos testes de recusa*.
**Estado:** `CORRIGIDO` · 11 testes do script **OK**.
---

## 4. Achados `NOT_ACTIONABLE` com prova

### B3. `internal/agent/mcp.go:452` — `dangerous-exec-command`

A função `Register` valida, **antes** de chegar ao `exec.CommandContext`:

| defesa | linha |
|---|---|
| ID e `Command` obrigatórios | 100-102 |
| `Command` tem de ser **caminho absoluto** | 107-109 |
| tem de existir, **ficheiro regular**, **não symlink**, executável | 110-113 |
| **`rejectGenericInterpreterCommand`** recusa `sh`, `bash`, `cmd`, `powershell`, … | 114-116 |
| `Args` ≤ 64, sem NUL, ≤ 4096 B, **sem credenciais literais** | 117-127 |
| `AllowedMethods` obrigatório | 129-139 |
| `EnvironmentVars` ≤ 32, nome validado, valor do ambiente com limite 32 KiB | 140-150 |
| `WorkingDirectory` absoluto, criado `0700` | 152-156 |
| escopo por organização (`ErrPluginOrganizationScope`) | 159-163 |

No `startLocked`: `cmd.Dir` ancorado, **`cmd.Env = ["PATH=/usr/bin:/bin", "HOME=/tmp"]`** (PATH fixo — **não herda o host**), env só por nome, `stderr` limitado.
A exceção para interpretadores é **allowlist por caminho exacto** (`OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST`) — não por sufixo nem regex. Teste de negação: `TestMCPRejectsLiteralCredentialArgumentsAndHidesArgsFromTenantCatalog`.
**Veredito:** o scanner vê a **primitiva**; não vê as 20 validações que a precedem. `NOT_ACTIONABLE`.

### B4. `internal/agent/tools.go:280` e `:764`

`trustedToolExecutable()` resolve o binário; `validateTerminalArguments()` valida; `command.Dir` é **pinado pelo descritor aberto** (`/proc/PID/fd/N`); `Env = ["PATH=/usr/bin:/bin", "HOME=/", "PWD=…"]`; saídas em `limitedBuffer` de 64 KiB; timeout 30 s.
O sítio `:764` é o **sandbox** `unshare` com `--user --map-root-user --mount --pid --fork --mount-proc **--net**` — rede negada por construção. `NOT_ACTIONABLE`.

### B5. `internal/agent/git_repo.go:491`

`gitExecutable` resolvido antes; `command.Dir` ancorado; ambiente **`PATH=safeToolPath()`, `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_SYSTEM=`, `GIT_CONFIG_GLOBAL=`, `GIT_ATTR_NOSYSTEM=1`, `GIT_DIR=`**; `-c core.fsmonitor=false -c core.pager=cat -c diff.external=` (neutraliza hooks e pagers); timeout 8 s. `NOT_ACTIONABLE`.

### B6. `internal/multillm/proxy.go:381`

`registry.go` exige, para `ProviderTypeCLI`: **`Executable` absoluto** (219), **`AllowExecution=true`** opt-in explícito (222), args sem `\r\n\x00` (225-228), paths de protocolo proibidos (230-234). Env de `cliCommandEnv` → `procenv.Minimal`. `limitedBuffer` 16 MiB/64 KiB, timeout ≤ 30 min.
Para tipos HTTP: **HTTPS obrigatório**, `http` permitido **só** com `AllowInsecureLoopback && AllowPrivate && isLoopbackHostname` (240-242) — **é o item C do adendo, já implementado**. `NOT_ACTIONABLE`.

### C2. Item C (TLS/rede local) — já implementado e testado

`app/server/server.go:287` chama **`forceLoopbackHostSpec(host)`**; `app/server/host_loopback_test.go` cobre `0.0.0.0:11434 → 127.0.0.1:11434`; `server/cloud_proxy.go` usa `isLoopbackHost`; `server/agent_auth_policy_test.go` testa `127.0.0.1` **vs** `0.0.0.0` contra a política de autenticação.
`NOT_ACTIONABLE` **no código**; o que faltava era o **Dockerfile nascer em loopback** — corrigido em A.

### E. GITLEAKS — 13 achados (nenhum valor reproduzido)

| # | caminho:linha | natureza | veredito |
|---|---|---|---|
| 1 | `internal/agent/omnichannel_test.go:235` | `SecretEnv: "<nome>"` — **nome** de variável, não valor | `NOT_ACTIONABLE` |
| 2 | `internal/agent/deploy_test.go:315` | fixture de resposta HTTP de teste | `NOT_ACTIONABLE` |
| 3-5 | `internal/agent/whatsapp_secret_leak_test.go:22,24,25` | **o próprio teste de DLP** | `NOT_ACTIONABLE` |
| 6 | `internal/agent/workspace_write_test.go:188` | linha vazia | `NOT_ACTIONABLE` |
| 7 | `internal/agent/workspace_snapshot_test.go:39` | fixture `config.txt` com `api_key=<sintético>` | `NOT_ACTIONABLE` |
| 8 | `internal/agent/workspace_snapshot_test.go:1106` | chave de tenant sintética | `NOT_ACTIONABLE` |
| 9 | `internal/agent/runtime_test.go:399` | fixture de **payload corrompido** para testar redacção | `NOT_ACTIONABLE` |
| 10 | `internal/proxy/codex_desktop_test.go:1585` | header de teste (valor sintético) | `NOT_ACTIONABLE` |
| 11 | **`model/parsers/glm46.go:32`** | **`glm46ArgKeyOpenTag = "<arg_key>"`** — **tag XML do protocolo de tool-call do GLM-4.6** | `NOT_ACTIONABLE` (falso positivo) |
| 12 | `internal/agent/company_ledger_test.go:311` | chave sintética de teste | `NOT_ACTIONABLE` |
| 13 | `README.md:186` | prosa: a palavra `api_key` em documentação | `NOT_ACTIONABLE` |

**Nenhum é credencial activa.** Não foi adicionada allowlist ampla, nem regex que esconda chave real, nem exclusão do scanner: **o scanner continua a apontar os 13**; o que existe é esta classificação com prova.

---

## 5. `NEEDS_REVIEW` — dívida de hardening, sem exploração demonstrada

| achado | porque não foi fechado nesta fatia |
|---|---|
| `mlxrunner/client.go:350` — `cmd.Env = os.Environ()` | Subprocesso do **próprio binário** (`os.Executable()` + `EvalSymlinks`), mas herda o ambiente do operador. Fix proporcional: `procenv.Minimal` + vars de biblioteca do MLX. **Não feito porque não há como validar MLX neste host** — mudar o ambiente de um runner que não consigo executar é trocar um risco por uma regressão invisível. |
| `internal/agent/auth.go:566` — `hmac.New(sha1.New, …)` | HMAC-SHA1 não está quebrado como o SHA-1 puro, mas SHA-256 seria melhor. Migrar **invalida tokens existentes**; exige versão/rotação planeada. |
| `internal/agent/deploy.go:570` — `sha1.Sum` | **É o contrato da API do Netlify** (espera o SHA-1 dos ficheiros). Não é decisão de segurança nossa; a integridade real vem de HTTPS. Documentado, não alterado. |
| `crypto/math_random` (9) | Verificar caso a caso: se algum escolhe token/nonce/sessão, é `CONFIRMED`. Não fechado nesta fatia. |
| `import-text-template` (5), `string-formatted-query` (1), `reverseproxy-director` (1) | Sem sink demonstrado até agora; exigem traçar fonte→validação→sink antes de julgar. |
| `unsafe` (198) | **Camada nativa/upstream** (`app/`, `mlx/`, `app/wintray/`, `app/dialog/`). *Reescrever 198 sítios de FFI sem poder correr Windows/macOS/GPU aqui seria a forma mais eficaz de quebrar o produto e chamar-lhe segurança.* |
| Dependabot cooldown (4), `github-actions-mutable-action-tag` (1) | Governança de CI; mudam o pipeline e exigem decisão do proprietário sobre janela de atualização. |
| `use-tls` (1), `react-insecure-request` (1), `hardcoded_secrets.node_*` (2) | A localizar e julgar na próxima fatia. |

---

## 6. `BLOCKED_EXTERNAL`

| item | blocker | desbloqueio |
|---|---|---|
| build/execução do `Dockerfile` | **Docker ausente neste host** — não há como provar `docker build`/`run`/`id` ≠ 0 | host/CI com Docker |
| validação GPU sem root | exige GPU + NVIDIA Container Toolkit | runner com GPU |
| MLX runner env | exige macOS/Apple Silicon | runner macOS |
| scan actualizado no HEAD | **não há Semgrep/Gitleaks instalados neste host**; o relatório é de terceiro | instalar scanners ou correr o workflow dedicado |

---

## 7. Riscos residuais e próximos passos

1. **O scan é da `main`; o código é do PR #438 (603 commits à frente).** A triagem acima é sobre o **código do PR**; a comparação de contagens com o relatório antigo **não é medida de progresso**.
2. **`gitleaks` continua fora do CI** (gap `E-1164`): a cura está desenhada mas **não activada**. *Um secret scan que só existe quando alguém se lembra de correr não é um gate: é uma auditoria.*
3. Os 198 `unsafe` e parte dos `exec.Command` de `cmd/launch/` são **upstream/launcher** — classificados separadamente, não "fechados por serem antigos".
4. **Nada aqui autoriza merge, release ou publicação.** O veredito permanece `NOT_RELEASE_READY` até CI concluído, auditoria independente, homologação real e aprovação humana explícita.