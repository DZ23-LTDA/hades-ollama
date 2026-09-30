package agent

import (
	"errors"
	"strings"
)

type SlashCommand struct {
	Name      string
	Objective string
}

var ErrUnknownSlashCommand = errors.New("unknown slash command")

func ParseSlashCommand(input string) (SlashCommand, error) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return SlashCommand{}, ErrUnknownSlashCommand
	}
	name := strings.TrimPrefix(strings.ToLower(fields[0]), "/")
	switch name {
	case "goal", "plan", "test", "review":
	default:
		return SlashCommand{}, ErrUnknownSlashCommand
	}
	objective := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), fields[0]))
	if objective == "" {
		return SlashCommand{}, errors.New("slash command objective is required")
	}
	return SlashCommand{Name: name, Objective: objective}, nil
}

func SlashGoalRoles() []AgentRole {
	return []AgentRole{RoleResearch, RoleProgram, RoleTesting, RoleSecurity, RoleReview}
}
