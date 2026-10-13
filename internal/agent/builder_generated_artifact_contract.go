package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------------------------------------------
// CONTRATO DO ARTEFATO GERADO: OFFLINE E COM LIMITES DECLARADOS (E-285)
// ---------------------------------------------------------------------------------------------------------------
//
// O app full-stack gerado usa PERSISTÊNCIA EM ARQUIVO com escrita atômica, não um SGBD relacional. Isso não é
// descuido: é DECISÃO MEDIDA.
//
//   - `mattn/go-sqlite3` (o driver já presente no repositório) exige CGO, e esta máquina NÃO tem compilador C
//     (medido em E-274): um app gerado que dependa dele NÃO compila em ambiente limpo;
//   - `jackc/pgx/v5` exige um PostgreSQL RODANDO: um artefato entregue ao usuário não pode assumir isso.
//
// Logo, o artefato gerado é AUTOSSUFICIENTE: sem `require` no go.mod, compila offline, e DECLARA os próprios
// limites. Um artefato que precisa baixar módulos — ou compilar C — deixa de ser local-first.
//
// As duas travas abaixo impedem que alguém reintroduza uma dependência externa ou apague a declaração do limite:
// em ambos os casos o artefato passaria a prometer mais do que entrega.

// GeneratedArtifactIssues audita um artefato publicado: devolve os problemas encontrados (lista vazia = conforme).
func GeneratedArtifactIssues(dir string) []string {
	problemas := []string{}
	if strings.TrimSpace(dir) == "" {
		return []string{"sem diretório de artefato: nada foi entregue"}
	}
	// 1. go.mod SEM dependências externas: o artefato precisa compilar offline.
	goMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		problemas = append(problemas, "go.mod ausente: o artefato não é um módulo autossuficiente")
	} else if texto := string(goMod); strings.Contains(texto, "require") {
		// `require` — em qualquer forma — significa download de módulo no build do usuário.
		problemas = append(problemas, "go.mod declara dependência externa: o artefato deixa de compilar offline")
	}
	// 2. Os limites precisam estar DECLARADOS: limite não declarado é o que engana quem recebe o artefato.
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		problemas = append(problemas, "README.md ausente: os limites do artefato não foram declarados")
		return problemas
	}
	texto := string(readme)
	// Busca em DUAS partes: o README usa negrito no meio da frase ("**nao** um SGBD relacional"), e casar a frase
	// inteira exigiria adivinhar a formatação — verificação que depende de formatação quebra por motivo errado.
	if !strings.Contains(texto, "Persistencia em ARQUIVO") || !strings.Contains(texto, "SGBD relacional") {
		problemas = append(problemas, "README não declara a persistência em arquivo (e a ausência de SGBD relacional)")
	}
	if !strings.Contains(texto, "Token em memoria") {
		problemas = append(problemas, "README não declara que o token vive em memória (sessões caem no reinício)")
	}
	return problemas
}
