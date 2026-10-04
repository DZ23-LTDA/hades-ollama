package server

// OAuth and SAML identity routes for the agent API. Extracted from
// agent_routes.go to keep that file focused on core mission/connector routing
// (Q-01: split monolithic files). Same package, no behavior change.

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// builtinOAuthProvider holds the fixed, public OAuth endpoints for a well-known
// provider so the operator only has to supply the app's client_id/secret — not
// the URLs. These are the published authorize/token/userinfo endpoints and the
// default scopes; loopback redirect is allowed so the desktop app can receive
// the callback on 127.0.0.1.
type builtinOAuthProvider struct {
	AuthorizeURL string
	TokenURL     string
	UserInfoURL  string
	Scopes       []string
}

// builtinOAuthProviders maps a provider id to its public OAuth endpoints. One
// Google app covers Gmail, Drive, Calendar, etc. (scopes are requested per
// connection). The operator still registers the app on the provider and pastes
// the client_id/secret; nothing here fabricates credentials.
var builtinOAuthProviders = map[string]builtinOAuthProvider{
	"google": {
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
	},
	"microsoft": {
		AuthorizeURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
		Scopes:       []string{"openid", "email", "profile", "offline_access"},
	},
	"github": {
		AuthorizeURL: "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		UserInfoURL:  "https://api.github.com/user",
		Scopes:       []string{"read:user", "user:email"},
	},
	"slack": {
		AuthorizeURL: "https://slack.com/oauth/v2/authorize",
		TokenURL:     "https://slack.com/api/oauth.v2.access",
		Scopes:       []string{"users:read"},
	},
	"dropbox": {
		AuthorizeURL: "https://www.dropbox.com/oauth2/authorize",
		TokenURL:     "https://api.dropboxapi.com/oauth2/token",
		Scopes:       []string{"account_info.read"},
	},
	"canva": {
		AuthorizeURL: "https://www.canva.com/api/oauth/authorize",
		TokenURL:     "https://api.canva.com/rest/v1/oauth/token",
		Scopes:       []string{"profile:read"},
	},
	"facebook": {
		AuthorizeURL: "https://www.facebook.com/v19.0/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/v19.0/oauth/access_token",
		UserInfoURL:  "https://graph.facebook.com/me",
		Scopes:       []string{"public_profile", "email"},
	},
	"linkedin": {
		AuthorizeURL: "https://www.linkedin.com/oauth/v2/authorization",
		TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
		UserInfoURL:  "https://api.linkedin.com/v2/userinfo",
		Scopes:       []string{"openid", "profile", "email"},
	},
	"zoom": {
		AuthorizeURL: "https://zoom.us/oauth/authorize",
		TokenURL:     "https://zoom.us/oauth/token",
		UserInfoURL:  "https://api.zoom.us/v2/users/me",
		Scopes:       []string{"user:read"},
	},
	"pinterest": {
		AuthorizeURL: "https://www.pinterest.com/oauth/",
		TokenURL:     "https://api.pinterest.com/v5/oauth/token",
		UserInfoURL:  "https://api.pinterest.com/v5/user_account",
		Scopes:       []string{"user_accounts:read"},
	},
	"twitter": {
		AuthorizeURL: "https://twitter.com/i/oauth2/authorize",
		TokenURL:     "https://api.twitter.com/2/oauth2/token",
		UserInfoURL:  "https://api.twitter.com/2/users/me",
		Scopes:       []string{"tweet.read", "users.read", "offline.access"},
	},
	"salesforce": {
		AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
		TokenURL:     "https://login.salesforce.com/services/oauth2/token",
		UserInfoURL:  "https://login.salesforce.com/services/oauth2/userinfo",
		Scopes:       []string{"openid", "email", "api", "refresh_token"},
	},
	"zoho": {
		AuthorizeURL: "https://accounts.zoho.com/oauth/v2/auth",
		TokenURL:     "https://accounts.zoho.com/oauth/v2/token",
		Scopes:       []string{"AaaServer.profile.READ"},
	},
	"mercado-livre": {
		AuthorizeURL: "https://auth.mercadolivre.com.br/authorization",
		TokenURL:     "https://api.mercadolibre.com/oauth/token",
		UserInfoURL:  "https://api.mercadolibre.com/users/me",
		Scopes:       []string{"read"},
	},
	"tiktok-business": {
		AuthorizeURL: "https://www.tiktok.com/v2/auth/authorize/",
		TokenURL:     "https://open.tiktokapis.com/v2/oauth/token/",
		Scopes:       []string{"user.info.basic"},
	},
	"notion": {
		AuthorizeURL: "https://api.notion.com/v1/oauth/authorize",
		TokenURL:     "https://api.notion.com/v1/oauth/token",
		Scopes:       []string{},
	},
	"hubspot": {
		AuthorizeURL: "https://app.hubspot.com/oauth/authorize",
		TokenURL:     "https://api.hubapi.com/oauth/v1/token",
		Scopes:       []string{"oauth"},
	},
	"gitlab": {
		AuthorizeURL: "https://gitlab.com/oauth/authorize",
		TokenURL:     "https://gitlab.com/oauth/token",
		UserInfoURL:  "https://gitlab.com/oauth/userinfo",
		Scopes:       []string{"read_user"},
	},
	"discord": {
		AuthorizeURL: "https://discord.com/oauth2/authorize",
		TokenURL:     "https://discord.com/api/oauth2/token",
		UserInfoURL:  "https://discord.com/api/users/@me",
		Scopes:       []string{"identify", "email"},
	},
}

