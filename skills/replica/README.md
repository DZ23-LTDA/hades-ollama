# Pacote Replica — 11 skills de terceiros instaladas no Hades

**Isto não é código do Hades.** É o pacote [Replica](https://github.com/Jakeschincariol/replica-skill)
de **Jake Schincariol**, licenciado em **MIT**, trazido para dentro do repositório como
catálogo de skills. A origem exacta, o commit e o `sha256` de cada ficheiro estão em
[`PROVENANCE.md`](PROVENANCE.md) — e um teste confere-os a cada `go test`.

## O que está aqui

| caminho | o que é |
| --- | --- |
| `replica-*/` | o conteúdo upstream, **verbatim** (11 pastas: `SKILL.md` + auxiliares) |
| `manifests/*.json` | 11 manifestos nativos do Hades, **derivados** do frontmatter |
| `LICENSE` | MIT — © 2026 Jake Schincariol (**obrigatório**: a licença exige o aviso) |
| `UPSTREAM_README.md` | o README de origem, para consulta |
| `PROVENANCE.md` | origem, commit, data e `sha256` de cada ficheiro |

As 11 skills: `replica-recon` → `replica-entrepreneur` → `replica-architect` →
`replica-design` → `replica-build` → `replica-backend` → `replica-test` →
`replica-diff` → `replica-brand` → `replica-launch` → `replica-deploy`.

## Como o Hades carrega isto

O `ContextStore` carrega **uma pasta de ficheiros `.json`**:

```go
store, _ := agent.NewContextStore(root)          // cria <root>/skills
store.LoadSkillsForOrganization("skills/replica/manifests", agent.LocalOrganizationID)
skills := store.SkillsForOrganization(agent.LocalOrganizationID)   // 11
```

Pontos que **não** são óbvios e que custaram medição para descobrir:

- **A organização é obrigatória.** `LoadSkillsForOrganization(dir, "")` carrega, mas
  `SkillsForOrganization("")` devolve **zero** — ACL vazia é *ninguém*, nunca *todos*.
  O `LocalOrganizationID` do Hades é `"local"`.
- **A pasta de produção é diferente da pasta do repositório.** Aqui os manifestos são
  a **fonte de verdade** versionada; para o app carregar, copie-os para a pasta do
  `ContextStore` em uso — `<dataRoot>/.agent-context/skills/` (runtime) ou
  `<storeRoot>/context/skills/` (servidor):
  ```sh
  cp skills/replica/manifests/*.json <destino>/skills/
  ```
- **`disallowUnknownFields`.** Um campo que não exista em `SkillManifest` faz o
  carregamento **falhar** — não é ignorado em silêncio.

## Limitações reais (medidas, não supostas)

1. **As skills chegam `trusted=false` e não podem ser activadas.** O `trusted` **nunca**
   vem do JSON: o carregador força `false` e a única via de elevação é
   `CapabilityPolicy.PromoteSkillTrusted`, que exige **assinatura Ed25519** com chave
   autorizada (`signing_key_id` + `signature`). Não há chave configurada para este
   pacote, logo a activação está **bloqueada por assinatura** — de propósito: *catálogo
   não concede privilégio*. O teste prova que a promoção **recusa** (`ErrSkillKeyUnauthorized`).
2. **O `content_sha256` gravado está ligado à organização.** O campo `organization_id`
   entra nos bytes assinados (`SkillManifestSignatureBytes`), portanto o digest vale
   para `local`. Carregar para outro tenant **invalida** o digest e exigiria re-assinar.
   O valor já está preenchido com o digest medido pelo próprio Hades — assinar é um
   passo só.
3. **Nenhum `scopes` nem `tools` declarados.** Uma skill instalada não reivindica
   capacidade nenhuma; se um dia precisar de escopos, isso é uma decisão de política,
   não um efeito de instalação.
4. **`token_budget` é uma estimativa declarada** (`bytes do SKILL.md ÷ 4`), não uma
   contagem de tokens do modelo. Zero significaria «não declarado»; declarar um número
   aproximado é melhor do que omitir o custo — mas não é uma medição.
5. **O upstream grava CRLF nos blobs.** Por isso `.gitattributes` marca o pacote como
   `-text`: sem essa regra o `* text=auto` normalizaria para LF no commit e os `sha256`
   de `PROVENANCE.md` deixariam de bater noutro checkout (a armadilha do **E-183**).

## Como verificar

```sh
go test ./internal/agent/ -run Replica -v
```

Três testes, e cada um afirma uma coisa diferente:

- `TestReplicaPackCarregaOnzeSkills` — os 11 manifestos carregam, com os ids esperados,
  `trusted=false`, `enabled=true`, sem escopos e com orçamento declarado.
- `TestReplicaPackDigestEProcedenciaIntactos` — `content_sha256` bate com o digest
  calculado **agora** (catraca: editar `description`/`version` quebra) e os **29**
  `sha256` de `PROVENANCE.md` batem com os ficheiros em disco, sem linhas órfãs.
- `TestReplicaPackRecusaCampoDesconhecido` — `trusted:true` no JSON é ignorado e um
  campo inventado faz o carregamento falhar.

## Se for actualizar o pacote

1. Clone o upstream e **fixe o commit** (`git rev-parse HEAD`).
2. Rode o instalador (`_tmp/install_replica_skills.py`, que valida antes de escrever) e
   depois a passagem do digest — ela usa o código Go do projeto, não uma reimplementação.
3. **Não edite** nada dentro de `replica-*/`: o `sha256` deixaria de bater e o teste
   falharia — o que é o objectivo. Conteúdo de terceiros edita-se no upstream.
