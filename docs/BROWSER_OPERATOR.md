# Browser Operator — usar o seu Chrome real (opcional)

Por padrão, o Browser Operator do Hades navega num Chromium **isolado e
headless** (sem suas contas), com guarda anti-SSRF e **aprovação humana por
ação**. Isso é o modo seguro e recomendado.

Se você quiser que o agente opere **dentro do seu Chrome, com as suas sessões
logadas** (ex.: um site onde você já entrou), dá para anexar o Hades ao seu
Chrome via **CDP (Chrome DevTools Protocol)**, de forma controlada.

## Como ligar

1. Abra o Chrome com uma **porta de depuração** e um **perfil dedicado** (o
   Chrome moderno bloqueia depuração no perfil padrão, de propósito). Faça login
   nos sites que você quer usar **nesse** perfil:

   ```bat
   "C:\Program Files\Google\Chrome\Application\chrome.exe" ^
     --remote-debugging-port=9222 ^
     --user-data-dir="%LOCALAPPDATA%\Hades\chrome-agent-profile"
   ```

2. Aponte o Hades para esse debugger (antes de iniciar o runtime do agente):

   ```
   OLLAMA_AGENT_BROWSER_CDP_URL=http://127.0.0.1:9222
   ```

3. (Opcional) Para o modo de **lançar** um Chrome visível em vez de headless
   (quando NÃO usa CDP), defina `OLLAMA_AGENT_BROWSER_HEADLESS=0` e
   `OLLAMA_AGENT_BROWSER_EXECUTABLE` para o `chrome.exe`.

## Proteções que continuam valendo (importante)

- **Só loopback:** o Hades só anexa a um debugger em `127.0.0.1`. Nunca a um
  navegador remoto.
- **Perfil dedicado:** use `--user-data-dir` próprio, separado do seu perfil
  principal. Assim o agente **não** herda todas as suas contas — só as que você
  logar nesse perfil.
- **Anti-SSRF:** toda navegação/subrequest é validada; endereços não públicos
  são bloqueados (salvo o bypass de teste explícito).
- **Aprovação por ação:** o Browser Operator exige aprovação humana antes de
  cada ação consequente. Mantenha isso ligado.
- **Risco a entender:** um agente autônomo operando numa sessão logada pode ser
  alvo de *prompt injection* por uma página maliciosa. Use perfil dedicado,
  aprove cada ação e não logue contas sensíveis (banco, e-mail principal) nesse
  perfil.
