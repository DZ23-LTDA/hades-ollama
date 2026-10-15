package agent

import (
	"strings"
	"unicode"
)

// Complemento 48 / FASE 2 — MENCOES (`@agente`, `@equipe`, `@todos`).
//
// O QUE FOI MEDIDO ANTES DE ESCREVER (2026-10-15, commit 93f12ac19): `mention` tinha **ZERO** ocorrencias como
// conceito de conversa em `internal/agent`, em `server/` e em **todos os `.tsx`** do repositorio - o unico
// `mentions` era escopo do conector X/Twitter. Nao existia parser, nem destinos, nem regra nenhuma.
//
// REGRA DO COMPLEMENTO (③): **mencao so' dispara execucao com autorizacao + capacidade + orcamento.** Este
// arquivo NAO executa nada: ele apenas DIZ QUEM foi mencionado. *Quem decide se executa e' a politica, nao o
// parser* - e essa separacao e' o que impede uma mencao de virar um comando.

// Mencao e' um destino mencionado num corpo de mensagem.
type Mencao struct {
	// Nome e' o destino sem o `@`, normalizado em minusculas (a comparacao de mencao nao deve depender de
	// maiusculas que o autor digitou por habito).
	Nome string
	// Coletiva marca `@todos` e `@equipe`: falam com o time inteiro, nao com um agente.
	//
	// *Uma mencao coletiva e' mais perigosa que uma individual* - ela acorda varios executores ao mesmo tempo.
	// Por isso o tipo e' explicito: quem for decidir orcamento/execucao precisa distinguir as duas **sem**
	// comparar strings por conta propria.
	Coletiva bool
}

// nomesColetivos sao os destinos que falam com o time inteiro.
var nomesColetivos = map[string]bool{"todos": true, "equipe": true, "all": true, "time": true}

// MencoesNoTexto extrai as mencoes de um corpo de mensagem, na ORDEM em que aparecem e sem repetir
// (a primeira ocorrencia vence, para que a ordem seja reproduzivel).
//
// CUIDADOS QUE O TESTE FIXA (cada um ja' custou caro noutro lugar):
//   - **email NAO e' mencao**: em `ana@exemplo.com` o `@` esta' colado a um caractere alfanumerico ⇒ ignorado.
//     *Sem isto, todo endereco de e-mail acordaria um agente imaginario.*
//   - **`@` solto NAO e' mencao**: `@` sem nome depois nao gera nada (nem mencao vazia).
//   - **pontuacao nao engorda o nome**: `@ana,` ⇒ `ana` (a virgula fica de fora), e `@ana.` idem.
//   - **nome com acento e' nome**: `@joão` ⇒ `joão` (letras unicode contam).
//   - **`@` no inicio ou depois de espaco/pontuacao conta**: `@ana` e `(@ana)` sao mencoes.
func MencoesNoTexto(corpo string) []Mencao {
	if strings.TrimSpace(corpo) == "" {
		return nil
	}
	runas := []rune(corpo)
	vistas := map[string]bool{}
	var mencoes []Mencao
	for i := 0; i < len(runas); i++ {
		if runas[i] != '@' {
			continue
		}
		// O `@` so' ABRE mencao se nao estiver colado a um caractere de palavra (o caso do e-mail).
		if i > 0 && ehCaractereDeNome(runas[i-1]) {
			continue
		}
		j := i + 1
		var nome []rune
		for j < len(runas) && (ehCaractereDeNome(runas[j]) || runas[j] == '.' || runas[j] == '-') {
			nome = append(nome, runas[j])
			j++
		}
		// Apara pontos e hifens das pontas: `@ana.` -> `ana`, `@-x-` -> `x`.
		limpo := strings.Trim(strings.ToLower(string(nome)), ".-")
		i = j - 1 // nao reprocessa o que ja' foi consumido
		if limpo == "" {
			continue
		}
		if vistas[limpo] {
			continue
		}
		vistas[limpo] = true
		mencoes = append(mencoes, Mencao{Nome: limpo, Coletiva: nomesColetivos[limpo]})
	}
	return mencoes
}

// MencoesDeAgentes devolve apenas os destinos que NAO sao coletivos (agentes/pessoas nomeados).
// Existe para que quem for disparar execucao distinga, sem reimplementar a regra, "acordar um" de "acordar
// todos". *Nao e' conveniencia: e' a fronteira entre uma mencao e um alarme.*
func MencoesDeAgentes(mencoes []Mencao) []Mencao {
	var agentes []Mencao
	for _, mencao := range mencoes {
		if !mencao.Coletiva {
			agentes = append(agentes, mencao)
		}
	}
	return agentes
}

func ehCaractereDeNome(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