// builtinOAuthScopes returns the default scopes for a known provider id.
func builtinOAuthScopes(name string) []string {
	if b, ok := builtinOAuthProviders[strings.ToLower(strings.TrimSpace(name))]; ok {
		return b.Scopes
	}
	return nil
}

func oauthProviderFromEnv(name string) agent.OAuthProvider {
	key := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(name)))
	prefix := "OLLAMA_AGENT_OAUTH_" + key
	redirects := make([]string, 0)
	for _, value := range strings.FieldsFunc(os.Getenv(prefix+"_REDIRECT_URIS"), func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if value = strings.TrimSpace(value); value != "" {
			redirects = append(redirects, value)
		}
	}
	authorizeURL := os.Getenv(prefix + "_AUTHORIZE_URL")
	tokenURL := os.Getenv(prefix + "_TOKEN_URL")
	userInfoURL := os.Getenv(prefix + "_USERINFO_URL")
	allowLoopback := strings.EqualFold(os.Getenv(prefix+"_ALLOW_LOOPBACK_REDIRECT"), "true")
	clientIDEnv := os.Getenv(prefix + "_CLIENT_ID_ENV")
	secretEnv := os.Getenv(prefix + "_SECRET_ENV")
	// Fall back to the built-in public endpoints for well-known providers so the
	// operator only supplies the client_id/secret. Explicit env overrides win.
	if builtin, ok := builtinOAuthProviders[strings.ToLower(strings.TrimSpace(name))]; ok {
		if authorizeURL == "" {
			authorizeURL = builtin.AuthorizeURL
		}
		if tokenURL == "" {
			tokenURL = builtin.TokenURL
		}
		if userInfoURL == "" {
			userInfoURL = builtin.UserInfoURL
		}
		// A desktop app registered on these providers receives the callback on a
		// loopback address, so allow it by default for the built-ins.
		allowLoopback = true
		if clientIDEnv == "" {
			clientIDEnv = prefix + "_CLIENT_ID"
		}
		if secretEnv == "" {
			secretEnv = prefix + "_SECRET"
		}
	}
	// Even for providers without a built-in profile, default the credential
	// variable names to the canonical ones so the file-backed OAuth client
	// store (ensureOAuthClientCredentials) has a stable place to publish the
	// client_id/secret. An explicit *_CLIENT_ID_ENV / *_SECRET_ENV override
	// still wins.
	if clientIDEnv == "" {
		clientIDEnv = prefix + "_CLIENT_ID"
	}
	if secretEnv == "" {
		secretEnv = prefix + "_SECRET"
	}
	return agent.OAuthProvider{Name: name, AuthorizeURL: authorizeURL, TokenURL: tokenURL, RevocationURL: os.Getenv(prefix + "_REVOCATION_URL"), UserInfoURL: userInfoURL, IssuerURL: os.Getenv(prefix + "_ISSUER_URL"), Audience: os.Getenv(prefix + "_AUDIENCE"), ClientIDEnv: clientIDEnv, SecretEnv: secretEnv, RedirectURIs: redirects, AllowLoopbackRedirect: allowLoopback}
}

