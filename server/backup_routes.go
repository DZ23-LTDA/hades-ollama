package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// downloadBackup streams a local-first backup of the agent data root as a
// gzip-tar attachment (A6). Secrets (auth credentials, push tokens) are excluded
// by BackupData; the archive's SHA-256 is returned in the X-Backup-SHA256 header
// so a later restore can verify integrity.
func (a *agentAPI) downloadBackup(c *gin.Context) {
	if a.runtime == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent runtime unavailable"))
		return
	}
	// BackupData archives the entire data root, which in multi-tenant mode holds
	// every organization's data. GET is a "read" action allowed to every role,
	// and there is no cross-org super-admin, so serving a full backup under auth
	// would let any member exfiltrate other organizations' data. This is a
	// local-first, single-tenant feature; refuse it when auth is enabled.
	if a.authRequired {
		writeAgentError(c, http.StatusForbidden, errors.New("full backup is only available in local-first single-tenant mode"))
		return
	}
	var buf bytes.Buffer
	manifest, err := agent.BackupData(a.runtime.DataRoot(), &buf)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	filename := fmt.Sprintf("hades-backup-%s.tar.gz", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("X-Backup-SHA256", manifest.SHA256)
	c.Data(http.StatusOK, "application/gzip", buf.Bytes())
}
