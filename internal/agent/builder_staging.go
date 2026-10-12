package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Estagio 12 (deploy em staging e rollback): o Builder publicava arquivos e nunca SUBIA o que publicou, entao nao
// havia como verificar saude antes de promover uma versao. Aqui um projeto fullstack e compilado e executado num
// palco (staging), e a promocao so acontece DEPOIS de o palco responder saudavel.

// ErrBuilderStagingUnsupported indica kind sem processo executavel (hoje so o fullstack tem backend proprio).
var ErrBuilderStagingUnsupported = errors.New("builder staging is only supported for fullstack projects")

// ErrBuilderPromotionUnhealthy indica promocao recusada: promover o que nao responde saudavel trocaria uma versao
// boa por uma quebrada.
var ErrBuilderPromotionUnhealthy = errors.New("builder promotion refused: staged process is not healthy")

// BuilderStage e um app publicado RODANDO: guarda a versao exata que foi subida, para a promocao ser daquela
// versao e nao da "atual" (que pode ter mudado no meio do caminho).
type BuilderStage struct {
	ProjectID      string
	OrganizationID string
	Version        int
	PublishedPath  string
	URL            string
	DataDir        string

	processo *exec.Cmd
}

// StageFullstack compila e sobe o app publicado da versao ativa. A senha vem do chamador: nao existe credencial
// padrao, e o app gerado se recusa a subir sem ela.
func (b *BuilderService) StageFullstack(ctx context.Context, id, adminPassword string) (*BuilderStage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(adminPassword) == "" {
		return nil, errors.New("builder staging requires an admin password: generated apps refuse to start without one")
	}
	project, err := b.Get(id)
	if err != nil {
		return nil, err
	}
	if project.Kind != BuilderFullstack {
		return nil, fmt.Errorf("%w: kind=%s", ErrBuilderStagingUnsupported, project.Kind)
	}
	publicado := strings.TrimSpace(project.PublishedPath)
	if publicado == "" {
		return nil, errors.New("builder staging requires a published version: chame PublishLocal antes")
	}
	if info, err := os.Stat(publicado); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("published path not usable: %s", publicado)
	}

	// Compila fora do diretorio publicado (nao poluir a versao publicada) e com o modulo proprio do app.
	dirBin, err := os.MkdirTemp("", "hades-stage-bin-")
	if err != nil {
		return nil, err
	}
	nome := "app"
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}
	binario := filepath.Join(dirBin, nome)
	compilar := exec.CommandContext(ctx, "go", "build", "-o", binario, ".")
	compilar.Dir = publicado
	compilar.Env = os.Environ()
	if saida, err := compilar.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("staging build failed: %w: %s", err, strings.TrimSpace(string(saida)))
	}

	dirDados, err := os.MkdirTemp("", "hades-stage-data-")
	if err != nil {
		return nil, err
	}
	processo := exec.Command(binario)
	processo.Dir = publicado
	processo.Env = append(os.Environ(),
		"APP_ADMIN_PASSWORD="+adminPassword,
		"APP_DATA_DIR="+dirDados,
		"PORT=0",
	)
	saida, err := processo.StdoutPipe()
	if err != nil {
		return nil, err
	}
	processo.Stderr = os.Stderr
	if err := processo.Start(); err != nil {
		return nil, fmt.Errorf("staging start failed: %w", err)
	}
	linha, err := bufio.NewReader(saida).ReadString('\n')
	if err != nil {
		_ = processo.Process.Kill()
		return nil, fmt.Errorf("staged app did not announce a port: %w", err)
	}
	linha = strings.TrimSpace(linha)
	if !strings.HasPrefix(linha, "PORT=") {
		_ = processo.Process.Kill()
		return nil, fmt.Errorf("staged app announced %q instead of PORT=<numero>", linha)
	}
	// A versao do palco vem do CAMINHO PUBLICADO (published/<id>/vN), nao de project.Version: depois de um
	// rollback o projeto continua na ultima versao, mas o que esta sendo servido e a versao revertida. Usar
	// project.Version aqui faria o palco MENTIR sobre o que serve.
	versao, err := builderVersionFromPublishedPath(publicado)
	if err != nil {
		return nil, err
	}
	return &BuilderStage{
		ProjectID:      project.ID,
		OrganizationID: project.OrganizationID,
		Version:        versao,
		PublishedPath:  publicado,
		URL:            "http://127.0.0.1:" + strings.TrimPrefix(linha, "PORT="),
		DataDir:        dirDados,
		processo:       processo,
	}, nil
}

// Health exige 200 em /healthz: "subiu o processo" nao e o mesmo que "esta atendendo".
func (s *BuilderStage) Health(ctx context.Context) error {
	if s == nil || s.processo == nil || s.processo.Process == nil {
		return errors.New("stage is not running")
	}
	cliente := &http.Client{Timeout: 3 * time.Second}
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL+"/healthz", nil)
	if err != nil {
		return err
	}
	resposta, err := cliente.Do(requisicao)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resposta.StatusCode)
	}
	return nil
}

// Stop encerra o processo do palco. Idempotente.
func (s *BuilderStage) Stop() {
	if s == nil || s.processo == nil || s.processo.Process == nil {
		return
	}
	_ = s.processo.Process.Kill()
	_, _ = s.processo.Process.Wait()
	s.processo = nil
}

// PromoteIfHealthy re-aponta a publicacao ativa para a versao DO PALCO, e somente se o palco responder saudavel.
// Fail-closed: qualquer duvida sobre a saude impede a promocao — o publicado continua sendo o que ja funcionava.
func (b *BuilderService) PromoteIfHealthy(ctx context.Context, id, organizationID string, stage *BuilderStage) (BuilderProject, error) {
	if err := ctx.Err(); err != nil {
		return BuilderProject{}, err
	}
	if stage == nil {
		return BuilderProject{}, errors.New("builder promotion requires a stage")
	}
	if stage.ProjectID != id {
		return BuilderProject{}, fmt.Errorf("stage belongs to project %q, not %q", stage.ProjectID, id)
	}
	if err := stage.Health(ctx); err != nil {
		return BuilderProject{}, fmt.Errorf("%w: %v", ErrBuilderPromotionUnhealthy, err)
	}
	project, err := b.GetForOrganization(id, organizationID)
	if err != nil {
		return BuilderProject{}, err
	}
	if info, err := os.Stat(stage.PublishedPath); err != nil || !info.IsDir() {
		return BuilderProject{}, fmt.Errorf("%w: versao do palco nao existe mais", ErrBuilderPromotionUnhealthy)
	}
	project.PublishedPath = stage.PublishedPath
	project.Status = "published"
	project.UpdatedAt = time.Now().UTC()
	b.mu.Lock()
	b.projects[id] = project
	err = b.persistLocked()
	b.mu.Unlock()
	return project, err
}

// builderVersionFromPublishedPath extrai o N da versao a partir de published/<id>/vN. Recusa caminho sem versao em
// vez de adivinhar: uma versao inventada no palco viraria promocao da versao errada.
func builderVersionFromPublishedPath(caminho string) (int, error) {
	base := filepath.Base(filepath.Clean(caminho))
	if !strings.HasPrefix(base, "v") {
		return 0, fmt.Errorf("published path sem versao: %s", caminho)
	}
	versao, err := strconv.Atoi(strings.TrimPrefix(base, "v"))
	if err != nil || versao < 1 {
		return 0, fmt.Errorf("published path com versao invalida: %s", caminho)
	}
	return versao, nil
}
