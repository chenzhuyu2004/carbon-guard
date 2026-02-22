package output

import (
	"encoding/json"
	stderrors "errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
)

func TestHandleExitJSON(t *testing.T) {
	if os.Getenv("CG_HANDLE_EXIT_JSON") == "1" {
		HandleExit(cgerrors.New(stderrors.New("provider failure"), cgerrors.ProviderError), true)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleExitJSON")
	cmd.Env = append(os.Environ(), "CG_HANDLE_EXIT_JSON=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected subprocess to exit non-zero")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != cgerrors.ProviderError {
		t.Fatalf("exit code = %d, expected %d", exitErr.ExitCode(), cgerrors.ProviderError)
	}

	var resp ErrorResponse
	if unmarshalErr := json.Unmarshal(out, &resp); unmarshalErr != nil {
		t.Fatalf("failed to unmarshal JSON stderr: %v, stderr=%q", unmarshalErr, string(out))
	}
	if resp.Code != cgerrors.ProviderError {
		t.Fatalf("json code = %d, expected %d", resp.Code, cgerrors.ProviderError)
	}
	if resp.SchemaVersion != "v1" {
		t.Fatalf("json schema_version = %q, expected %q", resp.SchemaVersion, "v1")
	}
	if resp.Error != "provider failure" {
		t.Fatalf("json error = %q, expected %q", resp.Error, "provider failure")
	}
}

func TestHandleExitText(t *testing.T) {
	if os.Getenv("CG_HANDLE_EXIT_TEXT") == "1" {
		HandleExit(cgerrors.New(stderrors.New("bad input"), cgerrors.InputError), false)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestHandleExitText")
	cmd.Env = append(os.Environ(), "CG_HANDLE_EXIT_TEXT=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected subprocess to exit non-zero")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != cgerrors.InputError {
		t.Fatalf("exit code = %d, expected %d", exitErr.ExitCode(), cgerrors.InputError)
	}
	if !strings.Contains(string(out), "bad input") {
		t.Fatalf("expected stderr to contain plain error text, got %q", string(out))
	}
}
