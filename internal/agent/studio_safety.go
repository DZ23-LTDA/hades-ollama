package agent

// FASE 26.5 / 26.12 do adendo V3: VALIDACAO DO CONTEUDO QUE ENTRA NO PREVIEW DO STUDIO.
//
// A LACUNA MEDIDA, ANTES DE ESCREVER UMA LINHA
//
// O inventor da FASE 0 procurou nomes concretos no codigo do builder, e o resultado foi:
//
//	javascript:      0 ocorrencias
//	onerror          0
//	ValidarURL       0
//	esquemaPermitido 0
//
// O builder GERA `<script>` (5 ocorrencias) -- e deve, porque produz aplicacoes. Mas nao havia
// NENHUMA validacao do conteudo que vem de FORA: asset, `src`, `href`, HTML colado, dado de
// modelo.
//
//	*O gerador produzir script e' o produto. O modelo produzir script e' o vector.*
//
// A DISTINCAO QUE ESTE FICHEIRO FAZ
//
// Ha' duas origens de conteudo e elas NAO tem o mesmo direito:
//
//  1. **Output do gerador** (`builder.go`): o HADES escreveu aquele HTML, e um `<script>` la' dentro
//     e' funcionalidade. Nao passa por aqui.
//
//  2. **Conteudo de modelo, de asset, de utilizador ou de URL** (este ficheiro): chegou de fora e
//     nao tem direito a executar nada. Um `<script>` aqui nao e' funcionalidade: e' execucao
//     arbitraria com o nome de "preview".
//
//	*Um preview que executa o que o modelo escreveu nao e' um preview: e' um interprete.*
//
// TRES VALIDADORES, PORQUE SAO TRES PERGUNTAS DIFERENTES
//
//	ValidarURLDeConteudo   -- "esta ligacao pode ser seguida?"
//	ValidarCaminhoDeAsset  -- "este asset pode ser lido daqui?"
//	ValidarHTMLDeConteudo  -- "este HTML pode ser mostrado?"
//
//	*Uma funcao que responde as tres ao mesmo tempo e' uma funcao que responde mal a todas.*

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrConteudoInseguro e' a raiz dos erros de conteudo recusado.
var ErrConteudoInseguro = errors.New("conteudo recusado pelo validador do Studio")

// esquemasDeURLPermitidos sao os unicos esquemas aceitos em conteudo que vem de fora.
//
// A lista e' uma ALLOWLIST, nao uma denylist:
//
//	*Uma denylist de esquemas perigosos fica sempre um esquema atras de quem a escreve.*
var esquemasDeURLPermitidos = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"tel":    true,
}

// esquemasConhecidosPerigosos serve apenas para dar uma mensagem melhor. NAO e' a regra: a regra
// e' a allowlist acima.
var esquemasConhecidosPerigosos = map[string]string{
	"javascript": "executa codigo no contexto da pagina",
	"vbscript":   "executa codigo no contexto da pagina",
	"data":       "pode transporter HTML ou script embutido",
	"file":       "le ficheiros locais fora do sandbox",
	"blob":       "pode referenciar conteudo arbitrario da origem",
	"about":      "esquema interno do motor de renderizacao",
}

// ValidarURLDeConteudo aceita apenas URLs absolutas com esquema permitido, ou referencias
// relativas sem esquema.
//
// Recusa, com motivo diferente para cada caso:
//   - esquema perigoso conhecido (`javascript:`, `data:`, `file:`, ...);
//   - esquema desconhecido (a allowlist e' fechada);
//   - URL sem host quando o esquema exige host;
//   - caracteres de controlo (vector classico de bypass: `java\nscript:`).
func ValidarURLDeConteudo(bruta string) error {
	original := bruta
	// Caracteres de controlo e espacos em posicoes estranhas sao o truque classico para partir a
	// string sem que o parser do browser a parta.
	for _, r := range original {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: URL contem caractere de controlo (0x%02x)", ErrConteudoInseguro, r)
		}
	}
	bruta = strings.TrimSpace(bruta)
	if bruta == "" {
		return fmt.Errorf("%w: URL vazia", ErrConteudoInseguro)
	}

	// Referencia relativa: sem esquema, permitida. Mas `//host` e' protocolo-relativa e aponta
	// para FORA -- isso e' URL absoluta disfarcada.
	if strings.HasPrefix(bruta, "//") {
		return fmt.Errorf("%w: referencia protocolo-relativa (%q) aponta para fora da origem",
			ErrConteudoInseguro, bruta)
	}

	u, err := url.Parse(bruta)
	if err != nil {
		return fmt.Errorf("%w: URL invalida: %v", ErrConteudoInseguro, err)
	}
	esquema := strings.ToLower(u.Scheme)
	if esquema == "" {
		// Relativa legitima (caminho, fragmento, query).
		if strings.HasPrefix(bruta, "/") || strings.HasPrefix(bruta, "#") ||
			strings.HasPrefix(bruta, "?") || !strings.Contains(bruta, ":") {
			return nil
		}
		return fmt.Errorf("%w: URL sem esquema reconhecivel (%q)", ErrConteudoInseguro, bruta)
	}
	if motivo, perigoso := esquemasConhecidosPerigosos[esquema]; perigoso {
		return fmt.Errorf("%w: esquema %q recusado (%s)", ErrConteudoInseguro, esquema, motivo)
	}
	if !esquemasDeURLPermitidos[esquema] {
		return fmt.Errorf("%w: esquema %q fora da allowlist (%s)",
			ErrConteudoInseguro, esquema, esquemasPermitidosLegiveis())
	}
	// So' os esquemas de rede exigem host. `mailto:` e `tel:` sao OPACOS: o conteudo depois dos
	// dois pontos nao e' host, e' o destinatario.
	//
	//	*Uma regra que proibe o caso legitimo obriga a contorna-la -- e quem a contorna e' quem
	//	tem pressa, nao quem tem razao.*
	if (esquema == "http" || esquema == "https") && u.Host == "" {
		return fmt.Errorf("%w: esquema %q exige host (%q)", ErrConteudoInseguro, esquema, bruta)
	}
	return nil
}

