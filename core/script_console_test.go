package core

import (
	"strings"
	"testing"
)

func TestScriptConsoleTokenStableAndEncrypted(t *testing.T) {
	setupAgentBucket(t, "script-console-token")
	resetScriptConsoleTokenForTest()
	t.Cleanup(resetScriptConsoleTokenForTest)

	token := scriptConsoleToken()
	if len(token) < 32 {
		t.Fatalf("token too short: %q", token)
	}
	if !ValidateScriptConsolePassword(token) {
		t.Fatal("generated token should validate")
	}
	if ValidateScriptConsolePassword("wrong-token") {
		t.Fatal("wrong token must not validate")
	}
	raw := BucketApp.GetString(scriptConsoleTokenKey)
	if raw == "" || raw == token || !strings.HasPrefix(raw, secretEncPrefix) {
		t.Fatalf("token should be encrypted at rest, got %q", raw)
	}

	env := scriptConsoleEnv()
	if env["JIYING_CONSOLE_USER"] != ScriptConsoleUser || env["JIYING_CONSOLE_PASS"] != token {
		t.Fatalf("unexpected console env: %+v", env)
	}

	resetScriptConsoleTokenForTest()
	if got := scriptConsoleToken(); got != token {
		t.Fatalf("token should survive reload: got %q want %q", got, token)
	}
	if out := redactSecretText("token=" + token); strings.Contains(out, token) {
		t.Fatalf("secret not redacted: %q", out)
	}
}
