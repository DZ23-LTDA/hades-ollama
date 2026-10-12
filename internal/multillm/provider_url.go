package multillm

import (
	"net/url"
	"strings"
)

// JoinProviderURL compõe a URL final a partir do BaseURL declarado e da rota canônica do protocolo.
//
// REGRA: a rota canônica começa pelo segmento de versão (`/v1`, `/v1beta`, ...). Quando o CAMINHO do
// BaseURL já declara uma versão — `.../v1` (Groq, DeepSeek, Mistral, OpenRouter, xAI) ou
// `.../v1beta/openai` (camada de compatibilidade do Google) — o segmento repetido é removido.
// Sem isso o provedor recebe `/v1/v1/...` ou `/v1beta/openai/v1/...` e responde 404 de rota.
//
// Por que virou função: a regra existia só dentro do gateway. O instrumento opt-in de inferência real
// montava a URL por conta própria, divergiu e produziu um 404 que parecia defeito do produto. Com a
// regra em um único lugar, instrumento e produto não podem divergir — e há teste da URL FINAL de
// cada preset, não de um caso escolhido à mão.
func JoinProviderURL(baseURL, route string) string {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		// Base inválida: concatenação crua mantém o comportamento observável e deixa o erro
		// aparecer na chamada, onde há contexto para diagnosticar.
		return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(route, "/")
	}
	return joinProviderPath(base, route).String()
}

func joinProviderPath(base *url.URL, route string) *url.URL {
	copiada := *base
	caminho := route
	if versao, ok := versaoDaRota(caminho); ok && caminhoBaseJaTemVersao(copiada.Path) {
		caminho = strings.TrimPrefix(caminho, versao)
	}
	copiada.Path = strings.TrimRight(copiada.Path, "/") + "/" + strings.TrimLeft(caminho, "/")
	return &copiada
}

// versaoDaRota devolve o segmento de versão inicial de uma rota canônica ("/v1", "/v1beta", "/v2").
func versaoDaRota(caminho string) (string, bool) {
	if !strings.HasPrefix(caminho, "/v") {
		return "", false
	}
	fim := 2
	for fim < len(caminho) && caminho[fim] >= '0' && caminho[fim] <= '9' {
		fim++
	}
	if fim == 2 {
		return "", false
	}
	for fim < len(caminho) && ((caminho[fim] >= 'a' && caminho[fim] <= 'z') || (caminho[fim] >= 'A' && caminho[fim] <= 'Z')) {
		fim++
	}
	if fim < len(caminho) && caminho[fim] != '/' {
		return "", false
	}
	if fim == 2 {
		return "", false
	}
	return caminho[:fim], true
}

// caminhoBaseJaTemVersao informa se o caminho do BaseURL declara uma versão em qualquer segmento.
func caminhoBaseJaTemVersao(caminhoBase string) bool {
	for _, segmento := range strings.Split(caminhoBase, "/") {
		if _, ok := versaoDaRota("/" + segmento); ok {
			return true
		}
	}
	return false
}
