# Correção — CI vermelho no teste de cancelamento de processo (E-115)

## 1. O que aconteceu

O PR #188 (fábrica de empresa) abriu **CI vermelho** em dois gates:
`Go agentic Windows portability` e `PR gate`. O passo que falhou foi *Agent runtime tests*:

```
--- FAIL: TestRunToolCommandKillsProcessGroupOnCancellation (2.37s)
    tools_test.go:122: cancellation took too long: 2.3739172s
```

Isso **contradiz** o que eu vinha declarando: eu tratava essa falha como *"específica do host,
passa no CI"*. Para **este** teste, o relato estava **errado**.

## 2. A causa real, medida

O teste tinha **dois** problemas independentes:

1. **Dependia do `sh` do sistema** para criar o processo-alvo. Nesta máquina **não existe
   `sh`**, então o teste **nunca era realmente exercitado aqui** — e eu não tinha como
   verificá-lo. Era essa a origem do "falha só no host".
2. **Limite de tempo arbitrário de 2 segundos** para derrubar o grupo de processos. No
   Windows, a **primeira** derrubada é lenta: reproduzi localmente e a primeira execução levou
   **2,53s**, exatamente o padrão do CI (as execuções seguintes levam **0,05s**). O limite não
   media o cancelamento: media a **carga da máquina** naquele instante.

## 3. A correção (sem enfraquecer a verificação)

| Antes | Depois |
| --- | --- |
| `exec.Command("sh", "-c", "sleep 10")` | o **próprio binário de teste** em modo auxiliar (`TestSleepHelperProcess` + `HADES_TEST_SLEEP_HELPER=1`) — **portátil** e verificável em qualquer host |
| limite de **2s** arbitrário | limite ligado à **propriedade medida**: metade do próprio sono (10s), ou seja **5s** — continua reprovando qualquer cancelamento que apenas "espere" o processo terminar |
| erro precisa conter `deadline exceeded` | **inalterado**: a asserção de corretude é a mesma |

## 4. Verificação

```
go test ./internal/agent/ -run TestRunToolCommandKillsProcessGroupOnCancellation -count=3
--- PASS (2.53s)   ← a primeira derrubada, que antes reprovava
--- PASS (0.05s)
--- PASS (0.05s)
```

- **Três PASS**, e a primeira execução em **2,53s** documenta a causa em vez de escondê-la.
- `golangci-lint` **0 issues** (o falso positivo do `misspell` foi **reescrito**, não silenciado).
- Depois da correção, o PR #188 ficou **verde** (20 checks, 0 falhas) e foi integrado; a `main`
  ficou com **7 checks verdes**.

## 5. Estado corrigido das falhas "ambientais"

| Teste | Antes | Agora |
| --- | --- | --- |
| `TestRunToolCommandKillsProcessGroupOnCancellation` | declarado "ambiental (sem `sh`), passa no CI" | **CORRIGIDO**: portátil, passa **nesta máquina** e no CI |
| `TestLargeUploadProjectImportStreamsZipAndCreatesIsolatedWorktree` | ambiental (caminho longo/`git` no Windows) | segue **ambiental** |
| `TestImportMissionAttachmentsCreatesIndexedOrgProject` | ambiental (caminho longo/`git` no Windows) | segue **ambiental** |

São **duas** falhas ambientais hoje, não três — e a que saiu da lista saiu por **correção**, não
por omissão.
