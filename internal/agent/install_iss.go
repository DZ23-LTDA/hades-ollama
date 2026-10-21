package agent

// FASE 28.6-A: a PONTE entre o catalogo de pacotes e o instalador Inno Setup.
//
// O PROBLEMA
//
// O `app/ollama.iss` tem 324 linhas e **nenhuma seccao `[Types]` ou `[Components]`**. Os 10
// `Source:` instalam tudo, sempre. O catalogo Go (`install_profile.go`) sabe quais sao os 9
// perfis e os 18 pacotes, mas esse conhecimento nao chega ao instalador -- e um catalogo que so'
// existe no servidor nao muda o que o utilizador instala.
//
// O QUE ESTE FICHEIRO FAZ
//
// Traduz o catalogo para o texto Inno Setup que o instalador precisa: `[Types]` (os perfis) e
// `[Components]` (os pacotes). Nao escreve ficheiros, nao toca no `.iss`, nao executa o Inno:
// **devolve texto**, para a decisao ser testavel sem privilégios e sem payload.
//
// A REGRA QUE MANDA EM TUDO
//
// **Nenhum componente que exige aprovacao sai marcado por omissao.**
//
// O adendo diz que o perfil e' intencao, nao permissao. E o Inno tem uma forma silenciosa de
// violar isto: um componente sem `Types:` atribuido aparece como selecionavel, e um marcado com
// `selected` aparece JA' ESCOLHIDO. *Um componente que cria servico e abre porta nao pode
// aparecer escolhido antes de o utilizador ler a linha.*
//
// O SEGUNDO CUIDADO: `hades-core` NAO e' componente
//
// O nucleo e' obrigatorio. No Inno, um componente sem `Types:` e' opcional -- e tornar o nucleo
// opcional permitiria instalar um HADES sem Mission Control. *Um produto sem o proprio nucleo
// nao e' uma instalacao minima: e' uma instalacao avariada.*

import (
	"fmt"
	"sort"
	"strings"
)

// NomeDoTipoInno devolve o nome que o perfil tem no `[Types]`.
//
// Minusculas com hifen: *o Inno trata os nomes de tipo sem distincao de maiusculas, e um nome com
// espacos ou underscore e' uma armadilha na linha de comando `/TYPE=`.*
func NomeDoTipoInno(p PerfilDeInstalacao) string {
	return strings.ToLower(strings.ReplaceAll(string(p), "_", "-"))
}

// DescricaoDoPerfil e' o texto que o instalador mostra no seletor de tipo.
//
// Vem do perfil, nao da UI: *uma descricao escrita no frontend descreve o produto que o
// frontend imagina, nao o que o instalador instala.*
func DescricaoDoPerfil(p PerfilDeInstalacao) string {
	switch p {
	case PerfilBasic:
		return "Basico - HADES Core, Mission Control e diagnostico"
	case PerfilRecommended:
		return "Recomendado - Core + IA local, navegador controlado e plugins"
	case PerfilDeveloperComplete:
		return "Desenvolvimento - Core + toolchains de codigo e release"
	case PerfilMobile:
		return "Movel - Core + Android, Expo e Flutter"
	case PerfilMediaAndVoice:
		return "Media e voz - Core + media, voz e OCR"
	case PerfilBrowserAndComputerUse:
		return "Navegador e Computer Use - Core + controlo do computador"
	case PerfilContainersAndCloud:
		return "Containers e nuvem - Core + Docker e CLIs de cloud"
	case PerfilCustom:
		return "Personalizado - escolha componente a componente"
	case PerfilFullOffline:
		return "Offline completo - Core + IA local e toolchains sem rede"
	}
	return string(p)
}

// ComponenteObrigatorio diz se o pacote nao pode ser desmarcado.
//
// Apenas `hades-core`. *Um nucleo opcional permitiria instalar um HADES sem Mission Control.*
func ComponenteObrigatorio(pacoteID string) bool {
	return pacoteID == "hades-core"
}

