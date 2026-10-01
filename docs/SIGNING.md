# Code signing do instalador Windows

## Provider e modelo

O instalador Windows do Hades/Ollama Full está preparado para assinatura em nuvem com **SSL.com eSigner OV**. O fluxo usa a action oficial `sslcom/esigner-codesign` e não depende de token USB no runner.

A assinatura é **fail-safe e honesta**:

- com os quatro secrets configurados, o `.exe` é assinado antes do checksum;
- sem qualquer secret obrigatório, a etapa de assinatura é pulada e o instalador é publicado como **UNSIGNED**;
- o workflow não imprime valores de credenciais nos logs;
- o SHA-256 sempre é calculado sobre o arquivo final que será entregue.

## Secrets do GitHub

Adicione em **Settings → Secrets and variables → Actions → New repository secret** os nomes exatos abaixo. Os valores são fornecidos pelo proprietário da conta SSL.com e **não devem ser commitados**:

| Secret | Uso |
| --- | --- |
| `SSL_COM_USERNAME` | Usuário da conta SSL.com eSigner |
| `SSL_COM_PASSWORD` | Senha da conta SSL.com eSigner |
| `SSL_COM_CREDENTIAL_ID` | ID da credencial/certificado OV usado para assinar |
| `SSL_COM_TOTP_SECRET` | Segredo OAuth/TOTP usado pela automação eSigner |

A action só é executada quando os quatro valores estão presentes. Configuração parcial permanece unsigned para não produzir um estado ambíguo.

## Como disparar

O workflow é `.github/workflows/dz23-windows-installer.yaml` e pode ser iniciado por:

1. **Actions → dz23-windows-installer → Run workflow**; ou
2. push de uma tag `v*` (por exemplo, `v1.2.3`), conforme os gatilhos definidos no workflow.

O artefato terá um dos nomes:

- `OllamaClasseAPlusSetup-windows-amd64-signed`, quando a assinatura SSL.com foi executada;
- `OllamaClasseAPlusSetup-windows-amd64-unsigned`, quando os secrets não existem ou estão incompletos.

O resumo do job também informa explicitamente `SIGNED` ou `UNSIGNED`. O arquivo `.sha256` acompanha o instalador e é calculado somente depois da etapa de assinatura.

## Verificação sem secrets

A validação padrão do CI não exige credenciais de assinatura. Sem os quatro secrets, o job deve:

1. construir `dist/OllamaClasseAPlusSetup.exe`;
2. pular a action SSL.com;
3. mostrar `INSTALLER_SIGNING=UNSIGNED`;
4. gerar o checksum do instalador unsigned;
5. publicar o artefato `...-unsigned` sem falhar.

A lógica de ativação pode ser revisada no step `Detect SSL.com signing configuration`, sem inserir valores reais. A execução assinada só deve ser considerada validada depois de um run com credenciais reais e de uma verificação externa da assinatura no `.exe`.

## Plataformas

- **Windows:** SSL.com eSigner OV está preparado no CI deste workflow.
- **macOS:** distribuição assinada exige Apple Developer Program (US$ 99/ano), certificado e notarização; permanece como próximo passo separado.
- **Linux:** os pacotes continuam unsigned por padrão; assinatura de pacote Linux será tratada em uma etapa própria.

Nunca marque um artefato como assinado apenas porque os secrets existem: a confirmação final depende do sucesso real da action e da verificação da assinatura produzida.