func prepareOIDCProvider(ctx context.Context, provider agent.OAuthProvider) (agent.OAuthProvider, error) {
	if strings.TrimSpace(provider.IssuerURL) == "" {
		return provider, provider.Validate()
	}
	discovery, err := provider.Discover(ctx, newServerEgressClient("server.agent.provider_discovery", false))
	if err != nil {
		return agent.OAuthProvider{}, err
	}
	if provider.AuthorizeURL == "" {
		provider.AuthorizeURL = discovery.AuthorizationEndpoint
	}
	if provider.TokenURL == "" {
		provider.TokenURL = discovery.TokenEndpoint
	}
	if provider.UserInfoURL == "" {
		provider.UserInfoURL = discovery.UserInfoEndpoint
	}
	return provider, provider.Validate()
}

func samlProviderFromEnv(name string) agent.SAMLProviderConfig {
	key := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(name)))
	prefix := "OLLAMA_AGENT_SAML_" + key
	return agent.SAMLProviderConfig{
		Name:               name,
		EntityID:           os.Getenv(prefix + "_ENTITY_ID"),
		IDPMetadataURL:     os.Getenv(prefix + "_IDP_METADATA_URL"),
		MetadataURL:        os.Getenv(prefix + "_METADATA_URL"),
		ACSURL:             os.Getenv(prefix + "_ACS_URL"),
		SPPrivateKeyFile:   os.Getenv(prefix + "_SP_PRIVATE_KEY_FILE"),
		SPCertificateFile:  os.Getenv(prefix + "_SP_CERTIFICATE_FILE"),
		DefaultRedirectURI: os.Getenv(prefix + "_DEFAULT_REDIRECT_URI"),
		AllowIDPInitiated:  strings.EqualFold(os.Getenv(prefix+"_ALLOW_IDP_INITIATED"), "true"),
	}
}

func loadSAMLService(ctx context.Context, name string) (*agent.SAMLService, error) {
	config := samlProviderFromEnv(name)
	return agent.NewSAMLService(ctx, config, newServerEgressClient("server.agent.saml", false))
}

func (a *agentAPI) samlService(ctx context.Context, name string) (*agent.SAMLService, error) {
	a.samlMu.Lock()
	defer a.samlMu.Unlock()
	if service, ok := a.samlServices[name]; ok {
		return service, nil
	}
	service, err := loadSAMLService(ctx, name)
	if err != nil {
		return nil, err
	}
	a.samlServices[name] = service
	return service, nil
}

func (a *agentAPI) oauthStart(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(c.Query("redirect_uri"))
	verifier := strings.TrimSpace(c.Query("code_verifier"))
	if redirectErr != nil || verifier == "" {
		if redirectErr == nil {
			redirectErr = errors.New("redirect_uri and PKCE code_verifier are required")
		}
		writeAgentError(c, http.StatusBadRequest, redirectErr)
		return
	}
	userID := ""
	if value, ok := c.Get("agent.user"); ok {
		if user, ok := value.(agent.User); ok {
			userID = user.ID
		}
	}
	nonce := fmt.Sprintf("%x", sha256.Sum256([]byte(provider.Name+"|"+redirectURI+"|"+verifier+"|"+time.Now().UTC().String())))
	state, _, err := a.auth.CreateOAuthStateWithNonce(provider.Name, redirectURI, verifier, nonce, userID, 5*time.Minute)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	scopes := strings.Fields(c.Query("scope"))
	if len(scopes) == 0 {
		scopes = []string{"openid", "email"}
	}
	authorizationURL, err := provider.AuthorizationURLWithNonce(state, nonce, scopes)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": provider.Name, "authorization_url": authorizationURL, "state": state, "expires_in": 300})
}

