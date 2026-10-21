# Ponte `[Components]` — estado real e o que falta

Este ficheiro explica a ligação entre o catálogo de pacotes do HADES
(`internal/agent/install_profile.go`) e o instalador Windows (`app/ollama.iss`).

## O que existe hoje

| peça | estado |
|---|---|
| catálogo de 9 perfis e 18 pacotes | `internal/agent/install_profile.go` |
| gerador de `[Types]` + `[Components]` | `internal/agent/install_iss.go` |
| testes do gerador | `internal/agent/install_iss_test.go` — 13 testes |
| texto gerado | `app/components.generated.iss` (32 linhas) |
| `app/ollama.iss` real | **NÃO inclui** o texto gerado |

## Porque o `.iss` real ainda não inclui

O `app/ollama.iss` tem **10 linhas `Source:` sem atributo `Components:`** (linhas ~106 a ~125).
Cada uma copia um ficheiro ou diretório do payload inteiro.

Acrescentar `[Components]` ao `.iss` **sem** atribuir `Components:` a cada `Source:` produziria
uma UI de selecção que não controla nada: o utilizador desmarcaria «Containers e Buildx» e o
instalador copiaria os mesmos ficheiros na mesma.

Ou seja: o que falta **não é o texto**, é a **divisão do payload**. É trabalho de build, não de
código Go:

1. separar `dist\windows-amd64\` em subdiretórios por pacote;
2. acrescentar `Components: <id>` a cada `Source:` do `.iss`;
3. gerar o `.iss` final a partir de `components.generated.iss` (ou incluir com `#include`);
4. construir o instalador e provar que `BASIC` instala menos bytes que `FULL_OFFLINE`.

**Enquanto o passo 2 não acontecer, incluir o `[Components]` seria uma mentira visível na UI:**
checkbox que não muda o que é copiado.

⇒ *Um instalador que oferece escolhas e copia o mesmo não é modular: é um questionário.*

## As duas regras que o gerador já garante

1. **`hades-core` não é componente.** Vai no `[Files]` sem condição.
   *Um núcleo opcional permitiria instalar um HADES sem Mission Control — uma instalação
   avariada, não uma instalação mínima.*

2. **Componente que exige aprovação sai `Flags: unchecked`.**
   Nove pacotes têm impacto (elevação, serviço, reinício, porta, driver, EULA, ambiente global,
   conta externa) e nenhum deles aparece escolhido antes de o utilizador ler a linha.
   *Um componente que cria serviço e abre porta não pode aparecer escolhido por omissão.*

## Como regenerar

```
go test ./internal/agent/ -run TestZzEmitirTexto
```

(ou, quando a CLI existir, `hades install components`). O texto é determinístico: duas execuções
produzem bytes iguais.

## Veredito

`PARTIAL_IMPLEMENTED_NOT_E2E_VERIFIED` — gerador e testes verdes, instalador real ainda monolítico.