// ComponenteSelecionavelPorOmissao diz se o pacote pode aparecer marcado.
//
// Falso para qualquer pacote cujo impacto exija aprovacao. *Um componente que cria servico e
// abre porta nao pode aparecer escolhido antes de o utilizador ler a linha.*
func ComponenteSelecionavelPorOmmissao(pacoteID string) bool {
	if ComponenteObrigatorio(pacoteID) {
		// O nucleo nao e' "selecionavel": e' fixo. Vai em `[Files]`, nao em `[Components]`.
		return false
	}
	pkg, existe := NoCatalogo(pacoteID)
	if !existe {
		return false
	}
	return !pkg.Impacto.ExigeAprovacao()
}

// SeccaoTypes gera o texto do `[Types]`.
//
// `CUSTOM` leva `Flags: iscustom` e e' o UNICO que o leva: *dois tipos "custom" no Inno e' um
// erro de sintaxe que so' aparece no build.*
func SeccaoTypes() string {
	var b strings.Builder
	b.WriteString("[Types]\n")
	// "Completo" primeiro: e' o que o utilizador espera ver no topo.
	b.WriteString("; Gerado a partir de internal/agent/install_profile.go. NAO EDITAR A MAO.\n")
	for _, p := range PerfisDeInstalacao() {
		nome := NomeDoTipoInno(p)
		if p == PerfilCustom {
			b.WriteString(fmt.Sprintf("Name: \"%s\"; Description: \"%s\"; Flags: iscustom\n",
				nome, escaparInno(DescricaoDoPerfil(p))))
			continue
		}
		b.WriteString(fmt.Sprintf("Name: \"%s\"; Description: \"%s\"\n",
			nome, escaparInno(DescricaoDoPerfil(p))))
	}
	return b.String()
}

// SeccaoComponents gera o texto do `[Components]`.
//
// Cada componente lista os tipos que o incluem. `hades-core` fica de FORA: e' obrigatorio e vai
// no `[Files]` sem condicao.
//
// Um componente sem nenhum tipo seria invisivel; um componente com tipo e sem `selected` aparece
// desmarcado. E' exactamente o que se quer para o que exige aprovacao.
func SeccaoComponents() string {
	var b strings.Builder
	b.WriteString("[Components]\n")
	b.WriteString("; Gerado a partir de internal/agent/install_profile.go. NAO EDITAR A MAO.\n")
	b.WriteString("; `hades-core` fica de fora: e' obrigatorio e vai no [Files] sem condicao.\n")

	for _, pkg := range CatalogoDePacotes() {
		if ComponenteObrigatorio(pkg.ID) {
			continue
		}
		// Tipos que incluem este pacote, em ordem canonica.
		var tipos []string
		for _, p := range PerfisDeInstalacao() {
			if p == PerfilCustom {
				continue
			}
			for _, incluido := range pkg.Perfis {
				if incluido == p {
					tipos = append(tipos, NomeDoTipoInno(p))
					break
				}
			}
		}
		sort.Strings(tipos)

		linha := fmt.Sprintf("Name: \"%s\"; Description: \"%s\"",
			pkg.ID, escaparInno(pkg.Nome))
		if len(tipos) > 0 {
			linha += "; Types: " + strings.Join(tipos, " ")
		}
		// A regra: sem `selected` para o que exige aprovacao.
		// O Inno seleciona por omissao os componentes cujo `Types` casa com o tipo escolhido,
		// exceto quando a linha traz `Flags: unchecked` ou o utilizador desmarca. Para o que
		// exige aprovacao, `unchecked` garante que nunca aparece escolhido por engano.
		if !ComponenteSelecionavelPorOmmissao(pkg.ID) {
			linha += "; Flags: unchecked"
		}
		b.WriteString(linha + "\n")
	}
	return b.String()
}

// TextoDoInstaladorModular devolve `[Types]` + `[Components]` prontos a colar.
func TextoDoInstaladorModular() string {
	return SeccaoTypes() + "\n" + SeccaoComponents()
}

