package core

import (
	"testing"
	"time"
)

func TestAgentTimeoutDefaultsAndClamp(t *testing.T) {
	cfg := AgentConfig{}
	normalizeAgentLimits(&cfg)
	if cfg.TimeoutSeconds != agentDefaultTimeout {
		t.Fatalf("default timeout=%d", cfg.TimeoutSeconds)
	}
	cfg.TimeoutSeconds = 5
	normalizeAgentLimits(&cfg)
	if cfg.TimeoutSeconds != agentMinTimeout {
		t.Fatalf("timeout should clamp to min, got %d", cfg.TimeoutSeconds)
	}
	cfg.TimeoutSeconds = 9999
	normalizeAgentLimits(&cfg)
	if cfg.TimeoutSeconds != agentMaxTimeout {
		t.Fatalf("timeout should clamp to max, got %d", cfg.TimeoutSeconds)
	}
}

func TestAgentRunTimeoutCoversRequestTimeout(t *testing.T) {
	cfg := AgentConfig{TimeoutSeconds: 300}
	if got := agentRunTimeout(cfg); got < 360*time.Second {
		t.Fatalf("run timeout should cover request timeout, got %s", got)
	}
	cfg.TimeoutSeconds = 30
	if got := agentRunTimeout(cfg); got < 240*time.Second {
		t.Fatalf("run timeout should keep the 240s floor, got %s", got)
	}
}
