package fastcdp

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadDevToolsActivePort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DevToolsActivePort")
	if err := os.WriteFile(path, []byte("45678\n/devtools/browser/abc-123\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint, err := readDevToolsActivePort(path)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "ws://127.0.0.1:45678/devtools/browser/abc-123" {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestReadDevToolsActivePortRejectsMalformedPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DevToolsActivePort")
	if err := os.WriteFile(path, []byte("not-a-port\nrelative-path\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readDevToolsActivePort(path); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestBrowserArgsKeepSandboxAndAvoidFidelityFlagsByDefault(t *testing.T) {
	args := browserArgs("/usr/bin/google-chrome", "/tmp/profile", BrowserConfig{})
	wantPrefix := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=/tmp/profile",
		"--no-first-run",
		"--no-default-browser-check",
		"--headless=new",
	}
	if !reflect.DeepEqual(args[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("args prefix = %#v", args)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "--no-sandbox") {
		t.Fatalf("sandbox unexpectedly disabled: %s", joined)
	}
	if args[len(args)-1] != "about:blank" {
		t.Fatalf("last arg = %q", args[len(args)-1])
	}
}

func TestBrowserArgsHeadlessShellDoesNotAddHeadlessFlag(t *testing.T) {
	args := browserArgs("/opt/chrome-headless-shell", "/tmp/profile", BrowserConfig{})
	for _, arg := range args {
		if strings.HasPrefix(arg, "--headless") {
			t.Fatalf("headless shell received redundant flag %q", arg)
		}
	}
}

func TestBoundedBufferKeepsOnlyNewestDiagnostics(t *testing.T) {
	buffer := newBoundedBuffer(8)
	_, _ = buffer.Write([]byte("12345"))
	_, _ = buffer.Write([]byte("67890"))
	if got := buffer.String(); got != "34567890" {
		t.Fatalf("buffer = %q, want %q", got, "34567890")
	}
	_, _ = buffer.Write([]byte("ABCDEFGHIJK"))
	if got := buffer.String(); got != "DEFGHIJK" {
		t.Fatalf("buffer = %q, want newest 8 bytes", got)
	}
}