// escaparInno protege aspas e barras no texto de descricao.
//
// O Inno usa `""` para representar uma aspa dentro de uma string entre aspas. *Um nome com aspa
// partiria a linha inteira e o build falharia num sitio que nao aponta para aqui.*
func escaparInno(texto string) string {
	return strings.ReplaceAll(texto, `"`, `""`)
}

// ProblemaDoInstalador e' um achado da auto-auditoria do texto gerado.
type ProblemaDoInstalador struct {
	Mensagem string `json:"mensagem"`
}

// AuditarTextoDoInstalador verifica invariantes que o Inno so' reclamaria no build.
//
// Cobre: secções presentes, `iscustom` unico, nucleo fora de `[Components]`, nenhum componente
// com impacto marcado por omissao, e nomes de tipo sem caracteres proibidos.
//
// *Um erro de sintaxe no `.iss` custa um build; um teste aqui custa milissegundos.*
func AuditarTextoDoInstalador() []ProblemaDoInstalador {
	var problemas []ProblemaDoInstalador
	texto := TextoDoInstaladorModular()

	if !strings.HasPrefix(texto, "[Types]") {
		problemas = append(problemas, ProblemaDoInstalador{
			Mensagem: "o texto nao comeca por [Types]",
		})
	}
	if !strings.Contains(texto, "\n[Components]") {
		problemas = append(problemas, ProblemaDoInstalador{
			Mensagem: "falta a seccao [Components]",
		})
	}

	// `iscustom` exatamente uma vez.
	if n := strings.Count(texto, "iscustom"); n != 1 {
		problemas = append(problemas, ProblemaDoInstalador{
			Mensagem: fmt.Sprintf("`iscustom` aparece %d vezes; o Inno exige exatamente 1", n),
		})
	}

	// O nucleo nao pode estar em [Components].
	if strings.Contains(texto, `Name: "hades-core"`) {
		problemas = append(problemas, ProblemaDoInstalador{
			Mensagem: "hades-core aparece em [Components]; o nucleo nao e' opcional",
		})
	}

	// Nenhum componente com impacto pode aparecer sem `unchecked`.
	for _, linha := range strings.Split(texto, "\n") {
		if !strings.HasPrefix(linha, "Name: \"") {
			continue
		}
		if strings.HasPrefix(linha, "Name: \"basic\"") ||
			strings.HasPrefix(linha, "Name: \"recommended\"") ||
			strings.HasPrefix(linha, "Name: \"developer-complete\"") ||
			strings.HasPrefix(linha, "Name: \"mobile\"") ||
			strings.HasPrefix(linha, "Name: \"media-and-voice\"") ||
			strings.HasPrefix(linha, "Name: \"browser-and-computer-use\"") ||
			strings.HasPrefix(linha, "Name: \"containers-and-cloud\"") ||
			strings.HasPrefix(linha, "Name: \"custom\"") ||
			strings.HasPrefix(linha, "Name: \"full-offline\"") {
			continue // e' um tipo, nao um componente
		}
		id := extrairIDDaLinha(linha)
		if id == "" {
			continue
		}
		if !ComponenteSelecionavelPorOmmissao(id) && !strings.Contains(linha, "unchecked") {
			problemas = append(problemas, ProblemaDoInstalador{
				Mensagem: "componente " + id + " exige aprovacao mas nao tem `unchecked`",
			})
		}
	}

	// Nomes de tipo sem espacos nem underscore.
	for _, p := range PerfisDeInstalacao() {
		nome := NomeDoTipoInno(p)
		if strings.ContainsAny(nome, " _") {
			problemas = append(problemas, ProblemaDoInstalador{
				Mensagem: "nome de tipo invalido: " + nome,
			})
		}
	}
	return problemas
}

// extrairIDDaLinha le o identificador de uma linha `Name: "x"; ...`.
func extrairIDDaLinha(linha string) string {
	const prefixo = `Name: "`
	i := strings.Index(linha, prefixo)
	if i < 0 {
		return ""
	}
	resto := linha[i+len(prefixo):]
	fim := strings.Index(resto, `"`)
	if fim < 0 {
		return ""
	}
	return resto[:fim]
}
