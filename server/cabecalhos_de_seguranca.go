package server

import (
	"github.com/gin-gonic/gin"
)

// PoliticaDeSegurancaDeConteudo e' a CSP que o servidor envia.
//
// O QUE ESTA POLITICA E' -- E O QUE ELA DELIBERADAMENTE NAO E'
//
// Tres directivas, e so' tres:
//
//	object-src 'none'      -- nao se carregam plugins/embeds (Flash, Java, PDF embed)
//	base-uri 'self'        -- um `<base href>` injectado nao consegue redireccionar caminhos relativos
//	frame-ancestors 'self' -- a UI nao pode ser embutida por outro site (clickjacking)
//
// As outras ficam de FORA por medicao, nao por esquecimento: `script-src`/`style-src`/`connect-src`
// mexem no que a SPA carrega e liga (bundles, websockets, workers, blobs). Uma CSP restritiva escrita
// sem medir o que a app precisa **parte a UI** -- e uma UI partida e' pior do que uma CSP parcial.
//
//	*Uma politica que ninguem consegue ligar nao protege ninguem; uma que liga e nao parte nada e'
//	uma melhoria real, ainda que pequena.*
//
// Estas tres nao restringem scripts, estilos nem ligacoes: o risco de regressao e' proximo de zero, e
// o ganho (embed, base-hijack, clickjacking) e' concreto.
const PoliticaDeSegurancaDeConteudo = "object-src 'none'; base-uri 'self'; frame-ancestors 'self'"

// cabecalhosDeSeguranca acrescenta a CSP a todas as respostas.
//
// A TODAS, e nao so' ao HTML: a CSP em JSON e' ignorada pelos browsers, e aplicar aqui evita ter de
// saber qual rota serve o documento -- a resposta a "onde esta' o index.html" muda com o build da UI.
func cabecalhosDeSeguranca() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", PoliticaDeSegurancaDeConteudo)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

// garantirCabecalhosDeSeguranca e' o ponto unico onde a politica e' instalada no router.
func garantirCabecalhosDeSeguranca(r *gin.Engine) {
	if r == nil {
		return
	}
	r.Use(cabecalhosDeSeguranca())
}