func (a *agentAPI) oauthCallback(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(c.Query("redirect_uri"))
	code := strings.TrimSpace(c.Query("code"))
	stateValue := strings.TrimSpace(c.Query("state"))
	if redirectErr != nil || code == "" || stateValue == "" {
		if redirectErr == nil {
			redirectErr = errors.New("code, state and redirect_uri are required")
		}
		writeAgentError(c, http.StatusBadRequest, redirectErr)
		return
	}
	state, err := a.auth.ConsumeOAuthState(stateValue, provider.Name, redirectURI)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	payload, err := provider.ExchangeCode(c.Request.Context(), newServerEgressClient("server.agent.oauth.exchange", false), code, redirectURI, state.CodeVerifier)
	if err != nil {
		writeAgentError(c, http.StatusBadGateway, err)
		return
	}
	if provider.IssuerURL != "" {
		idToken, _ := payload["id_token"].(string)
		if idToken == "" {
			writeAgentError(c, http.StatusBadGateway, errors.New("oidc token response has no id_token"))
			return
		}
		claims, validationErr := provider.ValidateIDToken(c.Request.Context(), newServerEgressClient("server.agent.oauth.id_token", false), idToken, state.Nonce)
		if validationErr != nil {
			writeAgentError(c, http.StatusBadGateway, validationErr)
			return
		}
		payload["id_token_claims"] = claims
	}
	userID := state.UserID
	if userID == "" && (provider.UserInfoURL != "" || provider.IssuerURL != "") {
		accessToken, _ := payload["access_token"].(string)
		userinfo, userinfoErr := provider.FetchUserInfo(c.Request.Context(), newServerEgressClient("server.agent.oauth.userinfo", false), accessToken)
		if userinfoErr != nil {
			writeAgentError(c, http.StatusBadGateway, userinfoErr)
			return
		}
		payload["userinfo"] = userinfo
	}
	if userID == "" {
		user, _, _, provisionErr := a.auth.ProvisionOAuthUser(payload, provider.Name)
		if provisionErr != nil {
			writeAgentError(c, http.StatusBadRequest, provisionErr)
			return
		}
		userID = user.ID
	}
	organization, _, err := a.auth.FirstOrganization(userID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, errors.New("OAuth user has no organization"))
		return
	}
	credential, err := a.auth.StoreOAuthCredential(provider.Name, userID, organization.ID, payload)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	localToken, session, err := a.auth.IssueToken(userID, organization.ID, 24*time.Hour)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": localToken, "token": session, "credential_id": credential.ID, "provider": provider.Name, "organization": organization, "expires_at": credential.ExpiresAt})
}

func oauthCredentialPublic(credential agent.OAuthCredential) gin.H {
	return gin.H{
		"credential_id": credential.ID,
		"provider":      credential.Provider,
		"expires_at":    credential.ExpiresAt,
		"updated_at":    credential.UpdatedAt,
		"revoked_at":    credential.RevokedAt,
	}
}

