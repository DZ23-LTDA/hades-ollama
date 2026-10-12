# Code signing do instalador Windows

## Estado atual

O Hades está preparado para solicitar assinatura gratuita pelo **SignPath Foundation**. Até a aprovação da Foundation e a configuração do secret pelo proprietário, os builds saem **UNSIGNED**. Isso é intencional e honesto: o CI nunca declara assinatura sem receber e aplicar um artefato assinado real.

A frase exigida para a política pública do projeto está no README:

> Free code signing provided by SignPath.io, certificate by SignPath Foundation

## Configuração SignPath

Os identificadores abaixo são específicos do projeto criado no dashboard SignPath e devem ser preenchidos pelo proprietário; não são inventados ou inferidos pelo código. Em **Settings → Secrets and variables → Actions → Variables**, configure:

| Variable | Valor |
| --- | --- |
| `SIGNPATH_ORGANIZATION_ID` | Organization ID fornecido pelo SignPath |
| `SIGNPATH_PROJECT_SLUG` | Project slug aprovado no SignPath |
| `SIGNPATH_SIGNING_POLICY_SLUG` | Signing policy slug aprovada |
| `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG` | Artifact configuration slug com raiz `<zip-file>` |

Em **Secrets**, configure:

| Secret | Uso |
| --- | --- |
| `SIGNPATH_API_TOKEN` | Token SignPath com permissão de submitter para o projeto/policy |

Nunca coloque o token no código, em documentação, em artefatos ou em logs.

## Fluxo do workflow

O workflow `.github/workflows/dz23-windows-installer.yaml` usa apenas `windows-latest`, builda `dist/OllamaFullSetup.exe` e funciona em dois estados:

1. **Sem `SIGNPATH_API_TOKEN`:** pula a assinatura, calcula o checksum do executável unsigned e publica `OllamaFullSetup-windows-amd64-unsigned`.
2. **Com `SIGNPATH_API_TOKEN`:** envia o executável ao GitHub Actions Artifact, solicita a assinatura usando a configuração SignPath, extrai o `.zip` assinado, substitui o executável local e só então calcula o checksum; publica `OllamaFullSetup-windows-amd64-signed`.

A configuração `hades-exe` deve aceitar um `<zip-file>` na raiz, porque o `actions/upload-artifact@v4` fornece o arquivo ao SignPath como ZIP. A action oficial de submissão usa o ID do artefato produzido pelo upload anterior.

O workflow registra `INSTALLER_SIGNING=SIGNED` ou `INSTALLER_SIGNING=UNSIGNED` no log e no resumo do job. A assinatura só pode ser considerada homologada depois de um run real com token válido e verificação do Authenticode no `.exe` retornado.

Referência oficial: [SignPath — GitHub Actions](https://docs.signpath.io/trusted-build-systems/github).

## Como disparar

- **Actions → dz23-windows-installer → Run workflow**; ou
- push de uma tag `v*`, conforme os gatilhos do workflow.

A Foundation ainda precisa aprovar o projeto e habilitar o Trusted Build System `GitHub.com`. Reputação do projeto, release pública e eventual fork visível são responsabilidades do proprietário durante o processo de elegibilidade; não são simuladas pelo código.

## Metadados e desinstalação

O instalador usa ProductName `Hades`, CompanyName `DZ23 LTDA`, versão derivada de `git describe`/`PKG_VERSION` e gera desinstalador por padrão do Inno Setup. Os binários Go e o aplicativo desktop recebem a mesma versão derivada pelo script de build.

## Outras plataformas

- **macOS:** exige Apple Developer Program (US$ 99/ano), certificado e notarização; permanece como trabalho separado.
- **Linux:** pacotes permanecem unsigned por padrão; assinatura de pacotes Linux é um próximo passo.

## Notarização macOS (App Store Connect API key)

O projeto notariza em **macOS** (`xcrun notarytool`). Há dois modos de autenticação, e o script escolhe sozinho:

| Modo | Variáveis | Quando é usado |
| --- | --- | --- |
| **Chave de API (.p8)** | `APPLE_API_KEY` (caminho), `APPLE_API_KEY_ID`, `APPLE_API_ISSUER` | quando as **três** estão preenchidas |
| Apple ID | `APPLE_ID`, `APPLE_PASSWORD`, `APPLE_TEAM_ID` | caso contrário (comportamento antigo) |

A seleção está em [`scripts/notarytool-auth.sh`](../scripts/notarytool-auth.sh) e é **testada** exercitando os três
casos (chave completa, chave ausente e chave incompleta) sem chamar o `notarytool`.

### O que a chave de API faz — e o que ela **não** faz

- **Faz**: autentica a **submissão** ao serviço de notarização e outras operações do App Store Connect.
- **Não faz**: **assinar** o binário. Assinar continua exigindo o certificado **Developer ID Application**
  (`APPLE_IDENTITY`) e sua chave.
- **Não funciona fora do macOS**: `xcrun notarytool` é ferramenta do Xcode.

### Segredos necessários

| Nome | Tipo | Origem |
| --- | --- | --- |
| `APPLE_API_KEY_P8` | **secret** | conteúdo do arquivo `AuthKey_<KEYID>.p8` (App Store Connect ▸ Users and Access ▸ Integrations ▸ Keys) |
| `APPLE_API_KEY_ID` | secret ou var | o `<KEYID>` do nome do arquivo (ex.: `YQ5PVAP29M`) |
| `APPLE_API_ISSUER` | var | **Issuer ID** (UUID) mostrado no topo da mesma tela do App Store Connect |
| `APPLE_IDENTITY` | secret | identidade de assinatura já usada hoje |

```sh
# O conteúdo do .p8 vai por STDIN, para não aparecer no histórico do shell:
gh secret set APPLE_API_KEY_P8 < /caminho/AuthKey_YQ5PVAP29M.p8
gh secret set APPLE_API_KEY_ID --body "YQ5PVAP29M"
gh variable set APPLE_API_ISSUER --body "<ISSUER-UUID>"
```

**A chave privada nunca é commitada.** O repositório guarda apenas o **caminho** (`APPLE_API_KEY`), e o CI
materializa o arquivo em `$RUNNER_TEMP` com `umask 077`, aplicando `::add-mask::` no conteúdo para que ele não
apareça no log. A publicação do artefato notarizado segue dependendo de credencial de release (ver
`docs/RELEASE_READINESS.md`).
