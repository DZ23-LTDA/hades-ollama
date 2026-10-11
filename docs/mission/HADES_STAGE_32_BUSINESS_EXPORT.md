# Estágio 32 — exportação CSV do painel de negócio (E-136)

## 1. O limite que esta rodada fecha

Registrei nas rodadas anteriores que o BI não tinha **exportação**. E há um motivo específico para
tratar exportação com cuidado: **um arquivo exportado circula fora do produto** — é aberto numa
planilha, mandado por e-mail, colado num relatório. A tela não vai junto.

## 2. O que foi entregue

`Runtime.BusinessExportCSV` e a rota `GET /companies/:id/business-export`, que responde
`text/csv; charset=utf-8` com `Content-Disposition: attachment`.

Esquema **uniforme** em todas as linhas: `secao,chave,valor,observacao`.

| `secao` | Conteúdo |
| --- | --- |
| `meta` | organização, instante da consulta e `sandbox=true` |
| `indicador` | os 29 indicadores, cada linha com a **origem** na coluna de observação |
| `nao_medido` | **as lacunas, com motivo** |
| `serie` | `resumo` (retenção, fora do recorte, ausência de base) e uma linha por (janela, métrica) |
| `tendencia` | os deltas com `de`/`para`/`intervalo`; ou `sem_base` **com o motivo** |

## 3. Quatro decisões deliberadas

| Decisão | Por quê |
| --- | --- |
| **Esquema único** | cabeçalho diferente por bloco quebra qualquer leitor de planilha; a série vira uma linha por (janela, métrica) |
| **As lacunas vão DENTRO do arquivo** | fora do produto ninguém vê a tela: um número exportado sem as lacunas **parece completo**, que é exatamente o engano que o painel evita |
| **A origem vai em cada indicador** | o número precisa poder ser conferido longe da interface |
| **Escaping pela biblioteca** (`encoding/csv`) | montar CSV por concatenação de strings é a forma clássica de corromper relatório |

E uma garantia de efeito colateral: **exportar NÃO registra amostra** — o teste confere que a série
continua vazia depois do export. Exportar é leitura.

## 4. Verificação (3 testes no domínio + 1 na rota)

| Teste | O que prova |
| --- | --- |
| `TestBusinessExportIsValidCSVWithGapsSourcesAndSeries` | **todas** as linhas com 4 campos; `meta` com escopo e sandbox; indicadores reais (`pedidos=3`, `receita_cents=3500`, `violacoes_sla=1`) **cada um com origem**; a publicação externa declarada; **o mesmo número de lacunas do relatório**, cada uma com motivo; série só com `resumo` quando não há amostra e `sem_base` **com motivo** |
| `TestBusinessExportEscapesHostileValues` | valor com vírgula, **aspas**, ponto e vírgula e **quebra de linha** sobrevive intacto ao round-trip, e a quebra **não** vira linha nova |
| `TestBusinessExportIsScopedByOrganization` | outra organização exporta zeros e **não** contém o identificador da dona; a dona exporta os seus números; escopo ausente é erro; **exportar não cria amostra** |
| `TestBusinessExportRouteDownloadsValidCSV` | **200** com `Content-Type` de CSV e `Content-Disposition` de anexo; corpo parseável com 4 campos por linha; lacunas presentes; empresa inexistente **404** |

## 5. Dois achados durante a construção (registrados)

1. **`\r\n` não é round-trip fiel, e isso é a biblioteca padrão**: o leitor de `encoding/csv`
   normaliza CRLF para LF dentro de campo citado — comportamento **documentado**, não defeito do
   export. O teste passou a usar LF e a garantir o que importa (vírgula, aspas e quebra de linha
   sobrevivem), com o motivo escrito no próprio teste.
2. **Meu erro de nomenclatura**: o erro de escopo saiu como não exportado (`errBusinessExportScope`) e
   o mapeador HTTP não compilou. Renomeado para `ErrBusinessExportScope`, seguindo os irmãos
   `ErrBusinessStatusScopeRequired` e `ErrBusinessSnapshotScopeRequired`.

Também vale registrar um **falso alarme** que eu mesmo causei: rodei as suítes completas **sem** apontar
`APPDATA`/`LOCALAPPDATA` para `_tmp` e várias provas falharam com *"mkdir
C:\Users\...\AppData\Roaming\ollama-agent: Acesso negado"*. **Não era regressão**: era o meu comando
incompleto. Com o bloco de ambiente correto, `internal/agent` e `server` passam inteiros — e a distinção
importa, porque registrar isso como defeito do produto teria custado uma investigação inútil.

## 6. Limites declarados

- **Sem exportação em PDF nem planilha nativa** (`.xlsx`): o formato é CSV, que abre em qualquer
  planilha.
- **Sem escolha de período/colunas no export**: exporta o painel do agora e a série retida, completos.
- **Sem streaming**: o documento é montado em memória; com a retenção atual (96 amostras) o tamanho é
  de milhares de linhas, mas isso é um limite a declarar, não a esconder.
- **Sem BOM UTF-8**: alguns planilheiros em pt-BR podem exibir acentos errados ao abrir direto; abrir
  como UTF-8 resolve.
