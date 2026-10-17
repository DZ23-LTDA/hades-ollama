# Migração de marca: DZ23 Agentic → HADES

Executada em `E-1068`, por decisão do dono: *«Migra para `com.dz23ltd.hades` com plano de migração
explícito.»*

## O que mudou

| campo | antes | depois |
|---|---|---|
| `app.json` `name` | `DZ23 Agentic` | **`HADES`** |
| `app.json` `slug` | `dz23-agentic` | **`hades`** |
| `app.json` `scheme` | `dz23-agentic` | **`hades`** |
| `app.json` `android.package` | `com.lmprado.dz23agentic` | **`com.dz23ltd.hades`** |
| `app.json` `ios.bundleIdentifier` | `com.lmprado.dz23agentic` | **`com.dz23ltd.hades`** |
| `android/app/build.gradle` `namespace` | `com.lmprado.dz23agentic` | **`com.dz23ltd.hades`** |
| `android/app/build.gradle` `applicationId` | `com.lmprado.dz23agentic` | **`com.dz23ltd.hades`** |
| `package.json` `name` | `ollama-dz23-agentic-mobile` | **`hades-mobile`** |
| `App.tsx` UI (eyebrow) | `DZ23 AGENTIC` | **`HADES`** |
| `README.md` | `DZ23 Agentic Mobile` | **`HADES Mobile`** |

## O que NÃO mudou, e por quê

### As chaves de storage ficam: `dz23.agent.token` e `dz23.agent.base`

Mudar uma chave de storage **apaga a sessão do utilizador**. O prompt diz *«corrigir marca/package
futuro com plano de migração; não quebrar usuários silenciosamente»*. Uma migração de marca que
desliga a sessão de quem já a usava **não é uma migração: é uma perda**.

Estas chaves ficam com o prefixo `dz23.` **como exceção documentada**. Se um dia forem renomeadas,
tem de ser com **leitura das duas** durante uma versão de transição: ler `dz23.agent.token`; se não
existir, ler `hades.agent.token`; e só escrever o novo quando o antigo já tiver sido migrado.

### O `applicationId` mudou: o que acontece a quem tem a versão antiga

No Android, o `applicationId` **é a identidade da app para o sistema**. Mudar de
`com.lmprado.dz23agentic` para `com.dz23ltd.hades` faz o sistema tratar a nova app como **outra
app**: a versão antiga **não** é actualizada, fica instalada ao lado.

**Mitigação, e ela é honesta**:

- **não existe APK publicado com o id antigo** — o `E-1067` mostrou que o Gradle nunca chegou a
  produzir um APK neste host, logo **não há utilizadores a quebrar**;
- se um dia houver: o Gradle tem `MigrateAndroidApplicationId` (AGP 8+) ou um `BroadcastReceiver` de
  `MY_PACKAGE_REPLACED` para migrar dados;
- a app antiga **não** é desinstalada automaticamente — o utilizador tem de a remover.

## Como verificar

```
python -X utf8 -c "import json;e=json.load(open('apps/mobile-agentic/app.json'))['expo'];print(e['name'], e['android']['package'], e['ios']['bundleIdentifier'])"
# => HADES com.dz23ltd.hades com.dz23ltd.hades
```

## O que isto NÃO prova

**Não há APK** (`E-1067`: o Gradle não arranca neste host). A migração está feita no **código** e
**não foi testada numa build**.

*Uma migração de marca que nunca compilou está pronta a compilar — não está verificada.*