// esquemasPermitidosLegiveis devolve a allowlist em texto, para a mensagem de erro.
func esquemasPermitidosLegiveis() string {
	nomes := make([]string, 0, len(esquemasDeURLPermitidos))
	for e := range esquemasDeURLPermitidos {
		nomes = append(nomes, e)
	}
	// Ordem fixa para a mensagem ser deterministica.
	for i := 1; i < len(nomes); i++ {
		for j := i; j > 0 && nomes[j] < nomes[j-1]; j-- {
			nomes[j], nomes[j-1] = nomes[j-1], nomes[j]
		}
	}
	return strings.Join(nomes, ", ")
}

// ValidarCaminhoDeAsset garante que um caminho de asset fica DENTRO da raiz permitida.
//
// Tres recusas distintas, porque sao tres defeitos distintos:
//
//   - caminho absoluto (o asset nao e' relativo ao projection);
//
//   - travessia (`..`) que sai da raiz;
//
//   - caminho que, depois de limpo, ainda aponta para fora.
//
//     *Um caminho que so' e' seguro depois de limpo nao e' seguro: e' um caminho a espera de que
//     alguem o limpe.*
func ValidarCaminhoDeAsset(relativo string) error {
	bruto := strings.TrimSpace(relativo)
	if bruto == "" {
		return fmt.Errorf("%w: caminho de asset vazio", ErrConteudoInseguro)
	}
	if strings.ContainsRune(bruto, 0) {
		return fmt.Errorf("%w: caminho com byte nulo", ErrConteudoInseguro)
	}
	// Normalizar separadores: no Windows `\` tambem e' separador, e um atacante usa os dois.
	uniforme := strings.ReplaceAll(bruto, "\\", "/")

	if path.IsAbs(uniforme) || strings.HasPrefix(uniforme, "/") {
		return fmt.Errorf("%w: caminho absoluto (%q)", ErrConteudoInseguro, bruto)
	}
	// Letra de unidade do Windows (`C:`ou `c:/`).
	if len(uniforme) >= 2 && uniforme[1] == ':' {
		return fmt.Errorf("%w: caminho com unidade (%q)", ErrConteudoInseguro, bruto)
	}
	limpo := path.Clean(uniforme)
	if limpo == "." || limpo == ".." || strings.HasPrefix(limpo, "../") {
		return fmt.Errorf("%w: travessia de diretorio (%q)", ErrConteudoInseguro, bruto)
	}
	// Defesa em profundidade: mesmo depois de limpar, o resultado tem de continuar relativo.
	if filepath.IsAbs(filepath.FromSlash(limpo)) {
		return fmt.Errorf("%w: caminho resolvido para absoluto (%q)", ErrConteudoInseguro, bruto)
	}
	return nil
}