func (a *agentAPI) oauthRefresh(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var request struct {
		CredentialID string `json:"credential_id"`
	}
	if err := decodeJSON(c, &request); err != nil || strings.TrimSpace(request.CredentialID) == "" {
		if err == nil {
			err = errors.New("credential_id is required")
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	credential, err := a.auth.RefreshOAuthCredentialForOrganization(c.Request.Context(), agentOrganizationID(c), provider, request.CredentialID, newServerEgressClient("server.agent.oauth.refresh", false))
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, agent.ErrOAuthCredentialConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, agent.ErrOAuthCredentialRevoked) || strings.Contains(err.Error(), "outside the active organization") {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.JSON(http.StatusOK, oauthCredentialPublic(credential))
}

func (a *agentAPI) oauthRevoke(c *gin.Context) {
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(c.Param("provider")))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var request struct {
		CredentialID string `json:"credential_id"`
	}
	if err := decodeJSON(c, &request); err != nil || strings.TrimSpace(request.CredentialID) == "" {
		if err == nil {
			err = errors.New("credential_id is required")
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if err := a.auth.RevokeOAuthCredentialForOrganization(c.Request.Context(), agentOrganizationID(c), request.CredentialID, provider, newServerEgressClient("server.agent.oauth.revoke", false)); err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, agent.ErrOAuthCredentialConflict) {
			status = http.StatusConflict
		} else if errors.Is(err, agent.ErrOAuthCredentialRevoked) || strings.Contains(err.Error(), "outside the active organization") {
			status = http.StatusForbidden
		}
		writeAgentError(c, status, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// connectorOAuthProvider maps a catalog connector id to the OAuth provider that
// issues its tokens. The Google services share a single Google OAuth app;
// every other connector authenticates against a provider named after itself.
func connectorOAuthProvider(connectorID string) string {
	switch strings.ToLower(strings.TrimSpace(connectorID)) {
	case "gmail", "google-drive", "google-calendar", "google-analytics", "google-ads", "google-cloud", "youtube", "google-workspace":
		return "google"
	case "azure", "outlook", "microsoft-365", "onedrive":
		return "microsoft"
	case "facebook-pages", "instagram", "meta-ads":
		// Meta's platform uses a single Facebook Login OAuth app for Pages,
		// Instagram and the Marketing (Ads) API.
		return "facebook"
	case "x-twitter":
		return "twitter"
	default:
		return strings.ToLower(strings.TrimSpace(connectorID))
	}
}

// connectorOAuthAPIBaseURLs pins the REST base URL used when a connector is
// registered through OAuth and has no quick-connect base in the catalog.
var connectorOAuthAPIBaseURLs = map[string]string{
	"gmail":            "https://gmail.googleapis.com",
	"google-drive":     "https://www.googleapis.com",
	"google-workspace": "https://www.googleapis.com",
	"google-calendar":  "https://www.googleapis.com/calendar/v3",
	"google-analytics": "https://analyticsdata.googleapis.com",
	"google-ads":       "https://googleads.googleapis.com",
	"youtube":          "https://www.googleapis.com/youtube/v3",
	"outlook":          "https://graph.microsoft.com/v1.0",
}

// providerOAuthAPIBaseURLs is the last-resort REST base keyed by provider.
var providerOAuthAPIBaseURLs = map[string]string{
	"google":    "https://www.googleapis.com",
	"microsoft": "https://graph.microsoft.com/v1.0",
	"github":    "https://api.github.com",
	"slack":     "https://slack.com/api",
	"dropbox":   "https://api.dropboxapi.com/2",
	"canva":     "https://api.canva.com/rest/v1",
}

// connectorOAuthExtraScopes are read-only scopes requested in addition to the
// provider's default scopes when connecting a specific service.
var connectorOAuthExtraScopes = map[string][]string{
	"gmail":            {"https://www.googleapis.com/auth/gmail.readonly"},
	"google-drive":     {"https://www.googleapis.com/auth/drive.readonly"},
	"google-calendar":  {"https://www.googleapis.com/auth/calendar.readonly"},
	"google-analytics": {"https://www.googleapis.com/auth/analytics.readonly"},
	"google-ads":       {"https://www.googleapis.com/auth/adwords"},
	"youtube":          {"https://www.googleapis.com/auth/youtube.readonly"},
}

func connectorOAuthScopes(connectorID, providerName string) []string {
	scopes := make([]string, 0, 6)
	seen := map[string]bool{}
	add := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			scopes = append(scopes, value)
		}
	}
	add(builtinOAuthScopes(providerName)...)
	add(connectorOAuthExtraScopes[strings.ToLower(strings.TrimSpace(connectorID))]...)
	if len(scopes) == 0 {
		add("openid", "email")
	}
	return scopes
}

func connectorOAuthBaseURL(connectorID, providerName string) string {
	id := strings.ToLower(strings.TrimSpace(connectorID))
	if base := connectorOAuthAPIBaseURLs[id]; base != "" {
		return base
	}
	for _, entry := range agent.ConnectorCatalog() {
		if entry.ID == id && strings.TrimSpace(entry.APIBaseURL) != "" {
			return entry.APIBaseURL
		}
	}
	return providerOAuthAPIBaseURLs[strings.ToLower(strings.TrimSpace(providerName))]
}

func connectorOAuthOperations(connectorID string) []agent.ConnectorOperation {
	if operations := agent.QuickConnectOperations(connectorID); len(operations) > 0 {
		return operations
	}
	// Least-privilege default: allow only read (GET) requests until an operator
	// configures a richer policy through the advanced connector registration.
	return []agent.ConnectorOperation{{Name: "read", Methods: []string{"GET"}, PathPrefixes: []string{"/"}}}
}

// ensureOAuthClientCredentials publishes the provider's stored app
// client_id/secret into the environment variables the OAuth flow reads
// (ClientIDEnv/SecretEnv). An explicit OS environment value always wins; only
// an empty variable is filled from the file store. The secret is never logged.
func (a *agentAPI) ensureOAuthClientCredentials(providerName string) {
	if a == nil || a.oauthClients == nil {
		return
	}
	clientID, secret, ok := a.oauthClients.Get(providerName)
	if !ok {
		return
	}
	provider := oauthProviderFromEnv(providerName)
	if provider.ClientIDEnv != "" && strings.TrimSpace(os.Getenv(provider.ClientIDEnv)) == "" {
		_ = os.Setenv(provider.ClientIDEnv, clientID)
	}
	if provider.SecretEnv != "" && strings.TrimSpace(os.Getenv(provider.SecretEnv)) == "" {
		_ = os.Setenv(provider.SecretEnv, secret)
	}
}

// oauthClientConfigured reports whether the provider has an app client_id and
// secret available, either from the file store or the OS environment.
func (a *agentAPI) oauthClientConfigured(providerName string) bool {
	if a != nil && a.oauthClients != nil && a.oauthClients.Configured(providerName) {
		return true
	}
	provider := oauthProviderFromEnv(providerName)
	return provider.ClientIDEnv != "" && provider.SecretEnv != "" &&
		strings.TrimSpace(os.Getenv(provider.ClientIDEnv)) != "" &&
		strings.TrimSpace(os.Getenv(provider.SecretEnv)) != ""
}

// saveOAuthClient stores the app client_id/secret for a provider. Admin only.
// The secret is never returned.
func (a *agentAPI) saveOAuthClient(c *gin.Context) {
	if !a.requireApprovalApprover(c) {
		return
	}
	provider := strings.ToLower(strings.TrimSpace(c.Param("provider")))
	if provider == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("provider é obrigatório"))
		return
	}
	if a.oauthClients == nil {
		writeAgentError(c, http.StatusInternalServerError, errors.New("armazenamento de credenciais OAuth indisponível"))
		return
	}
	var request struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(request.ClientID) == "" || strings.TrimSpace(request.ClientSecret) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("client_id e client_secret são obrigatórios"))
		return
	}
	if err := a.oauthClients.Save(provider, request.ClientID, request.ClientSecret); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved", "provider": provider, "configured": true})
}

