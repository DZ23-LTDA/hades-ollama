# Auditoria de compatibilidade Windows — WIN-1

**Data:** 2026-09-30 18:03 -03
**Branch:** `recovery/ollama-full-snapshot`

## Resultado

O backend agora compila sem CGO e em cross-build Windows completo:

```text
CGO_ENABLED=0 go build ./...                         PASS
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./... PASS
```

O build nativo Linux com CGO e os testes de `internal/agent` e `server` também passaram.

## Correções

Os caminhos nativos MLX/xgrammar e o webview foram isolados por build tags `cgo`. Fallbacks `!cgo` preservam os contratos de compilação e retornam estados honestos de indisponibilidade; não fingem executar GPU, MLX ou uma janela nativa. O wrapper desktop sem CGO só é incluído nas plataformas desktop suportadas, evitando que o pacote `app/store` seja incluído indevidamente em Linux headless. A execução Git removeu a dependência de `os.DevNull` e usa configuração global/sistema vazia, compatível com Git for Windows.

O workflow `class-a-plus-integrity` recebeu o job `platform-builds`, que executa ambos os comandos acima em cada push relevante.

## Limitações

A evidência é de compilação cruzada no sandbox. Ainda não é evidência de execução em um host Windows físico, DACL nativo, MLX/WebView nativos, instalador assinado ou smoke end-to-end do aplicativo Windows. O CI remoto é o gate final desta alteração.
