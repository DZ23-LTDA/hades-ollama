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

## 5. Se algo der errado

- **"O servidor está offline" / não responde:** feche e abra o Hades novamente.
  Ele tenta se reconectar sozinho; se continuar, reinicie o computador.
- **O download do modelo travou:** verifique sua conexão com a internet e tente
  baixar de novo pelo mesmo botão.
- **A tela azul do SmartScreen voltou:** repita o passo 1.3 (Mais informações →
  Executar assim mesmo), sempre com o arquivo oficial.

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
