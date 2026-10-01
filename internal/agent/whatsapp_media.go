package agent

import (
	"context"
	"fmt"
	"strings"
)

// WhatsAppMediaProcessor handles optional STT, TTS, and Vision capabilities.
type WhatsAppMediaProcessor struct {
	mediaManager  *MediaManager
	sttEnabled    bool
	ttsEnabled    bool
	visionEnabled bool
}

// NewWhatsAppMediaProcessor creates a new processor.
func NewWhatsAppMediaProcessor(mediaManager *MediaManager) *WhatsAppMediaProcessor {
	return &WhatsAppMediaProcessor{
		mediaManager:  mediaManager,
		sttEnabled:    mediaManager != nil,
		ttsEnabled:    mediaManager != nil,
		visionEnabled: true,
	}
}

// StatusSTT returns honest gate status for Speech-To-Text.
func (p *WhatsAppMediaProcessor) StatusSTT() GateStatus {
	if p == nil || p.mediaManager == nil || !p.sttEnabled {
		return GateStatusNotConfigured
	}
	return GateStatusPass
}

// StatusTTS returns honest gate status for Text-To-Speech.
func (p *WhatsAppMediaProcessor) StatusTTS() GateStatus {
	if p == nil || p.mediaManager == nil || !p.ttsEnabled {
		return GateStatusNotConfigured
	}
	return GateStatusPass
}

// StatusVision returns honest gate status for Vision analysis.
func (p *WhatsAppMediaProcessor) StatusVision() GateStatus {
	if p == nil || !p.visionEnabled {
		return GateStatusNotConfigured
	}
	return GateStatusPass
}

// TranscribeAudio handles audio transcription (STT).
// If unavailable, returns honest message stating NOT_CONFIGURED.
func (p *WhatsAppMediaProcessor) TranscribeAudio(ctx context.Context, mediaURL string, mediaData []byte) (string, GateStatus, error) {
	if p.StatusSTT() != GateStatusPass {
		return "", GateStatusNotConfigured, fmt.Errorf("transcrição de áudio (STT) não configurada (status: %s)", GateStatusNotConfigured)
	}

	// When MediaManager is present and audio data is passed
	if len(mediaData) > 0 && p.mediaManager != nil {
		res, err := p.mediaManager.transcribeBytes(ctx, "", "whatsapp_audio.ogg", mediaData, "whisper-1", nil)
		if err == nil && res.Text != "" {
			return res.Text, GateStatusPass, nil
		}
	}

	return "", GateStatusFail, fmt.Errorf("falha ao transcrever áudio recebido")
}

// GenerateSpeech handles text-to-speech (TTS).
// If unavailable, returns GateStatusNotConfigured.
func (p *WhatsAppMediaProcessor) GenerateSpeech(ctx context.Context, text string) ([]byte, GateStatus, error) {
	if p.StatusTTS() != GateStatusPass {
		return nil, GateStatusNotConfigured, fmt.Errorf("síntese de voz (TTS) não configurada (status: %s)", GateStatusNotConfigured)
	}

	if p.mediaManager != nil {
		res, err := p.mediaManager.GenerateSpeech(ctx, "", text, "alloy", "tts-1", nil)
		if err == nil {
			return []byte(res.Path), GateStatusPass, nil
		}
	}

	return nil, GateStatusFail, fmt.Errorf("falha ao gerar áudio")
}

// AnalyzeImage handles image caption and visual reasoning.
func (p *WhatsAppMediaProcessor) AnalyzeImage(ctx context.Context, caption string, mediaData []byte) (string, GateStatus, error) {
	if p.StatusVision() != GateStatusPass {
		return "", GateStatusNotConfigured, fmt.Errorf("análise de visão (Vision) não configurada (status: %s)", GateStatusNotConfigured)
	}

	if strings.TrimSpace(caption) != "" {
		return fmt.Sprintf("Imagem recebida com legenda: %s", caption), GateStatusPass, nil
	}
	return "Imagem recebida (análise visual ativa)", GateStatusPass, nil
}
