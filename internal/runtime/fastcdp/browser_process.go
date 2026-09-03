package fastcdp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBrowserStartupTimeout = 10 * time.Second
	browserDiagnosticLimit       = 64 << 10
)

// BrowserConfig owns only Chromium process startup. Target/context/page pooling
// remains a separate CDP concern.
type BrowserConfig struct {
	Executable   string
	UserDataDir  string
	ExtraArgs    []string
	Env          []string
	StartupTimeout time.Duration
	NoSandbox    bool
}

type BrowserProcess struct {
	Executable  string
	UserDataDir string
	Endpoint    string

	cmd      *exec.Cmd
	ownedDir bool
	stderr   *boundedBuffer
	done     chan struct{}

	mu      sync.Mutex
	waitErr error
	closeOnce sync.Once
}

// FindBrowser returns the first suitable Chromium-family executable. Explicit
// UIUX_CHROME_BIN wins; otherwise chrome-headless-shell is preferred for a
// low-overhead headless runtime, followed by common Chrome/Chromium names.
func FindBrowser() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("UIUX_CHROME_BIN")); explicit != "" {
		if info, err := os.Stat(explicit); err == nil && !info.IsDir() {
			return explicit, nil
		}
		if resolved, err := exec.LookPath(explicit); err == nil {
			return resolved, nil
		}
		return "", fmt.Errorf("fastcdp: UIUX_CHROME_BIN %q is not executable", explicit)
	}
	for _, name := range []string{"chrome-headless-shell", "google-chrome", "google-chrome-stable", "chromium", "chromium-browser"} {
		if resolved, err := exec.LookPath(name); err == nil {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("fastcdp: no Chromium executable found; set UIUX_CHROME_BIN")
}

// LaunchBrowser starts one resident Chromium process and waits only for the
// browser-level CDP endpoint. It does not create contexts/pages or navigate.
func LaunchBrowser(ctx context.Context, config BrowserConfig) (*BrowserProcess, error) {
	executable := strings.TrimSpace(config.Executable)
	if executable == "" {
		var err error
		executable, err = FindBrowser()
		if err != nil {
			return nil, err
		}
	}

	userDataDir := strings.TrimSpace(config.UserDataDir)
	ownedDir := false
	if userDataDir == "" {
		var err error
		userDataDir, err = os.MkdirTemp("", "uiuxmaster-chrome-")
		if err != nil {
			return nil, fmt.Errorf("fastcdp: create browser profile: %w", err)
		}
		ownedDir = true
	} else if err := os.MkdirAll(userDataDir, 0o700); err != nil {
		return nil, fmt.Errorf("fastcdp: create browser profile %q: %w", userDataDir, err)
	}

	args := browserArgs(executable, userDataDir, config)
	cmd := exec.Command(executable, args...)
	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), config.Env...)
	}
	stderr := newBoundedBuffer(browserDiagnosticLimit)
	cmd.Stdout = io.Discard
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		if ownedDir {
			_ = os.RemoveAll(userDataDir)
		}
		return nil, fmt.Errorf("fastcdp: start browser %q: %w", executable, err)
	}

	process := &BrowserProcess{
		Executable: executable,
		UserDataDir: userDataDir,
		cmd: cmd,
		ownedDir: ownedDir,
		stderr: stderr,
		done: make(chan struct{}),
	}
	go func() {
		err := cmd.Wait()
		process.mu.Lock()
		process.waitErr = err
		process.mu.Unlock()
		close(process.done)
	}()

	startupTimeout := config.StartupTimeout
	if startupTimeout <= 0 {
		startupTimeout = defaultBrowserStartupTimeout
	}
	startupCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	endpoint, err := waitForDevToolsEndpoint(startupCtx, process)
	if err != nil {
		_ = process.Close()
		return nil, err
	}
	process.Endpoint = endpoint
	return process, nil
}

func browserArgs(executable, userDataDir string, config BrowserConfig) []string {
	args := []string{
		"--remote-debugging-port=0",
		"--user-data-dir=" + userDataDir,
		"--no-first-run",
		"--no-default-browser-check",
	}
	if !strings.Contains(strings.ToLower(filepath.Base(executable)), "headless-shell") {
		args = append(args, "--headless=new")
	}
	if config.NoSandbox {
		args = append(args, "--no-sandbox")
	}
	args = append(args, config.ExtraArgs...)
	args = append(args, "about:blank")
	return args
}

func waitForDevToolsEndpoint(ctx context.Context, process *BrowserProcess) (string, error) {
	path := filepath.Join(process.UserDataDir, "DevToolsActivePort")
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if endpoint, err := readDevToolsActivePort(path); err == nil {
			return endpoint, nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("fastcdp: browser CDP startup: %w; diagnostics: %s", ctx.Err(), process.Diagnostics())
		case <-process.done:
			return "", fmt.Errorf("fastcdp: browser exited before CDP endpoint: %v; diagnostics: %s", process.WaitErr(), process.Diagnostics())
		case <-ticker.C:
		}
	}
}

func readDevToolsActivePort(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("fastcdp: invalid DevToolsActivePort payload")
	}
	port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil || port <= 0 || port > 65535 {
		return "", fmt.Errorf("fastcdp: invalid DevToolsActivePort port %q", lines[0])
	}
	wsPath := strings.TrimSpace(lines[1])
	if !strings.HasPrefix(wsPath, "/") {
		return "", fmt.Errorf("fastcdp: invalid DevToolsActivePort websocket path %q", wsPath)
	}
	return fmt.Sprintf("ws://127.0.0.1:%d%s", port, wsPath), nil
}

func (p *BrowserProcess) Done() <-chan struct{} { return p.done }

func (p *BrowserProcess) WaitErr() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
}

func (p *BrowserProcess) Diagnostics() string {
	if p.stderr == nil {
		return ""
	}
	return p.stderr.String()
}

func (p *BrowserProcess) Close() error {
	var closeErr error
	p.closeOnce.Do(func() {
		if p.cmd != nil && p.cmd.Process != nil {
			select {
			case <-p.done:
			default:
				if err := p.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
					closeErr = err
				}
			}
			<-p.done
		}
		if p.ownedDir {
			if err := os.RemoveAll(p.UserDataDir); err != nil && closeErr == nil {
				closeErr = err
			}
		}
	})
	return closeErr
}

type boundedBuffer struct {
	mu    sync.Mutex
	limit int
	buf   bytes.Buffer
}

func newBoundedBuffer(limit int) *boundedBuffer { return &boundedBuffer{limit: limit} }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	originalLen := len(p)
	if b.limit <= 0 {
		return originalLen, nil
	}
	if len(p) >= b.limit {
		b.buf.Reset()
		_, _ = b.buf.Write(p[len(p)-b.limit:])
		return originalLen, nil
	}
	overflow := b.buf.Len() + len(p) - b.limit
	if overflow > 0 {
		current := b.buf.Bytes()
		kept := append([]byte(nil), current[overflow:]...)
		b.buf.Reset()
		_, _ = b.buf.Write(kept)
	}
	_, _ = b.buf.Write(p)
	return originalLen, nil
}

func (b *boundedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
