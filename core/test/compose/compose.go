// Package compose implements Docker compose setup and teardown for integration tests.
package compose

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

// Up executes compose up.
func Up(t *testing.T, paths []string, env map[string]string) *tc.LocalDockerCompose {
	t.Helper()
	identifier := strings.ToLower(uuid.New().String())
	// nolint
	compose := tc.NewLocalDockerCompose(paths, identifier)
	execError := compose.
		WithCommand([]string{"up", "-d"}).
		WithEnv(env).
		Invoke()
	if err := execError.Error; err != nil {
		t.Errorf("compose up failed: %v", err)
		t.FailNow()
	}

	return compose
}

// Down executes compose down.
func Down(t *testing.T, compose *tc.LocalDockerCompose) {
	t.Helper()
	execError := compose.Down()
	if err := execError.Error; err != nil {
		t.Errorf("compose down failed: %v", err)
		t.FailNow()
	}
}
