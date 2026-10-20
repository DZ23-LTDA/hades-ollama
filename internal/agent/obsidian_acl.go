package agent

import (
	"errors"
	"fmt"
)

// Controlo de acesso a memoria por IDENTIDADE x ESCOPO x PERMISSAO.
//
// A regra e' uma so': sem concessao explicita, nega. Nao ha "escopo publico por defeito", nem "o dono ve tudo"
// -- se o dono precisa de ver, alguem escreveu a concessao. Uma ACL que permite por omissao e' uma ACL que
// ninguem consegue auditar: nao se distingue o que foi concedido do que simplesmente nunca foi negado.

// PermissaoDeMemoria: ler e escrever sao permissoes SEPARADAS -- quem le nao passa a poder escrever por simpatia.
type PermissaoDeMemoria uint8

const (
	PermissaoLer PermissaoDeMemoria = 1 << iota
	PermissaoEscrever
)

func (p PermissaoDeMemoria) String() string {
	switch p {
	case PermissaoLer:
		return "ler"
	case PermissaoEscrever:
		return "escrever"
	case PermissaoLer | PermissaoEscrever:
		return "ler+escrever"
	}
	return "nenhuma"
}

// ErrAcessoNegado: nao ha concessao que cubra esta identidade, este escopo e esta permissao.
var ErrAcessoNegado = errors.New("memory access denied")

// Concessao e' uma linha de ACL. Vazia de permissao nao concede nada -- e' recusada na validacao.
type Concessao struct {
	Identidade string
	Escopo     EscopoDeMemoria
	Permissoes PermissaoDeMemoria
}

// PoliticaDeAcesso e' a lista de concessoes. Sem concessoes, tudo negado.
type PoliticaDeAcesso struct {
	Concessoes []Concessao
}

// Autorizar diz se `identidade` pode exercer `permissao` sobre `escopo`.
//
// A mensagem de erro diz QUEM, ONDE e O QUE foi negado, e nao revela se existe memoria naquele escopo: negar
// acesso nao pode ser uma fuga de informacao sobre o que existe.
func (p PoliticaDeAcesso) Autorizar(identidade string, escopo EscopoDeMemoria, permissao PermissaoDeMemoria) error {
	if identidade == "" {
		return fmt.Errorf("%w: identidade vazia", ErrAcessoNegado)
	}
	if !escopoConhecido(escopo) {
		return fmt.Errorf("%w: identidade %q, escopo desconhecido %q", ErrAcessoNegado, identidade, string(escopo))
	}
	if permissao == 0 {
		return fmt.Errorf("%w: identidade %q, permissao ausente", ErrAcessoNegado, identidade)
	}

	for _, c := range p.Concessoes {
		if c.Identidade != identidade || c.Escopo != escopo {
			continue
		}
		if c.Permissoes&permissao == permissao {
			return nil
		}
		// Ha' concessao no escopo, mas nao para esta permissao: a recusa e' especifica.
		return fmt.Errorf("%w: identidade %q tem %s em %q, pediu %s",
			ErrAcessoNegado, identidade, c.Permissoes, string(escopo), permissao)
	}
	return fmt.Errorf("%w: identidade %q sem concessao em %q para %s",
		ErrAcessoNegado, identidade, string(escopo), permissao)
}

// Validar recusa concessoes que nao concedem nada e identidades/escopos vazios. Uma linha de ACL que nao faz
// nada e' pior do que nao existir: quem le a politica acredita que ela concede.
func (p PoliticaDeAcesso) Validar() error {
	for i, c := range p.Concessoes {
		if c.Identidade == "" {
			return fmt.Errorf("concessao %d sem identidade", i)
		}
		if !escopoConhecido(c.Escopo) {
			return fmt.Errorf("concessao %d com escopo desconhecido %q", i, string(c.Escopo))
		}
		if c.Permissoes == 0 {
			return fmt.Errorf("concessao %d (%s/%s) nao concede nada", i, c.Identidade, string(c.Escopo))
		}
		if c.Permissoes&^(PermissaoLer|PermissaoEscrever) != 0 {
			return fmt.Errorf("concessao %d com permissao desconhecida %d", i, c.Permissoes)
		}
	}
	return nil
}

// AnexarAutorizado e' o caminho de escrita COM a ACL aplicada -- a aresta so' protege quando alguem a consulta.
//
// A permissao vem do ESCOPO DA NOTA (`meta.Escopo`), nao de um argumento separado: dois parametros de escopo
// deixariam a porta aberta para autorizar em `PROJECT` e gravar em `TENANT`.
func AnexarAutorizado(raiz, relativo string, pol PoliticaDeAcesso, identidade string, meta MetaDaNota, m Memoria, versaoEsperada int) (bool, error) {
	if err := pol.Autorizar(identidade, meta.Escopo, PermissaoEscrever); err != nil {
		return false, err
	}
	return AnexarEmArquivoComBloqueio(raiz, relativo, meta, m, versaoEsperada)
}
