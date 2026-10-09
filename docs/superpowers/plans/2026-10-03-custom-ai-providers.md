# Providers de IA locais e compatíveis — Plano de implementação

> **Para agentes de implementação:** executar as etapas abaixo em sequência e conferir cada gate antes de integrar.

**Objetivo:** Permitir cadastrar pela UI desktop endpoints locais ou OpenAI-compatible, com modelos configurados e chave opcional protegida, para uso no chat/roteador.

**Arquitetura:** Estender a rota local `/api/v1/providers` para validar e persistir um provider OpenAI-compatible no arquivo configurado por `OLLAMA_DZ23_CONFIG`; armazenar chave somente pelo pacote `app/secrets`. Adicionar formulário acessível na tela Provedores de IA, atualizar listagem e reinicialização via mecanismo existente. Não alterar nem alegar modificar o aplicativo Manus proprietário.

**Stack:** Go, React/TypeScript, Vitest, testes `net/http/httptest`.

**Spec:** Pedido do usuário para habilitar IA local e APIs gratuitas no desktop Hades/Ollama Full; implementação usa endpoint compatível e não promete gratuidade/quota de terceiros.

## Restrições globais

- Continuar em `recovery/ollama-full-snapshot`; preservar arquivos locais não rastreados; sem force-push ou alteração de `main`.
- Reutilizar autenticação da UI, validadores `multillm.LoadBytes`, políticas e cofre de credenciais já existentes.
- Aceitar HTTP apenas para loopback local; endpoints externos devem ser HTTPS; não guardar chave no JSON, logs ou resposta.
- Não classificar um provider como gratuito por inferência: preço, quota e compatibilidade são definidos pelo serviço escolhido.

## Foco da revisão

- URL com userinfo, esquema perigoso, HTTP externo ou IP privado não autorizado deve ser rejeitada sem escrita.
- Nome duplicado, inválido ou que produza variável de chave ambígua deve ser rejeitado.
- Corpo inválido, tamanho excessivo, IDs duplicados/vazios ou lista sem modelos deve falhar sem alterar configuração.
- Chave só pode ser enviada pelo cliente na criação; não pode aparecer na listagem, arquivo de configuração ou logs.
- Provider local sem chave deve permanecer selecionável; provider com chave deve só ficar disponível depois do cofre estar configurado.

---

### Tarefa 1: API Go de criação segura

**Arquivos:** `app/ui/providers.go`, `app/ui/ui.go`, `app/ui/providers_test.go`.

- [ ] Criar testes para sucesso (local sem chave e remoto HTTPS com chave), duplicata, payload/URL inválidos e ausência de vazamento da chave.
- [ ] Implementar `POST /api/v1/providers`, validar request com limite, nome, base URL, até 100 IDs de modelo únicos; aplicar flags locais somente para loopback e passar a configuração por `multillm.LoadBytes`.
- [ ] Persistir provider com `writeProviderConfig`; se fornecida, salvar a chave via `secrets.Save` em variável gerada deterministicamente e nunca no JSON; reiniciar via `restartForProviders`.
- [ ] Rodar `go test ./app/ui -run 'Provider' -count=1` e `go test ./internal/multillm -count=1`.

### Tarefa 2: Formulário da UI e testes

**Arquivos:** `app/ui/app/src/components/ProvidersPage.tsx`, `app/ui/app/src/lib/providers.ts`, testes novos/atualizados em `app/ui/app/src/components` e `lib`.

- [ ] Adicionar contrato de criação e formulário em pt-BR com nome, endpoint, IDs de modelo, chave opcional, estado de envio/erro/sucesso e cancelamento.
- [ ] Explicar exemplo de endpoint local e que gratuidade/quota depende do provider; não incluir integração hard-coded a provedor externo.
- [ ] Testar submissão, validação de campos, estado de espera/erro e confirmação de sucesso; manter comportamento das configurações existentes.
- [ ] Rodar testes Vitest focalizados, lint e build da UI.

### Tarefa 3: revisão e evidência

- [ ] Revisar diff, `git diff --check`, rodar gates aplicáveis e confirmar que as duas imagens locais não rastreadas permanecem intocadas.
- [ ] Atualizar a matriz e o checkpoint de missão com escopo/evidência; registrar limitações de smoke externo sem credenciais.
- [ ] Não alegar alteração do Manus Desktop proprietário, paridade integral, API gratuita garantida ou smoke de provider remoto.