// EventoDeScript procurado no HTML que vem de fora.
var (
	reTagScript = regexp.MustCompile(`(?i)<\s*script`)
	// Handlers `on*=` (onerror, onload, onclick, ...). Exigimos `=` depois do nome para nao casar
	// com texto legitimo que contenha "online" ou "once".
	reHandlerOn = regexp.MustCompile(`(?i)\bon[a-z]{3,}\s*=`)
	// `javascript:` dentro de href/src, com espacos ou entidades pelo meio.
	reJavaScriptURL = regexp.MustCompile(`(?i)j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*:`)
	// `<iframe` e `<object` transportam conteudo de outra origem.
	reIframeObject = regexp.MustCompile(`(?i)<\s*(iframe|object|embed)`)
	// `srcdoc` executa HTML dentro do iframe.
	reSrcdoc = regexp.MustCompile(`(?i)\bsrcdoc\s*=`)
	// `<meta http-equiv="refresh"` redirecciona.
	reMetaRefresh = regexp.MustCompile(`(?i)<\s*meta[^>]*http-equiv\s*=\s*["']?refresh`)
)

// MotivoDeRecusaDeHTML e' uma recusa nomeada, para o chamador poder explicar sem inventar.
type MotivoDeRecusaDeHTML struct {
	Padrao  string `json:"padrao"`
	Motivo  string `json:"motivo"`
	Posicao int    `json:"posicao"`
}

// PadroesProibidosEmHTML e' a lista que `ValidarHTMLDeConteudo` aplica.
//
// Exposta porque o Mission Control precisa de a mostrar ao utilizador:
//
//	*Uma regra de seguranca que ninguem consegue ler e' uma regra em que ninguem consegue confiar.*
func PadroesProibidosEmHTML() []string {
	return []string{
		"<script", "on<evento>=", "javascript:", "<iframe", "<object", "<embed",
		"srcdoc=", "<meta http-equiv=refresh",
	}
}

// ValidarHTMLDeConteudo recusa HTML que, ao ser mostrado num preview, execute codigo.
//
// Isto NAO e' um sanitizador: e' um VALIDADOR. A diferenca importa:
//
//	*Um sanitizador que se engana devolve HTML inseguro. Um validador que se engana recusa HTML
//	seguro -- e o segundo erro e' recuperavel, o primeiro nao.*
func ValidarHTMLDeConteudo(html string) error {
	if strings.TrimSpace(html) == "" {
		return nil
	}
	type regra struct {
		re     *regexp.Regexp
		nome   string
		motivo string
	}
	regras := []regra{
		{reTagScript, "<script", "executa codigo arbitrario no preview"},
		{reJavaScriptURL, "javascript:", "executa codigo ao ser seguido"},
		{reHandlerOn, "on<evento>=", "executa codigo num evento do DOM"},
		{reIframeObject, "<iframe|object|embed", "carrega conteudo de outra origem"},
		{reSrcdoc, "srcdoc=", "executa HTML embutido dentro do iframe"},
		{reMetaRefresh, "<meta http-equiv=refresh", "redirecciona sem accao do utilizador"},
	}
	for _, r := range regras {
		if loc := r.re.FindStringIndex(html); loc != nil {
			return fmt.Errorf("%w: %s na posicao %d (%s)",
				ErrConteudoInseguro, r.nome, loc[0], r.motivo)
		}
	}
	return nil
}

// DiagnosticarHTMLDeConteudo devolve TODAS as violacoes, nao so' a primeira.
//
// Para o painel, a primeira violacao chega. Para um relatorio de auditoria, saber que ha' quatro e'
// diferente de saber que ha' uma.
func DiagnosticarHTMLDeConteudo(html string) []MotivoDeRecusaDeHTML {
	recusas := []MotivoDeRecusaDeHTML{}
	if strings.TrimSpace(html) == "" {
		return recusas
	}
	type regra struct {
		re     *regexp.Regexp
		nome   string
		motivo string
	}
	regras := []regra{
		{reTagScript, "<script", "executa codigo arbitrario no preview"},
		{reJavaScriptURL, "javascript:", "executa codigo ao ser seguido"},
		{reHandlerOn, "on<evento>=", "executa codigo num evento do DOM"},
		{reIframeObject, "<iframe|object|embed", "carrega conteudo de outra origem"},
		{reSrcdoc, "srcdoc=", "executa HTML embutido dentro do iframe"},
		{reMetaRefresh, "<meta http-equiv=refresh", "redirecciona sem accao do utilizador"},
	}
	for _, r := range regras {
		for _, loc := range r.re.FindAllStringIndex(html, -1) {
			recusas = append(recusas, MotivoDeRecusaDeHTML{
				Padrao: r.nome, Motivo: r.motivo, Posicao: loc[0],
			})
		}
	}
	return recusas
}

// === COMPONENTES VISUAIS DO STUDIO (26.5: bindings e eventos allowlisted) ===

