package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrMediaInputChanged = errors.New("media input changed during approval or read")

func prepareMediaProcessApproval(workspace string, input map[string]any) (map[string]any, error) {
	if input == nil {
		return nil, errors.New("media.process input is required")
	}
	operation := strings.TrimSpace(stringInput(input, "operation", ""))
	prepared := cloneMap(input)
	prepared["operation"] = operation
	switch operation {
	case "image.generate", "video.generate":
		prompt := stringInput(input, "prompt", "")
		if strings.TrimSpace(prompt) == "" {
			return nil, fmt.Errorf("%s prompt is required", strings.TrimSuffix(operation, ".generate"))
		}
		if err := rejectSensitiveMediaText("prompt", prompt); err != nil {
			return nil, err
		}
	case "speech.generate":
		text := stringInput(input, "text", "")
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("speech text is required")
		}
		if err := rejectSensitiveMediaText("speech text", text); err != nil {
			return nil, err
		}
	case "image.analyze":
		prompt := stringInput(input, "prompt", "")
		if strings.TrimSpace(prompt) == "" {
			return nil, errors.New("vision prompt is required")
		}
		if err := rejectSensitiveMediaText("prompt", prompt); err != nil {
			return nil, err
		}
		fallthrough
	case "audio.transcribe":
		path := stringInput(input, "input_path", "")
		limit := int64(100 << 20)
		if operation == "image.analyze" {
			limit = 25 << 20
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(workspace, path)
		}
		_, data, err := readMediaInputBounded(context.Background(), workspace, path, limit)
		if err != nil {
			return nil, err
		}
		if findings := ScanDLP(string(data)); len(findings) > 0 {
			return nil, fmt.Errorf("media input contains sensitive data (%s)", findings[0].Kind)
		}
		digest := sha256.Sum256(data)
		prepared["input_path"] = filepath.ToSlash(strings.ReplaceAll(path, "\\", "/"))
		prepared["expected_input_sha256"] = hex.EncodeToString(digest[:])
		prepared["expected_input_bytes"] = len(data)
	case "tone.generate":
		frequency, err := approvedFloatInput(input, "frequency_hz", 440)
		if err != nil || math.IsNaN(frequency) || math.IsInf(frequency, 0) || frequency <= 0 || frequency > 20000 {
			return nil, errors.New("tone frequency_hz must be greater than 0 and at most 20000")
		}
		duration, err := approvedIntInput(input, "duration_ms", 1000)
		if err != nil || duration <= 0 || duration > 30000 {
			return nil, errors.New("tone duration_ms must be between 1 and 30000")
		}
		prepared["frequency_hz"] = frequency
		prepared["duration_ms"] = duration
	case "":
		return nil, errors.New("media operation is required")
	default:
		return nil, fmt.Errorf("media operation %q is not allowlisted", operation)
	}
	return prepared, nil
}

func approvedFloatInput(input map[string]any, key string, fallback float64) (float64, error) {
	value, exists := input[key]
	if !exists || value == nil {
		return fallback, nil
	}
	switch number := value.(type) {
	case float64:
		return number, nil
	case float32:
		return float64(number), nil
	case int:
		return float64(number), nil
	case int64:
		return float64(number), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(number), 64)
	default:
		return 0, fmt.Errorf("%s must be numeric", key)
	}
}

func approvedIntInput(input map[string]any, key string, fallback int) (int, error) {
	value, exists := input[key]
	if !exists || value == nil {
		return fallback, nil
	}
	switch number := value.(type) {
	case int:
		if number > 30000 || number < -30000 {
			return 0, fmt.Errorf("%s is out of range", key)
		}
		return number, nil
	case int64:
		if number > 30000 || number < -30000 {
			return 0, fmt.Errorf("%s is out of range", key)
		}
		return int(number), nil
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || math.Trunc(number) != number || number > 30000 || number < -30000 {
			return 0, fmt.Errorf("%s must be an integer", key)
		}
		return int(number), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(number))
	default:
		return 0, fmt.Errorf("%s must be an integer", key)
	}
}

func readMediaInputBounded(ctx context.Context, workspace, inputPath string, limit int64, pinnedRoots ...*os.Root) (string, []byte, error) {
	if len(pinnedRoots) > 0 {
		return readMediaInputBoundedFromRoot(ctx, workspace, pinnedRoots[0], inputPath, limit)
	}
	root, file, relative, initial, err := openSafeMediaInput(workspace, inputPath)
	if err != nil {
		return "", nil, err
	}
	defer root.Close()
	defer file.Close()
	if initial.Size() < 0 || initial.Size() > limit {
		if limit == 100<<20 {
			return "", nil, errors.New("audio input exceeds 100 MiB")
		}
		if limit == 25<<20 {
			return "", nil, errors.New("image input exceeds 25 MiB")
		}
		return "", nil, fmt.Errorf("media input exceeds %d byte limit", limit)
	}
	data, err := readMediaFile(ctx, file, limit)
	if err != nil {
		return "", nil, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(initial, after) || initial.Size() != after.Size() || !initial.ModTime().Equal(after.ModTime()) || int64(len(data)) != initial.Size() {
		return "", nil, ErrMediaInputChanged
	}
	return relative, data, nil
}

func readApprovedMediaInput(ctx context.Context, workspace string, input map[string]any, limit int64, pinnedRoots ...*os.Root) (string, []byte, error) {
	path := stringInput(input, "input_path", "")
	expected := strings.ToLower(strings.TrimSpace(stringInput(input, "expected_input_sha256", "")))
	expectedBytes, ok := input["expected_input_bytes"].(float64)
	if !ok {
		if integer, integerOK := input["expected_input_bytes"].(int); integerOK {
			expectedBytes, ok = float64(integer), true
		}
	}
	if len(expected) != 64 || !ok || expectedBytes < 0 || expectedBytes > float64(limit) {
		return "", nil, errors.New("approved media input hash is missing or invalid")
	}
	relative, data, err := readMediaInputBounded(ctx, workspace, path, limit, pinnedRoots...)
	if err != nil {
		return "", nil, err
	}
	if float64(len(data)) != expectedBytes {
		return "", nil, fmt.Errorf("%w: size differs after approval", ErrMediaInputChanged)
	}
	if findings := ScanDLP(string(data)); len(findings) > 0 {
		return "", nil, fmt.Errorf("media input contains sensitive data (%s)", findings[0].Kind)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != expected {
		return "", nil, fmt.Errorf("%w: inspect the input and approve again", ErrMediaInputChanged)
	}
	return relative, data, nil
}
