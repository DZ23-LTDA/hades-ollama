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

## O quinto sítio: os ficheiros Kotlin (E-1293)

**Esta tabela listava quatro sítios onde o `namespace`/`applicationId` mudava, e nenhum deles era o
codigo Kotlin.** A migração cumpriu os quatro e deixou o quinto de fora:

```
android/app/src/main/java/com/lmprado/dz23agentic/MainActivity.kt    package com.lmprado.dz23agentic
android/app/src/main/java/com/lmprado/dz23agentic/MainApplication.kt package com.lmprado.dz23agentic
```

O `AndroidManifest.xml` referencia as duas classes com **nome relativo** — `.MainActivity` e
`.MainApplication` — e **nao declara atributo `package`** (o AGP 8.x removeu-o). Um nome com ponto
inicial resolve contra o **`namespace` do Gradle**: o Android procurava
`com.dz23ltd.hades.MainActivity`, **que nao existia**, e lancava `ClassNotFoundException` no
arranque. **A aplicacao nao abria.**

    *Um `namespace` que nao corresponde ao package do codigo e' um endereco para uma casa que nao
    existe: o build passa e o app nao abre.*

O build nao apanhava porque a compilacao Kotlin **nao exige** que o `package` corresponda ao caminho,
e o Gradle **nao verifica** que o `namespace` case com as classes declaradas. So' o dispositivo
encontrava -- e neste host nao ha' Android SDK nem emulador.

**Corrigido no `E-1293`:** os dois `.kt` foram movidos para
`android/app/src/main/java/com/dz23ltd/hades/` com `package com.dz23ltd.hades`.

**Licao para a proxima migração de identificador:** a lista tem de incluir o **codigo-fonte**, e nao
so' os ficheiros de configuracao. Um `namespace` e' um endereco; o `package` e' a casa. Mudar um sem o
outro deixa o endereco a apontar para o vazio.

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

**A verificacao que realmente decide se a app ABRE** e' a catraca do `E-1293`:

```
python scripts/hades_android_package_check.py          # exit 0 = coerente
python scripts/hades_android_package_check_test.py     # 18 PASS, inclui o defeito real
```

Ela verifica as cinco coisas que o `python -c` acima nao ve: que o `package` de cada `.kt`
corresponde ao caminho, que o `namespace` tem uma classe com esse nome, que **todo nome relativo do
manifest resolve para uma classe que existe**, que o `applicationId` e' coerente, e que o package
historico nao voltou a aparecer em codigo.

Esta catraca corre no **CI** (`.github/workflows/class-a-plus-integrity.yaml`) e no
**`scripts/hades_local_gates.ps1`** (seccao 9d). O `python -c` acima confirma a configuracao; a
catraca confirma a **resolucao de nomes em tempo de execucao** -- e era essa que estava quebrada.

## O que isto NÃO prova

**Não há APK** (`E-1067`: o Gradle não arranca neste host). A migração está feita no **código** e
**não foi testada numa build**.

*Uma migração de marca que nunca compilou está pronta a compilar — não está verificada.*
