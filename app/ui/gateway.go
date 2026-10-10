//go:build windows || darwin

package ui

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/ollama/ollama/app/secrets"
	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/multillm"
)

// gatewayConnection é o bloco "Configurar inferência de terceiros": endereço
// base, protocolos aceitos, rotação automática e a chave do gateway. A chave só
// é preenchida para um pedido da própria máquina.
type gatewayConnection struct {
	multillm.GatewayInfo
	RestartRequired bool `json:"restart_required,omitempty"`
}

// requestIsLocal reports whether the request came from this computer. The
// gateway key is a local secret: only a loopback caller may read or rotate it.
func requestIsLocal(r *http.Request) bool {
	if r == nil {
		return false
	}
	addr := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return isLoopbackHost(host)
}

// gatewayInfo builds the connection block from the provider config already on
// disk. The base URL is the same address the native Ollama server listens on,
// because that is where /v1/messages and /v1/responses are served.
func gatewayInfo(cfg multillm.Config, reveal bool) multillm.GatewayInfo {
	base := ""
	if host := envconfig.ConnectableHost(); host != nil {
		base = host.String()
	}
	return multillm.DescribeGateway(cfg, nil, base, providerConfigPath(), reveal)
}

// getGatewayConnection serves GET /api/v1/gateway/connection.
func (s *Server) getGatewayConnection(w http.ResponseWriter, r *http.Request) error {
	AdoptProviderCredentials()
	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(gatewayConnection{GatewayInfo: gatewayInfo(cfg, requestIsLocal(r))})
}

// newGatewayKey generates the bearer token the gateway expects from clients
// outside this computer. 32 random bytes keep it out of reach of guessing.
func newGatewayKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerar a chave do gateway: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// rotateGatewayKey serves POST /api/v1/gateway/key: it generates a new gateway
// key, stores it in the OS credential vault and, when the config never named an
// environment for it, records the default name so the next start of the server
// requires the key from remote callers. The response carries restart_required,
// because the running server only reads gateway_api_key_env at startup.
func (s *Server) rotateGatewayKey(w http.ResponseWriter, r *http.Request) error {
	if !requestIsLocal(r) {
		w.WriteHeader(http.StatusForbidden)
		return errors.New("a chave do gateway só pode ser gerada nesta máquina")
	}
	AdoptProviderCredentials()
	cfg, err := loadProviderConfig()
	if err != nil {
		return err
	}
	path := providerConfigPath()
	if path == "" {
		w.WriteHeader(http.StatusInternalServerError)
		return errors.New("OLLAMA_DZ23_CONFIG não está definido; instale novamente o aplicativo para gerar a chave do gateway")
	}

	restartRequired := false
	if strings.TrimSpace(cfg.GatewayAPIKeyEnv) == "" {
		cfg.GatewayAPIKeyEnv = multillm.DefaultGatewayKeyEnv
		if err := writeProviderConfig(path, cfg); err != nil {
			return err
		}
		restartRequired = true
	}
	envName := strings.TrimSpace(cfg.GatewayAPIKeyEnv)

	dir, err := providerSecretsDir()
	if err != nil {
		return err
	}
	key, err := newGatewayKey()
	if err != nil {
		return err
	}
	if _, err := secrets.Save(dir, envName, key); err != nil {
		if errors.Is(err, secrets.ErrInvalid) {
			w.WriteHeader(http.StatusBadRequest)
			return err
		}
		// The vault file is written before the user environment is updated and
		// multillm finds it by name even without the variable, so a rejected
		// environment update must not discard a key that already works. The
		// error from the vault never carries the key itself.
		s.log().Warn("gateway key saved without updating the user environment", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(gatewayConnection{
		GatewayInfo:     gatewayInfo(cfg, true),
		RestartRequired: restartRequired,
	})
}
