# Guia rápido do Hades (para quem não é técnico)

Este guia explica, em passos simples, como instalar e começar a usar o **Hades**
no Windows — sem precisar saber programar. Se algo não funcionar como descrito
aqui, isso é um problema do guia ou do programa: anote o que apareceu na tela e
avise a equipe.

## O que é o Hades

O Hades é um assistente de IA que roda **no seu próprio computador**. Por padrão,
os modelos de IA funcionam localmente — suas conversas não saem da sua máquina a
menos que **você** ative uma opção de nuvem. É pensado para conversar, pesquisar,
organizar projetos e automatizar tarefas.

## 1. Instalar

1. Peça à DZ23-LTDA o arquivo de instalação **`HadesSetup.exe`** (é o instalador
   oficial do projeto).
2. Dê dois cliques no `HadesSetup.exe`.
3. **Aviso azul do Windows (SmartScreen):** por enquanto o instalador ainda **não
   é assinado digitalmente**, então o Windows pode mostrar uma tela azul dizendo
   "O Windows protegeu o seu computador". Isso é esperado nesta fase. Para
   continuar: clique em **"Mais informações"** e depois em **"Executar assim
   mesmo"**. Só faça isso com o arquivo que veio da DZ23-LTDA; não baixe o Hades
   de outros lugares.
4. Siga o assistente de instalação (botão **Avançar/Instalar**). Ao final, o Hades
   aparece no **Menu Iniciar** com o nome **"Hades"**.

> Dica: o instalador também cria um atalho **"Hades - Configure APIs"**. Você só
> precisa dele se for conectar serviços externos (nuvem). Para uso local simples,
> pode ignorar.

## 2. Abrir e configurar na primeira vez

1. Abra o **Hades** pelo Menu Iniciar. Uma janela/página vai abrir com a tela de
   boas-vindas.
2. Na primeira vez, o Hades sugere **baixar um modelo pequeno** para funcionar
   offline (a recomendação inicial é um modelo leve, que baixa rápido e roda na
   maioria dos computadores). Clique no botão de **baixar o modelo recomendado** e
   aguarde a barra de progresso terminar.
3. Pronto: quando o modelo terminar de baixar, você já pode **conversar**.

## 3. Usar no dia a dia

- **Conversar:** digite sua mensagem na caixa de texto e envie. O modelo local
  responde na sua máquina.
- **Anexar arquivos:** você pode anexar documentos para o assistente usar como
  contexto. Ao **importar um projeto** (pasta ou ZIP), o Hades mostra **quantos
  arquivos foram indexados e quantos foram ignorados** (por exemplo, imagens ou
  arquivos muito grandes não são lidos). Se o seu documento importante aparecer
  como "ignorado", é por isso que o assistente pode não encontrá-lo.
- **Pesquisar:** para a pesquisa buscar fontes a partir de uma pergunta é preciso
  um provedor de busca configurado; sem ele, informe os endereços (URLs) que
  quer que o Hades leia. O Hades avisa de forma clara quando não há provedor.

## 4. Privacidade (importante)

- Por padrão, **tudo roda localmente**. Os modelos de nuvem só são usados se
  **você** ativar explicitamente a opção de nuvem na configuração.
- Ao ativar a nuvem, suas mensagens passam a ser enviadas ao provedor escolhido.
  O Hades confirma se a preferência de nuvem foi realmente salva antes de seguir.
- Nunca guarde senhas ou chaves dentro de conversas.

## 5. Se algo der errado - os 6 problemas mais comuns

Cada item traz a **mensagem real** que o programa mostra, para reconhecer o problema sem
adivinhar. Estas mensagens foram medidas numa execucao real, nao inventadas.

### 5.1 A tela azul do SmartScreen apareceu na instalacao
**O que aparece:** "O Windows protegeu o computador".
**Por que:** o instalador **ainda nao e' assinado digitalmente** (ver item 6).
**O que fazer:** Mais informacoes -> Executar assim mesmo -- **so** com o ficheiro baixado da
pagina oficial do projeto. Se veio de outro sitio, nao execute.

### 5.2 "O servidor esta' offline" ou a interface nao carrega
**O que aparece:** a interface abre mas nao responde, ou aviso de servidor offline.
**Por que (a causa mais comum):** **outra aplicacao ja' esta' a usar a porta 11434** -- por exemplo
uma instancia anterior do Hades que ficou aberta.
**O que fazer:** feche todas as janelas do Hades; no Gerenciador de Tarefas encerre qualquer processo
`ollama` ou `hades` que tenha ficado; abra de novo. Se persistir, reinicie o computador.

### 5.3 "Acesso negado" ao criar projeto ou missao
**O que aparece (mensagem real):**
`create agent store: mkdir C:\Users\<voce>\AppData\Roaming\ollama-agent\data\missions: Acesso negado`
**Por que:** a pasta de dados do agente esta' protegida (perfil corporativo, antivirus ou permissoes
restritas) e o Hades nao consegue escrever la'.
**O que fazer:** aponte os dados para uma pasta sua, definindo **antes de abrir**:

```
OLLAMA_AGENT_STORE=C:\Users\<voce>\Hades\dados
OLLAMA_AGENT_ROOT=C:\Users\<voce>\Hades\projetos
```

Depois abra o Hades. Nada e' apagado: a pasta antiga fica como estava.

### 5.4 "Salvei as configuracoes e nao ficou guardado"
**O que aparece:** a alteracao volta ao valor anterior, ou a mensagem
`Este endpoint e' atendido pelo aplicativo desktop do Hades; o servidor local nao persiste configuracao`
(o servidor responde **501**).
**Por que:** o **servidor local** (porta 11434) nao grava configuracoes -- quem grava e' o
**aplicativo desktop**. E' uma decisao, nao uma avaria.
**O que fazer:** faca a alteracao **pelo aplicativo desktop**, nao por chamada direta ao servidor.

### 5.5 "O Hades recusou executar e menciona o pare"
**O que aparece:** a execucao e' recusada e o estado cita o pare de emergencia.
**Por que:** o **pare de emergencia** foi acionado -- ele existe para parar **tudo**, e ninguem o
desarma por acidente.
**O que fazer:** **nao** tente contornar. Peca a quem administra a instalacao para rearmar, com
justificacao. O rearme e' registado e exige autoridade (`HADES_PARE_AUTORIDADE`); sem essa
autoridade configurada, o proprio administrador recebe "a autoridade nao esta' configurada".

### 5.6 O download do modelo travou
**O que aparece:** a barra de progresso para.
**Por que:** conexao instavel ou espaco em disco insuficiente para o modelo.
**O que fazer:** confira a internet e o espaco livre, e baixe de novo pelo mesmo botao -- o download
recomeca sem duplicar o que ja' veio.

## 6. O que ainda está em preparação (honestidade)

Para você saber o que esperar:

- O instalador **ainda não é assinado** (por isso o aviso do SmartScreen). A
  assinatura digital está planejada.
- Recursos de **nuvem, integrações externas e publicação** exigem configuração e
  credenciais próprias; sem isso, aparecem como "não configurado" — não é erro.
- O foco atual e testado é o **uso local** (conversar e organizar projetos na sua
  máquina).

Para detalhes técnicos (build, API, arquitetura), veja
[`CLASS_A_PLUS_GUIDE.md`](CLASS_A_PLUS_GUIDE.md).
