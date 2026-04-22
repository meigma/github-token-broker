//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var (
	bootstrapOnce sync.Once
	bootstrapDir  string
	bootstrapPath string
	bootstrapErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if bootstrapDir != "" {
		_ = os.RemoveAll(bootstrapDir)
	}
	os.Exit(code)
}

func buildHostBootstrap(t *testing.T) string {
	t.Helper()

	bootstrapOnce.Do(func() {
		bootstrapDir, bootstrapErr = os.MkdirTemp("", "github-token-broker-bootstrap-*")
		if bootstrapErr != nil {
			return
		}

		bootstrapPath = filepath.Join(bootstrapDir, "bootstrap")
		cmd := exec.Command("go", "build", "-tags", "lambda.norpc", "-o", bootstrapPath, "./cmd/github-token-broker")
		cmd.Dir = findRepoRoot(t)

		var output []byte
		output, bootstrapErr = cmd.CombinedOutput()
		if bootstrapErr != nil {
			bootstrapErr = &buildError{err: bootstrapErr, output: string(output)}
			return
		}

		var info os.FileInfo
		info, bootstrapErr = os.Stat(bootstrapPath)
		if bootstrapErr != nil {
			return
		}
		if info.IsDir() {
			bootstrapErr = &buildError{err: os.ErrInvalid, output: "bootstrap path is a directory"}
		}
	})

	if bootstrapErr != nil {
		t.Fatalf("build host bootstrap: %v", bootstrapErr)
	}

	return bootstrapPath
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod walking up from %s", cwd)
		}
		dir = parent
	}
}

type buildError struct {
	err    error
	output string
}

func (e *buildError) Error() string {
	if e.output == "" {
		return e.err.Error()
	}
	return e.err.Error() + ": " + e.output
}
