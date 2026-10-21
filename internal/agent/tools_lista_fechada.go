package agent

import (
	"sort"
	"strings"
)

// FATIA 4 / T3 — LISTA FECHADA de ferramentas.
//
// DEFEITO MEDIDO pelo dono: no chat, o modelo respondeu a um pedido de execucao com comandos
// INVENTADOS -- `hades create-dir`, `hades create-file`, `hades list-files`, `hades open-browser`
// -- e NENHUM deles existe no binario (o real tem serve, create, show, run, stop, pull, push,
// signin, signout, list, ps, cp, ...).
//
// A causa nao e' o modelo saber menos: e' o produto nao o confrontar com o que EXISTE. Um pedido de
// ferramenta que nao esta' no registo tem de ser recusado aqui, com nome explicito, em vez de
// seguir para o utilizador como se fosse possivel.

// FerramentaExiste indica se o nome corresponde a uma ferramenta REGISTADA no registry.
//
// O registo e' a fonte da verdade (tools.go, map[string]Tool); esta funcao nao mantem uma lista
// paralela que possa divergir.
func (r *Registry) FerramentaExiste(nome string) bool {
	if r == nil {
		return false
	}
	limpo := strings.TrimSpace(nome)
	if limpo == "" {
		return false
	}
	_, ok := r.Get(limpo)
	return ok
}

// NomesFerramentas devolve os nomes registados, ordenados. Serve para o prompt do sistema e para a
// validacao documental: se um nome citado nao estiver aqui, e' alucinacao.
func (r *Registry) NomesFerramentas() []string {
	if r == nil {
		return nil
	}
	descritores := r.Descriptors()
	nomes := make([]string, 0, len(descritores))
	for _, d := range descritores {
		if nome := strings.TrimSpace(d.Name); nome != "" {
			nomes = append(nomes, nome)
		}
	}
	sort.Strings(nomes)
	return nomes
}

// ErrFerramentaInexistente marca a recusa de uma ferramenta que nao esta' registada.
var ErrFerramentaInexistente = errFerramentaInexistente{}

type errFerramentaInexistente struct{}

func (errFerramentaInexistente) Error() string {
	return "tool is not registered: refuse instead of inventing steps"
}
