package compose

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	tc "github.com/testcontainers/testcontainers-go"
)

func Up(t *testing.T, paths []string, env map[string]string) *tc.LocalDockerCompose {
	t.Helper()
	identifier := strings.ToLower(uuid.New().String())
	compose := tc.NewLocalDockerCompose(paths, identifier)
	execError := compose.
		WithCommand([]string{"up", "-d"}).
		WithEnv(env).
		Invoke()
	err := execError.Error
	if err != nil {
		t.Errorf("compose up failed: %v", err)
		t.FailNow()
	}

	return compose
}

func Down(t *testing.T, compose *tc.LocalDockerCompose) {
	t.Helper()
	execError := compose.Down()
	err := execError.Error
	if err != nil {
		t.Errorf("compose down failed: %v", err)
		t.FailNow()
	}

	// TODO: Is it possible to delete the dangling volume?
	// docker volume rm $(docker volume ls -qf dangling=true)
}