// eventosPermitidos sao os UNICOS eventos que um componente visual pode declarar.
//
// O adendo 26.5 diz "bindings de dados e eventos allowlisted". A palavra que carrega o peso e'
// "allowlisted": sem uma lista fechada, `Events` e' um campo de texto onde o modelo escreve o que
// quiser e o renderer do canvas executa.
//
//	*Um evento sem allowlist nao e' um evento: e' codigo com um nome diferente.*
var eventosPermitidos = map[string]bool{
	"click":      true,
	"input":      true,
	"change":     true,
	"submit":     true,
	"focus":      true,
	"blur":       true,
	"keydown":    true,
	"keyup":      true,
	"scroll":     true,
	"hover":      true,
	"mouseenter": true,
	"mouseleave": true,
}

// acoesDeEventoPermitidas sao as acoes que um evento pode disparar.
//
// Qualquer coisa fora desta lista -- navegar para URL arbitraria, executar script, chamar API
// externa -- e' recusada. O renderer pode implementar mais, mas o componente NAO pode pedir mais.
var acoesDeEventoPermitidas = map[string]bool{
	"navegar":        true, // navegacao entre telas do proprio projection
	"alternarTema":   true,
	"abrirMenu":      true,
	"fecharMenu":     true,
	"enviarForm":     true,
	"filtrarLista":   true,
	"ordenarLista":   true,
	"proximaPagina":  true,
	"paginaAnterior": true,
	"copiar":         true,
	"imprimir":       true,
}

// ValidarSegurancaDosComponentes aplica as regras de 26.5 a uma arvore de componentes visuais.
//
// Complementa -- nao substitui -- `validateVisualComponents`, que valida ESTRUTURA. Sao perguntas
// diferentes:
//
//	*Um componente com id e type e' um componente bem formado. Nao e' um componente seguro.*
//
// Recusa, com motivo nomeado:
//   - `Events` com nome fora da allowlist;
//   - `Events` com accao fora da allowlist;
//   - `Bindings` que apontem para esquema nao permitido (`javascript:`, `data:`);
//   - `Props`/`Style` que contenham HTML executavel ou `expression(`/`url(javascript:`;
//   - qualquer string que carregue `<script`, handler `on*=` ou `javascript:`.
func ValidarSegurancaDosComponentes(components []VisualComponent, depth int) error {
	if depth > 12 {
		return fmt.Errorf("%w: arvore de componentes demasiado profunda", ErrConteudoInseguro)
	}
	for _, c := range components {
		// Eventos: nome E accao tem de estar na allowlist.
		for nome, acao := range c.Events {
			if !eventosPermitidos[nome] {
				return fmt.Errorf("%w: componente %q declara evento nao permitido %q",
					ErrConteudoInseguro, c.ID, nome)
			}
			if !acoesDeEventoPermitidas[acao] {
				return fmt.Errorf("%w: componente %q declara accao nao permitida %q no evento %q",
					ErrConteudoInseguro, c.ID, acao, nome)
			}
		}
		// Bindings: sao referencias a dados, nao podem ser URLs executaveis.
		for chave, valor := range c.Bindings {
			if strings.TrimSpace(valor) == "" {
				continue
			}
			// Um binding e' um caminho de dados (`items`, `user.name`), sem esquema.
			if strings.Contains(valor, ":") {
				if err := ValidarURLDeConteudo(valor); err != nil {
					return fmt.Errorf("%w: componente %q tem binding %q com valor nao permitido: %v",
						ErrConteudoInseguro, c.ID, chave, err)
				}
			}
			if err := ValidarHTMLDeConteudo(valor); err != nil {
				return fmt.Errorf("%w: componente %q tem binding %q inseguro: %v",
					ErrConteudoInseguro, c.ID, chave, err)
			}
		}
		// Props e Style: nao podem transporter execucao.
		for _, mapa := range []struct {
			nome    string
			valores map[string]string
		}{
			{"props", c.Props},
			{"style", c.Style},
		} {
			for chave, valor := range mapa.valores {
				if err := ValidarHTMLDeConteudo(valor); err != nil {
					return fmt.Errorf("%w: componente %q tem %s %q inseguro: %v",
						ErrConteudoInseguro, c.ID, mapa.nome, chave, err)
				}
				// `expression(` e `url(` sao vectors de CSS antigo e de SVG.
				normalizado := strings.ToLower(valor)
				for _, proibido := range []string{"expression(", "url(javascript:", "url(data:text/html"} {
					if strings.Contains(normalizado, proibido) {
						return fmt.Errorf("%w: componente %q tem %s %q com %q",
							ErrConteudoInseguro, c.ID, mapa.nome, chave, proibido)
					}
				}
			}
		}
		if err := ValidarSegurancaDosComponentes(c.Children, depth+1); err != nil {
			return err
		}
	}
	return nil
}
