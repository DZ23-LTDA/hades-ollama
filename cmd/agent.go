package cmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/internal/agent"
)

func agentCommand() *cobra.Command {
	command := &cobra.Command{Use: "agent", Short: "Create and control agentic missions"}

	var objective, model, workspace, project string
	var autoRun bool
	create := &cobra.Command{
		Use:   "create",
		Short: "Create an agentic mission",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAgentRequest(cmd.Context(), http.MethodPost, "/api/agent/v1/missions", map[string]any{
				"objective":  objective,
				"model":      model,
				"workspace":  workspace,
				"project_id": project,
				"auto_run":   autoRun,
			})
		},
	}
	create.Flags().StringVar(&objective, "objective", "", "Mission objective")
	create.Flags().StringVar(&model, "model", "", "Planner model (optional)")
	create.Flags().StringVar(&workspace, "workspace", "", "Authorized workspace path")
	create.Flags().StringVar(&project, "project", "", "Project identifier")
	create.Flags().BoolVar(&autoRun, "auto-run", false, "Run immediately when no approval is required")
	_ = create.MarkFlagRequired("objective")

	idCommand := func(use, short, method, suffix string, body func() any) *cobra.Command {
		return &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentRequest(cmd.Context(), method, "/api/agent/v1/missions/"+url.PathEscape(args[0])+suffix, body())
		}}
	}
	get := idCommand("get MISSION_ID", "Show a mission", http.MethodGet, "", func() any { return nil })
	run := idCommand("run MISSION_ID", "Run a mission", http.MethodPost, "/run", func() any { return map[string]any{} })
	cancel := idCommand("cancel MISSION_ID", "Cancel a mission", http.MethodPost, "/cancel", func() any { return map[string]any{} })
	events := idCommand("events MISSION_ID", "Show mission events", http.MethodGet, "/events", func() any { return nil })
	tools := &cobra.Command{Use: "tools", Short: "List agent tools", RunE: func(cmd *cobra.Command, _ []string) error {
		return runAgentRequest(cmd.Context(), http.MethodGet, "/api/agent/v1/tools", nil)
	}}
	var approved bool
	var reason string
	approve := &cobra.Command{Use: "approve MISSION_ID APPROVAL_ID", Short: "Approve or reject a protected step", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		return runAgentRequest(cmd.Context(), http.MethodPost, "/api/agent/v1/missions/"+url.PathEscape(args[0])+"/approvals/"+url.PathEscape(args[1]), map[string]any{"approved": approved, "reason": reason})
	}}
	approve.Flags().BoolVar(&approved, "approved", false, "Approve the step; omit to reject")
	approve.Flags().StringVar(&reason, "reason", "", "Decision reason")

	migratePostgres := &cobra.Command{
		Use:   "migrate-postgres",
		Short: "Apply agent database schema using the separate migrator credentials",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dsn := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL"))
			key, err := hex.DecodeString(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY")))
			if err != nil || len(key) < 32 {
				return fmt.Errorf("OLLAMA_AGENT_TENANT_CONTEXT_KEY must be at least 64 hexadecimal characters")
			}
			version, err := postgresTenantKeyVersionFromEnv()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			return agent.MigratePostgresAgentSchemaVersioned(ctx, dsn, key, version)
		},
	}
	rotatePostgresKey := &cobra.Command{
		Use:   "rotate-postgres-key",
		Short: "Fence runtime logins and atomically rotate the PostgreSQL tenant HMAC key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			adminDSN := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_POSTGRES_ADMIN_DATABASE_URL"))
			dsn := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_MIGRATOR_DATABASE_URL"))
			currentKey, err := hex.DecodeString(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY")))
			if err != nil || len(currentKey) < 32 {
				return errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY must be at least 64 hexadecimal characters")
			}
			currentVersion, err := postgresTenantKeyVersionFromEnv()
			if err != nil {
				return err
			}
			nextVersionRaw := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION"))
			nextVersion, err := strconv.Atoi(nextVersionRaw)
			if err != nil || nextVersion < 1 || nextVersion > 999999999 {
				return errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT_VERSION must be an integer between 1 and 999999999")
			}
			nextKey, err := hex.DecodeString(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT")))
			if err != nil || len(nextKey) < 32 {
				return errors.New("OLLAMA_AGENT_TENANT_CONTEXT_KEY_NEXT must be at least 64 hexadecimal characters")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			return agent.RotatePostgresAgentTenantKey(ctx, adminDSN, dsn, currentVersion, currentKey, nextVersion, nextKey)
		},
	}
	command.AddCommand(create, get, run, cancel, events, approve, tools, migratePostgres, rotatePostgresKey)
	return command
}

func postgresTenantKeyVersionFromEnv() (int, error) {
	return agent.TenantContextKeyVersionFromEnv()
}

func runAgentRequest(ctx context.Context, method, path string, payload any) error {
	base := envconfig.Host()
	base.Path = strings.TrimSuffix(base.Path, "/")
	base.Path += path
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, base.String(), body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("agent API %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if len(data) == 0 {
		return nil
	}
	var pretty any
	if err := json.Unmarshal(data, &pretty); err == nil {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(pretty)
	}
	_, err = os.Stdout.Write(data)
	return err
}
