package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrPoliticaDeAcessoInvalida recusa uma configuracao de concessoes que nao pode ser lida com
// seguranca. Fail-closed: configuracao ilegivel NAO vira "sem concessoes" por conveniencia --
// vira erro visivel, porque uma politica que se degrada em silencio esconde o proprio defeito.
var ErrPoliticaDeAcessoInvalida = errors.New("invalid access policy")

// concessaoEmJSON e' a forma textual de UMA concessao. Os nomes de campo sao os mesmos do contrato
// Go (`identidade`, `escopo`, `permissoes`); as permissoes sao nomes, nao numeros, para que ninguem
// tenha de saber que ler=1 e escrever=2.
type concessaoEmJSON struct {
	Identidade string   `json:"identidade"`
	Escopo     string   `json:"escopo"`
	Permissoes []string `json:"permissoes"`
}

// permissoesPorNome traduz os nomes aceites. Qualquer outro nome e' erro: aceitar um nome
// desconhecido como "nenhuma permissao" faria uma concessao mal escrita parecer inofensiva.
var permissoesPorNome = map[string]PermissaoDeMemoria{
	"ler":          PermissaoLer,
	"read":         PermissaoLer,
	"escrever":     PermissaoEscrever,
	"write":        PermissaoEscrever,
	"ler+escrever": PermissaoLer | PermissaoEscrever,
}

// escoposPorNome aceita o valor textual do escopo e o nome da constante Go.
var escoposPorNome = map[string]EscopoDeMemoria{
	"SESSION":           EscopoSession,
	"PERSON":            EscopoPerson,
	"TENANT":            EscopoTenant,
	"PROJECT":           EscopoProject,
	"WORKSPACE":         EscopoWorkspace,
	"WORKTREE":          EscopoWorktree,
	"GOAL":              EscopoGoal,
	"AGENT":             EscopoAgent,
	"SHARED_AUTHORIZED": EscopoSharedAuthorized,
}

// ParsearPoliticaDeAcesso le a configuracao de concessoes do vault.
//
// Entrada vazia devolve politica vazia (nega tudo) e NENHUM erro: nao configurar e' uma escolha
// legitima e o resultado e' o mais restritivo possivel. Entrada malformada devolve ERRO, nunca uma
// politica vazia -- as duas coisas nao podem ser a mesma resposta.
func ParsearPoliticaDeAcesso(texto string) (PoliticaDeAcesso, error) {
	limpo := strings.TrimSpace(texto)
	if limpo == "" {
		return PoliticaDeAcesso{}, nil
	}

	descodificador := json.NewDecoder(strings.NewReader(limpo))
	descodificador.DisallowUnknownFields()
	var entradas []concessaoEmJSON
	if err := descodificador.Decode(&entradas); err != nil {
		return PoliticaDeAcesso{}, fmt.Errorf("%w: %v", ErrPoliticaDeAcessoInvalida, err)
	}
	if len(entradas) == 0 {
		return PoliticaDeAcesso{}, fmt.Errorf("%w: lista de concessoes vazia", ErrPoliticaDeAcessoInvalida)
	}

	concessoes := make([]Concessao, 0, len(entradas))
	for i, entrada := range entradas {
		identidade := strings.TrimSpace(entrada.Identidade)
		if identidade == "" {
			return PoliticaDeAcesso{}, fmt.Errorf("%w: concessao %d sem identidade", ErrPoliticaDeAcessoInvalida, i)
		}
		escopo, conhecido := escoposPorNome[strings.ToUpper(strings.TrimSpace(entrada.Escopo))]
		if !conhecido {
			return PoliticaDeAcesso{}, fmt.Errorf("%w: concessao %d com escopo %q desconhecido", ErrPoliticaDeAcessoInvalida, i, entrada.Escopo)
		}
		if len(entrada.Permissoes) == 0 {
			return PoliticaDeAcesso{}, fmt.Errorf("%w: concessao %d sem permissoes", ErrPoliticaDeAcessoInvalida, i)
		}
		var permissoes PermissaoDeMemoria
		for _, nome := range entrada.Permissoes {
			valor, ok := permissoesPorNome[strings.ToLower(strings.TrimSpace(nome))]
			if !ok {
				return PoliticaDeAcesso{}, fmt.Errorf("%w: concessao %d com permissao %q desconhecida", ErrPoliticaDeAcessoInvalida, i, nome)
			}
			permissoes |= valor
		}
		concessoes = append(concessoes, Concessao{Identidade: identidade, Escopo: escopo, Permissoes: permissoes})
	}

	politica := PoliticaDeAcesso{Concessoes: concessoes}
	if err := politica.Validar(); err != nil {
		return PoliticaDeAcesso{}, fmt.Errorf("%w: %v", ErrPoliticaDeAcessoInvalida, err)
	}
	return politica, nil
}
