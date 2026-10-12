package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// VARREDURA DA INVARIANTE DE DLP NA TRILHA (E-193).
//
// O teste pontual `TestServiceAgentAuditKeepsContactTextOut` cobre UMA ferramenta (`crm.contact.read`). A falha
// intermitente observada na CI (`main`, passo `CGO_ENABLED=1 go test ./...`) apontou para esse mesmo teste e NÃO foi
// reproduzida em seis condições medidas (10× isolado, 20× isolado sob CGO=1 no Linux, suíte do pacote `-count=3`, o
// pacote inteiro sob CGO=1, `-shuffle` com três seeds e o pacote sob carga paralela).
//
// Como não consigo re-executar o job (sem credencial de API do GitHub nesta sessão), o passo que dá para dar é
// atacar a INVARIANTE em vez do sintoma: se o vazamento existe, ele pode estar em OUTRA ferramenta que o teste
// pontual nunca exercitou. Esta varredura roda o mesmo turno com o mesmo segredo para CADA ferramenta declarada de
// CADA agente do catálogo e exige que a trilha não contenha o texto de terceiro.
//
// O que esta varredura NÃO prova: que a falha intermitente da CI era este caminho (se ela não aparecer aqui, o
// sintoma segue sem causa demonstrada), nem que a redação é completa para conteúdo que não seja este segredo.
func TestServiceAgentAuditNeverStoresContactTextForAnyTool(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"

	catalogo := ServiceAgentCatalog()
	if len(catalogo) == 0 {
		t.Fatal("o catálogo de agentes está vazio: a varredura não cobriria nada")
	}

	ferramentasExercitadas := 0
	for _, spec := range catalogo {
		for _, ferramenta := range spec.Tools {
			ferramenta := ferramenta
			nome := strings.ReplaceAll(spec.ID+"/"+ferramenta, " ", "_")
			t.Run(nome, func(t *testing.T) {
				runtime, store, conversationID, now := auditFixture(t)
				_, err := runtime.RunTurn(ServiceAgentTurnRequest{
					OrganizationID: "org-a",
					AgentID:        spec.ID,
					ConversationID: conversationID,
					Intent:         segredo,
					RequestedTools: []string{ferramenta},
				}, now)
				if err != nil {
					// A ferramenta pode recusar (falta de consentimento, argumentos, escopo). Isso não é falha desta
					// varredura: a invariante é sobre o que FICOU GRAVADO, não sobre a ferramenta ter sucesso.
					t.Logf("turno com %q terminou com erro (aceitável): %v", ferramenta, err)
				}

				arquivo := filepath.Join(store.Root(), sanitizeIDComponent("org-a")+".json")
				conteudo, lerErr := os.ReadFile(arquivo)
				if lerErr != nil {
					// Sem trilha gravada não há o que vazar.
					t.Logf("nenhuma trilha gravada para %q (%v)", ferramenta, lerErr)
					return
				}
				texto := string(conteudo)
				if strings.Contains(texto, "4111") || strings.Contains(texto, segredo) {
					t.Fatalf(
						"VAZAMENTO DE DLP: a ferramenta %q gravou o texto do contato na trilha de auditoria.\n"+
							"Trilha gravada:\n%s", ferramenta, texto)
				}
			})
			ferramentasExercitadas++
		}
	}

	if ferramentasExercitadas == 0 {
		t.Fatal("nenhuma ferramenta foi exercitada: a varredura não provou nada")
	}
	t.Logf("varredura de DLP cobriu %d ferramentas em %d agentes", ferramentasExercitadas, len(catalogo))
}
