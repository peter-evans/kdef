// Package compose implements Docker compose setup and teardown for integration tests.
package compose

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
)

// Up executes compose up and registers teardown on test cleanup.
func Up(t *testing.T, paths []string, env map[string]string) {
	t.Helper()
	identifier := strings.ToLower(uuid.New().String())
	stackId := tc.StackIdentifier(identifier)
	compose, err := tc.NewDockerComposeWith(tc.WithStackFiles(paths...), stackId)

	require.NoError(t, err, "NewDockerComposeAPI()")

	t.Cleanup(func() {
		require.NoError(t, compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal), "compose.Down()")
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, compose.WithEnv(env).Up(ctx, tc.Wait(true)), "compose.Up()")
}
