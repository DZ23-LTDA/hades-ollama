# Estágio 23 — Company Factory: empresa em sandbox com artefatos reais

## 1. O que faltava

`CompanyStore.CreateRequest` criava o **registro** da empresa. O estágio pede **artefatos**
e uma operação inicial verificável. Implementei o lançamento que **materializa** o que a
empresa precisa para começar a operar, e um **checklist honesto**.

## 2. Decisões deliberadas

| Decisão | Motivo |
| --- | --- |
| **A FÁBRICA NÃO INVENTA CLIENTE** | criar contatos fictícios no CRM seria exatamente os **dados falsos** que o estágio proíbe. O CRM fica **vazio**, e há teste travando isso — o próprio relatório declara `customers_created = 0` |
| **PREÇO DERIVADO DE CUSTO E MARGEM, com a conta à vista** | `preço = custo × (1 + margem/100)`, arredondado **uma única vez**. Margem **≥ 100%** é recusada (não há preço finito frente à margem) e custo negativo é erro |
| **O QUE DEPENDE DE AUTORIDADE É DECLARADO BLOQUEADO** | CNPJ, contrato social, contabilidade e domínio próprio aparecem como `BLOQUEADO_BY_EXTERNAL_DEPENDENCY`, **com o procedimento exato** de desbloqueio. A fábrica **não emite** registro legal nem dado fiscal |
| **ARTEFATOS SÃO ARQUIVOS REAIS** | plano, marca, precificação e site são gravados em disco (escrita **atômica**: temporário + rename), e o relatório informa **caminho e tamanho medido** — artefato que só existe na memória não é entregável |
| **RELATÓRIO RECONSTRUÍVEL DO ESTADO REAL** | `GET /company/factory/report/:id` remonta o relatório a partir da empresa, do catálogo e dos **arquivos em disco**, em vez de guardar uma cópia que poderia divergir |

## 3. O que a fábrica produz

| Artefato | Conteúdo |
| --- | --- |
| `plano-inicial.md` | missão, posicionamento, modelo, público, oferta, moeda e a lista de itens que dependem de terceiros |
| `marca.md` | tagline, paleta declarada pelo operador e personalidade a partir do posicionamento |
| `precificacao.md` | tabela com **custo, margem, preço e estoque** de cada produto, o **valor de tabela** do estoque inicial e a **fórmula** usada; declara o que **não** inclui (impostos, frete) |
| `index.html` | página inicial estática com os **dados reais** da empresa e os preços do catálogo |

E no comércio: **catálogo com preço derivado** e **entrada de estoque** como movimento no
livro (auditável).

## 4. O checklist

**Feito**: empresa registrada no Hades · catálogo com preço derivado · marca, site e plano
gravados · (com evidência em cada item).

**Bloqueado por dependência externa**, com procedimento: CNPJ/registro legal · contrato
social · contabilidade e obrigações fiscais · domínio com certificado.

**Não feito, por decisão**: clientes cadastrados — *"a fábrica não inventa clientes; o CRM
fica vazio até existir contato real"*.

## 5. Verificação

- **3 testes de motor**: artefatos **existem em disco** com tamanho real e conteúdo
  conferido (o site contém os dados da empresa e o **preço real**; a precificação declara a
  fórmula) · preço derivado conferido (`3500×1,60 = 5600`, `5200×1,50 = 7800`) e valor de
  tabela (`95000`) · **CRM vazio** depois do lançamento · checklist com itens **feitos com
  evidência** e **bloqueados com procedimento** · margem de 100%, custo negativo e
  organização ausente recusados · **relançar não duplica** o catálogo (SKU é a chave) ·
  relatório de outra organização **recusado**.
- **1 teste de rota**: `201` com quatro artefatos existentes, relatório reconstruído `200`,
  empresa inexistente `404`, margem inválida `400` e corpo inválido `400`.
- Relatório tem `customers_created = 0` e o CRM é conferido **vazio**.
- `go build ./...` exit 0 · `go vet` exit 0 · `golangci-lint` **0 issues** (corrigidos um
  `staticcheck` de laço e um `wastedassign` de verdade, além de reescrever o falso positivo
  do `misspell`) · suítes apenas com as **falhas ambientais conhecidas**.

## 6. Limites declarados

1. **Nada legal é emitido**: sem CNPJ, contrato, contabilidade ou dado fiscal — depende de
   dados reais, autoridade e terceiros.
2. **Sem site publicado**: o artefato é HTML local; publicar exige domínio e hospedagem.
3. **Sem campanhas nem redes sociais**: dependem de conta autorizada.
4. **Sem clientes, pedidos ou receita iniciais**: a fábrica prepara a operação; os dados
   vêm do uso real.
5. **Paleta e tagline são declaradas pelo operador**: a fábrica não inventa identidade
   visual "ótima" sozinha.
