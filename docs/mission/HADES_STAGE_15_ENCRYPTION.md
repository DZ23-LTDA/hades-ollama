# Estágio 15 — criptografia da memória em repouso

## 1. O princípio: nada de criptografia caseira

O primitivo é o **mesmo já usado para credenciais de conector** (`encryptCredential`):
**AES-256-GCM**, chave derivada por SHA-256 de `OLLAMA_AGENT_CREDENTIAL_KEY` (mínimo 16
caracteres), nonce aleatório por registro, base64. Inventar um segundo esquema criaria
**duas verdades sobre a mesma chave** — e é onde projetos costumam errar feio.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| É **opt-in** | exigir chave de imediato quebraria instalações existentes |
| Com chave, o **texto claro não vai ao disco**: `Content` sai vazio e o cifrado fica em `encrypted_content` | é o que "em repouso" significa |
| A leitura é **fail-closed** | registro cifrado sem chave **falha com erro explícito**; devolver vazio em silêncio seria **perda de dado disfarçada de memória vazia** |
| **Um único caminho de escrita** (`writeMemoriesFile`) | eram **cinco** pontos gravando o mesmo arquivo; proteger quatro deixaria um salvando texto claro |
| `MemoryEncryption()` diz a **verdade** quando não há chave | o produto não pode sugerir que protege o que não protege |

## 3. O defeito que MEU PRÓPRIO TESTE encontrou

O teste que verifica **todos** os caminhos de escrita falhou no primeiro deles:
**atualizar** deixava o texto claro no disco. Causa: o caminho de versionamento grava o
**conteúdo anterior** em `Versions[].Content`, e eu havia cifrado apenas o registro
principal. O **histórico** é texto do usuário e vazava em claro.

Corrigido: `MemoryVersion` também ganhou `encrypted_content`, e o funil de disco cifra e
decifra **as revisões**. Foi o segundo caso nesta semana em que o teste que mede o
efeito — e não a releitura do código — encontrou o irmão esquecido.

## 4. Verificação

- Com chave: o arquivo **não contém** o texto claro e contém `encrypted_content`; o
  store reiniciado devolve o conteúdo **idêntico** (round-trip).
- Sem chave, ao ler memória cifrada: **falha** com erro que nomeia a chave ausente.
- Sem chave configurada: o **comportamento anterior continua** (texto claro) e o store
  abre normalmente — opt-in não quebra instalação existente.
- **Todos os caminhos** cifram: adicionar, atualizar (versionar), remover e aplicar
  **retenção**; após a retenção o conteúdo ainda é legível pelo store.

`go build ./...` exit 0 · suítes apenas com as **falhas ambientais conhecidas** ·
`golangci-lint` **0 issues** · `gofmt` limpo.

## 5. Estado do estágio 15

| Atividade | Situação |
| --- | --- |
| Isolamento · exportação/exclusão · contexto do modelo · troca de modelo · reinício · retomada | cobertos |
| Deduplicação · versionamento | cobertos (E-092, E-093) |
| **Criptografia em repouso** | **coberta nesta rodada** |
| Resumos · escopos missão/agente/equipe | **ainda ausentes** |

Continua **`IN_PROGRESS`**: faltam duas atividades.
