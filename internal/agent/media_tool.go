package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type mediaProcessTool struct {
	manager *MediaManager
}

func (mediaProcessTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{
		Name:             "media.process",
		Version:          "1",
		Description:      "Gerar ou analisar mídia usando provider explicitamente configurado; toda operação requer approval",
		Risk:             RiskExternalSideEffect,
		RequiresApproval: true,
		Scopes:           []string{"media:execute"},
	}
}

func (tool mediaProcessTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if err := validateStepID(toolContext.StepID); err != nil {
		return ToolResult{}, err
	}
	if toolContext.WorkspaceRoot == nil {
		return ToolResult{}, errors.New("pinned media workspace is required")
	}
	root := toolContext.WorkspaceRoot
	operation := strings.TrimSpace(stringInput(input, "operation", ""))
	if operation == "" {
		return ToolResult{}, errors.New("media operation is required")
	}
	var result MediaResult
	var err error
	switch operation {
	case "tone.generate":
		result, err = GenerateTone(toolContext.Workspace, floatInput(input, "frequency_hz", 440), time.Duration(intInput(input, "duration_ms", 1000))*time.Millisecond, root)
	case "audio.transcribe":
		if tool.manager == nil {
			return ToolResult{}, errors.New("media provider is not configured")
		}
		relative, data, inputErr := readApprovedMediaInput(ctx, toolContext.Workspace, input, 100<<20, root)
		if inputErr != nil {
			return ToolResult{}, inputErr
		}
		result, err = tool.manager.transcribeBytes(ctx, toolContext.Workspace, relative, data, stringInput(input, "model", ""), root)
	case "image.analyze":
		if tool.manager == nil {
			return ToolResult{}, errors.New("media provider is not configured")
		}
		_, data, inputErr := readApprovedMediaInput(ctx, toolContext.Workspace, input, 25<<20, root)
		if inputErr != nil {
			return ToolResult{}, inputErr
		}
		result, err = tool.manager.analyzeImageBytes(ctx, toolContext.Workspace, data, stringInput(input, "prompt", ""), stringInput(input, "model", ""), root)
	case "image.generate", "video.generate", "speech.generate":
		if tool.manager == nil {
			return ToolResult{}, errors.New("media provider is not configured")
		}
		switch operation {
		case "image.generate":
			result, err = tool.manager.GenerateImage(ctx, toolContext.Workspace, stringInput(input, "prompt", ""), stringInput(input, "model", ""), root)
		case "video.generate":
			result, err = tool.manager.GenerateVideo(ctx, toolContext.Workspace, stringInput(input, "prompt", ""), stringInput(input, "model", ""), root)
		case "speech.generate":
			result, err = tool.manager.GenerateSpeech(ctx, toolContext.Workspace, stringInput(input, "text", ""), stringInput(input, "voice", ""), stringInput(input, "model", ""), root)
		}
	default:
		return ToolResult{}, fmt.Errorf("media operation %q is not allowlisted", operation)
	}
	if err != nil {
		return ToolResult{}, err
	}
	result.Artifact.MissionID = toolContext.MissionID
	result.Artifact.StepID = toolContext.StepID
	return ToolResult{
		Value:     map[string]any{"operation": operation, "path": result.Path, "media_type": result.MediaType, "text": result.Text, "artifact": result.Artifact},
		Artifacts: []ArtifactManifest{result.Artifact},
	}, nil
}

func floatInput(input map[string]any, key string, fallback float64) float64 {
	value, ok := input[key]
	if !ok {
		return fallback
	}
	switch number := value.(type) {
	case float64:
		return number
	case float32:
		return float64(number)
	case int:
		return float64(number)
	case int64:
		return float64(number)
	case string:
		var parsed float64
		if _, err := fmt.Sscan(number, &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}
