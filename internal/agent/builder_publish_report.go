package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------------------------------------------
// RELATÓRIO DE PUBLICAÇÃO (E-281)
// ---------------------------------------------------------------------------------------------------------------
//
// Copiar arquivos NÃO é publicar aplicação. Enquanto o Builder não gera backend/auth/banco, a publicação precisa
// DIZER o que entregou — senão o operador lê "published" e acredita que tem app rodando.

// BuilderPublishReport descreve o que a publicação REALMENTE entregou.
type BuilderPublishReport struct {
	FilesCopied      int  `json:"files_copied"`
	BackendIncluded  bool `json:"backend_included"`
	AuthIncluded     bool `json:"auth_included"`
	DatabaseIncluded bool `json:"database_included"`
	// Runnable é FALSO sem backend: copiar arquivos não é aplicação rodando.
	Runnable bool     `json:"runnable"`
	Missing  []string `json:"missing,omitempty"`
	Note     string   `json:"note,omitempty"`
}

// InspectBuilderPublish mede o que existe no diretório publicado. A medição é por ARQUIVO e CONTEÚDO — presumir
// "tem backend porque o status diz published" seria repetir o problema em vez de medi-lo.
func InspectBuilderPublish(root string) BuilderPublishReport {
	relatorio := BuilderPublishReport{}
	if strings.TrimSpace(root) == "" {
		relatorio.Missing = []string{"backend", "auth", "banco"}
		relatorio.Note = "sem diretório publicado: nada foi entregue"
		return relatorio
	}
	temAuth := false
	temBanco := false
	_ = filepath.Walk(root, func(caminho string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		relatorio.FilesCopied++
		relativo, relErr := filepath.Rel(root, caminho)
		if relErr != nil {
			relativo = info.Name()
		}
		base := strings.ToLower(info.Name())
		dir := strings.ToLower(filepath.ToSlash(relativo))
		if base == "server.go" || base == "main.go" || base == "app.go" ||
			strings.HasPrefix(dir, "api/") || strings.HasPrefix(dir, "backend/") || strings.HasPrefix(dir, "server/") {
			relatorio.BackendIncluded = true
		}
		dados, lerErr := os.ReadFile(caminho)
		if lerErr != nil {
			return nil
		}
		conteudo := strings.ToLower(string(dados))
		if strings.Contains(conteudo, "auth") || strings.Contains(conteudo, "login") || strings.Contains(conteudo, "jwt") {
			temAuth = true
		}
		if strings.Contains(conteudo, "database") || strings.Contains(conteudo, "sqlite") || strings.Contains(conteudo, "postgres") {
			temBanco = true
		}
		return nil
	})
	relatorio.AuthIncluded = temAuth
	relatorio.DatabaseIncluded = temBanco
	relatorio.Runnable = relatorio.BackendIncluded
	if !relatorio.BackendIncluded {
		relatorio.Missing = append(relatorio.Missing, "backend")
	}
	if !relatorio.AuthIncluded {
		relatorio.Missing = append(relatorio.Missing, "auth")
	}
	if !relatorio.DatabaseIncluded {
		relatorio.Missing = append(relatorio.Missing, "banco")
	}
	if relatorio.FilesCopied == 0 {
		relatorio.Note = "nenhum arquivo publicado: não há aplicação alguma"
		return relatorio
	}
	if !relatorio.Runnable {
		relatorio.Note = "arquivos copiados; aplicação NÃO executável: falta backend (a cópia não é publicação de app)"
		return relatorio
	}
	relatorio.Note = "backend presente; execução real ainda depende de rodar e testar o artefato"
	return relatorio
}
