# DZ23 Agentic Mobile

Cliente Expo para acompanhar missões, visualizar timeline, aprovar/rejeitar passos protegidos, iniciar execução e consultar uma API agentic v1 autenticada.

A inbox consulta `GET /api/agent/v1/missions`, abre detalhe e timeline ao tocar numa missão e permite atualizar a lista. Lista e até 20 detalhes recentes são armazenados no AsyncStorage por servidor e organização; falha de rede lê o cache do mesmo escopo. O cache antigo de uma missão é migrado ao consultar um novo detalhe. Respostas HTTP de erro aparecem como erro, sem esconder uma falha do servidor com dados antigos. Na web, o token fica apenas em memória durante a sessão; no Android/iOS usa SecureStore.

## Desenvolvimento

```bash
npm install
npm run start
npm run typecheck
npm run test:inbox
npm run test:policy
```

No emulador ou dispositivo físico, informe em **Servidor** uma URL alcançável pelo celular, por exemplo `http://192.168.0.10:11434`. O cliente persiste a URL no AsyncStorage e o token Bearer no SecureStore nativo. Depois de carregar o token, consulta `/api/agent/v1/auth/session` e só habilita outbox/push autenticados após confirmar o `organization_id`. Cache de missão, fila offline e marca de push são namespaced por servidor e organização; uma sessão sem tenant confirmado não pode enfileirar uma mutação autenticada. O servidor continua sendo a autoridade de policy.

## Autenticação

Em modo multiusuário, forneça um token Bearer emitido pelo servidor. Para desenvolvimento local, o servidor pode habilitar explicitamente `OLLAMA_AGENT_AUTH_REQUIRED=true` e `OLLAMA_AGENT_AUTH_DEV=true`, emitir um token pelo endpoint de bootstrap e depois desabilitar o bootstrap de desenvolvimento. Não coloque tokens em `.env`, bundle, screenshots ou repositório.

## Android e iOS

Os identificadores de produção são `com.lmprado.dz23agentic`. Para builds EAS, configure primeiro a conta Expo/EAS e as credenciais de assinatura fora do repositório:

```bash
npm install -g eas-cli
eas login
eas build:configure
npm run build:android
npm run build:ios
npm run build:all
```

O arquivo `eas.json` possui perfis `development`, `preview` e `production`. O envio para lojas exige preencher o `ascAppId` do App Store Connect e configurar credenciais reais na conta EAS; esta etapa não é simulada pelo código.

O mobile não executa shell, browser, processos ou conectores diretamente. Ele solicita operações pela API, observa eventos e apresenta approvals. O outbox usa `Idempotency-Key`, `If-Match`, backoff, limite de tentativas e estado explícito de conflito; `401/403` não são tratados como simples retry e exigem nova autenticação/revisão. Polling periódico é usado como fallback compatível com Android/iOS; push notifications são registradas por organização quando o operador concede permissão. Builds físicos Android/iOS, entrega push remota, resolução de conflitos em rede real e publicação em lojas continuam dependentes de contas, dispositivos e credenciais do operador.

O lock mantém overrides transitivos mínimos para `image-size`, `postcss` e `uuid`, porque o `npm audit fix --force` sugeriria uma migração major de Expo/React Native. Verifique com `npm audit --omit=dev`; o estado publicado desta slice retorna zero vulnerabilidades de produção. A migração major continua separada e requer testes físicos antes de distribuição.