// listOAuthClients lists the configured OAuth providers (names only, never
// secrets), including the known built-ins with their configuration flag.
func (a *agentAPI) listOAuthClients(c *gin.Context) {
	if !a.requireApprovalApprover(c) {
		return
	}
	type providerStatus struct {
		Provider   string `json:"provider"`
		Configured bool   `json:"configured"`
		BuiltIn    bool   `json:"built_in"`
	}
	seen := map[string]bool{}
	result := make([]providerStatus, 0)
	for name := range builtinOAuthProviders {
		result = append(result, providerStatus{Provider: name, Configured: a.oauthClientConfigured(name), BuiltIn: true})
		seen[name] = true
	}
	if a.oauthClients != nil {
		for _, name := range a.oauthClients.Providers() {
			if !seen[name] {
				result = append(result, providerStatus{Provider: name, Configured: true, BuiltIn: false})
				seen[name] = true
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Provider < result[j].Provider })
	c.JSON(http.StatusOK, gin.H{"providers": result})
}

// connectorOAuthStart begins connecting a connector through OAuth: it ensures
// the provider's app credentials are available, mints a bound state, and
// returns the authorization URL the desktop opens in the browser.
func (a *agentAPI) connectorOAuthStart(c *gin.Context) {
	connectorID := strings.TrimSpace(c.Param("id"))
	if connectorID == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("connector id é obrigatório"))
		return
	}
	providerName := connectorOAuthProvider(connectorID)
	a.ensureOAuthClientCredentials(providerName)
	if !a.oauthClientConfigured(providerName) {
		writeAgentError(c, http.StatusPreconditionFailed, errors.New("configure o client_id/secret deste provedor primeiro"))
		return
	}
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(providerName))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	var request struct {
		RedirectURI  string `json:"redirect_uri"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(request.RedirectURI)
	verifier := strings.TrimSpace(request.CodeVerifier)
	if redirectErr != nil || verifier == "" {
		if redirectErr == nil {
			redirectErr = errors.New("redirect_uri e code_verifier (PKCE) são obrigatórios")
		}
		writeAgentError(c, http.StatusBadRequest, redirectErr)
		return
	}
	userID := ""
	if value, ok := c.Get("agent.user"); ok {
		if user, ok := value.(agent.User); ok {
			userID = user.ID
		}
	}
	nonce := fmt.Sprintf("%x", sha256.Sum256([]byte(provider.Name+"|"+connectorID+"|"+redirectURI+"|"+verifier+"|"+time.Now().UTC().String())))
	state, _, err := a.auth.CreateOAuthStateLoopback(provider.Name, redirectURI, verifier, nonce, userID, 5*time.Minute, provider.AllowLoopbackRedirect)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	authorizationURL, err := provider.AuthorizationURLWithNonce(state, nonce, connectorOAuthScopes(connectorID, provider.Name))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	// Request a long-lived connection: without offline access the provider only
	// returns a short-lived access token and the connector would stop working
	// after ~1h. Google needs access_type=offline (+ prompt=consent to re-issue a
	// refresh token on reconnection).
	authorizationURL = withOfflineAccessParams(provider.Name, authorizationURL)
	c.JSON(http.StatusOK, gin.H{"connector": connectorID, "provider": provider.Name, "authorization_url": authorizationURL, "state": state, "expires_in": 300})
}

// withOfflineAccessParams adds the provider-specific query parameters needed to
// obtain a refresh token, so a connected account keeps working after the initial
// access token expires. Unknown providers are returned unchanged.
func withOfflineAccessParams(providerName, rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := u.Query()
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "google":
		query.Set("access_type", "offline")
		query.Set("prompt", "consent")
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// connectorOAuthCallback completes the connector OAuth flow: it exchanges the
// code for a token, stores the credential for the active organization, and
// registers the connector with that OAuth provider. It answers with a simple
// HTML page because the browser (not the SPA) lands here.
func (a *agentAPI) connectorOAuthCallback(c *gin.Context) {
	connectorID := strings.TrimSpace(c.Param("id"))
	providerName := connectorOAuthProvider(connectorID)
	a.ensureOAuthClientCredentials(providerName)
	provider, err := prepareOIDCProvider(c.Request.Context(), oauthProviderFromEnv(providerName))
	if err != nil {
		writeConnectorOAuthHTML(c, http.StatusBadRequest, "Não foi possível preparar o provedor OAuth. Verifique o client_id/secret configurado.")
		return
	}
	redirectURI, redirectErr := provider.NormalizeRedirectURI(c.Query("redirect_uri"))
	code := strings.TrimSpace(c.Query("code"))
	stateValue := strings.TrimSpace(c.Query("state"))
	if redirectErr != nil || code == "" || stateValue == "" {
		writeConnectorOAuthHTML(c, http.StatusBadRequest, "Parâmetros de retorno inválidos (code, state e redirect_uri são obrigatórios).")
		return
	}
	state, err := a.auth.ConsumeOAuthState(stateValue, provider.Name, redirectURI)
	if err != nil {
		writeConnectorOAuthHTML(c, http.StatusBadRequest, "Sessão de autorização inválida, expirada ou já usada. Tente conectar novamente.")
		return
	}
	payload, err := provider.ExchangeCode(c.Request.Context(), newServerEgressClient("server.agent.connector.oauth.exchange", false), code, redirectURI, state.CodeVerifier)
	if err != nil {
		writeConnectorOAuthHTML(c, http.StatusBadGateway, "Falha ao trocar o código de autorização com o provedor.")
		return
	}
	organizationID := agentOrganizationID(c)
	if _, err := a.auth.StoreOAuthCredential(provider.Name, state.UserID, organizationID, payload); err != nil {
		writeConnectorOAuthHTML(c, http.StatusBadRequest, "Não foi possível salvar a credencial da conta para esta organização.")
		return
	}
	if err := a.registerConnectorOAuth(organizationID, connectorID, provider.Name); err != nil {
		writeConnectorOAuthHTML(c, http.StatusBadRequest, "A conta foi autorizada, mas o conector não pôde ser registrado: "+err.Error())
		return
	}
	slog.Info("connector connected via oauth", "connector", connectorID, "provider", provider.Name)
	writeConnectorOAuthHTML(c, http.StatusOK, "")
}

// registerConnectorOAuth registers (or refreshes) the catalog connector so it
// resolves its token through the OAuth provider for the active organization.
func (a *agentAPI) registerConnectorOAuth(organizationID, connectorID, providerName string) error {
	baseURL := connectorOAuthBaseURL(connectorID, providerName)
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("endereço de API desconhecido para o conector %q", connectorID)
	}
	config := agent.ConnectorConfig{
		ID:            connectorID,
		Provider:      connectorID,
		BaseURL:       baseURL,
		OAuthProvider: providerName,
		Operations:    connectorOAuthOperations(connectorID),
	}
	if a.authRequired {
		return a.runtime.RegisterConnectorForOrganization(organizationID, config)
	}
	return a.runtime.RegisterConnector(config)
}

// writeConnectorOAuthHTML renders the browser-facing result page in pt-BR. An
// empty message renders the success page.
func writeConnectorOAuthHTML(c *gin.Context, status int, message string) {
	title := "Conta conectada!"
	body := "Você já pode fechar esta aba e voltar ao Hades."
	if strings.TrimSpace(message) != "" {
		title = "Não foi possível conectar"
		body = message
	}
	page := "<!doctype html><html lang=\"pt-BR\"><head><meta charset=\"utf-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">" +
		"<title>" + html.EscapeString(title) + "</title></head>" +
		"<body style=\"font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;" +
		"display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0;background:#0b0b0f;color:#f5f5f7\">" +
		"<main style=\"max-width:28rem;padding:2rem;text-align:center\">" +
		"<h1 style=\"font-size:1.25rem;margin:0 0 .75rem\">" + html.EscapeString(title) + "</h1>" +
		"<p style=\"margin:0;color:#b8b8c0;line-height:1.5\">" + html.EscapeString(body) + "</p>" +
		"</main></body></html>"
	c.Data(status, "text/html; charset=utf-8", []byte(page))
}
