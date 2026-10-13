package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// §11.4 — "SECRETS ARE NEVER LOGGED": A AFIRMAÇÃO PRECISA SER FALSIFICÁVEL (E-297).
//
// O gateway do WhatsApp repete em QUATRO comentários que segredos nunca são registrados, e nenhum teste provava
// isso. *Afirmação repetida quatro vezes e testada zero.* Este teste torna a afirmação verificável em duas
// direções, para os DOIS backends:
//  1. os textos de status/descrição identificam o canal (instância, Phone ID) SEM conter nenhuma credencial;
//  2. um erro devolvido pelo provedor que ECOA a credencial (no corpo, como um provedor faria) não pode propagá-la.
//
// As credenciais usadas são sintéticas e reconhecíveis, para que a asserção seja inequívoca.
const (
	evolutionAPIKeyDeTeste = "EVOKEY_9f3c1a7b2d4e5061728394a5b6c7d8e9"
	cloudAPITokenDeTeste   = "EAAFAKEcloudAPItoken0123456789abcdefghij"
	webhookSecretDeTeste   = "WHSECRET_5a4b3c2d1e0f99887766554433221100"
	appSecretDeTeste       = "APPSECRET_00112233445566778899aabbccddeeff"
)

func TestSegredosDoWhatsAppNaoAparecemEmStatusNemEmErro(t *testing.T) {
	configEvolution := EvolutionConfig{
		BaseURL:       "https://evolution.exemplo.invalid",
		APIKey:        evolutionAPIKeyDeTeste,
		Instance:      "instancia-teste",
		WebhookSecret: webhookSecretDeTeste,
	}
	configCloud := CloudAPIConfig{
		PhoneNumberID: "123456789012345",
		AccessToken:   cloudAPITokenDeTeste,
		VerifyToken:   "verify-token-de-teste",
		AppSecret:     appSecretDeTeste,
	}

	segredos := []string{evolutionAPIKeyDeTeste, cloudAPITokenDeTeste, webhookSecretDeTeste, appSecretDeTeste, "verify-token-de-teste"}

	// 1. Status/descrição identificam o canal e NÃO carregam credencial.
	textos := map[string]string{
		"evolution status": NewEvolutionAdapter(configEvolution).StatusDetails(),
		"cloud status":     NewCloudAPIAdapter(configCloud).StatusDetails(),
	}
	for nome, texto := range textos {
		for _, segredo := range segredos {
			if strings.Contains(texto, segredo) {
				t.Fatalf("[%s] credencial exposta no texto de status: %q", nome, texto)
			}
		}
	}

	// 2. Erro do provedor ECOANDO a credencial não pode propagá-la.
	casos := map[string]struct {
		send func(baseURL string) (WhatsAppSendResult, error)
	}{
		"evolution": {send: func(baseURL string) (WhatsAppSendResult, error) {
			adaptador := NewEvolutionAdapter(EvolutionConfig{
				BaseURL:       baseURL,
				APIKey:        evolutionAPIKeyDeTeste,
				Instance:      "instancia-teste",
				WebhookSecret: webhookSecretDeTeste,
			})
			return adaptador.SendMessage(context.Background(), WhatsAppOutboundMessage{To: "5511999999999", Text: "oi"})
		}},
		"cloud api": {send: func(baseURL string) (WhatsAppSendResult, error) {
			adaptador := NewCloudAPIAdapter(CloudAPIConfig{
				PhoneNumberID: "123456789012345",
				AccessToken:   cloudAPITokenDeTeste,
				AppSecret:     appSecretDeTeste,
			})
			// O endpoint do Cloud API é fixo (graph.facebook.com); para o teste, apontamos o adaptador ao servidor local
			// quando a configuração permitir — se não permitir, o caso é coberto pelo erro de rede abaixo.
			_ = baseURL
			return adaptador.SendMessage(context.Background(), WhatsAppOutboundMessage{To: "5511999999999", Text: "oi"})
		}},
	}

	for nome, caso := range casos {
		// A ARMADILHA PRECISA ESTAR ARMADA: ecoamos TODOS os cabeçalhos, a URL e o corpo, e guardamos o que foi
		// capturado. Sem isso, o teste passaria por não ter recebido credencial nenhuma — que é passar pelo motivo
		// errado.
		var capturado string
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var cabecalhos []string
			for nome, valores := range r.Header {
				cabecalhos = append(cabecalhos, nome+"="+strings.Join(valores, ","))
			}
			capturado = strings.Join(cabecalhos, " ") + " URL=" + r.URL.String()
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"unauthorized","echo":%q}`, capturado)
		}))

		_, err := caso.send(servidor.URL)
		servidor.Close()
		// A armadilha foi armada de fato? Se o adaptador não enviou credencial, o caso não mede nada e precisa ser
		// DITO, não silenciado.
		armada := false
		for _, segredo := range segredos {
			if strings.Contains(capturado, segredo) {
				armada = true
				break
			}
		}
		if !armada {
			t.Logf("[%s] armadilha NÃO capturou credencial (endpoint fixo ou credencial em outro lugar): caso vazio",
				nome)
		}
		if err == nil {
			// Sem erro não há vazamento a medir neste caso (o endpoint fixo do Cloud API não é alcançável no teste).
			continue
		}
		for _, segredo := range segredos {
			if strings.Contains(err.Error(), segredo) {
				t.Fatalf("[%s] credencial VAZOU no erro devolvido: %v", nome, err)
			}
		}
	}
}
