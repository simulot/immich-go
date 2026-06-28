package nextcloudmemories

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "reconcile"}
	a := app.New(ctx, parent)
	cmd := NewCommand(ctx, a)

	assert.Equal(t, "nextcloud-memories [flags]", cmd.Use)
	assert.Contains(t, cmd.Short, "Reconcile Nextcloud Memories")
	assert.Contains(t, cmd.Long, "post-import convergence workflows")
	assert.NotNil(t, cmd.Flag("cleanup-migration-tags"))
	assert.NotNil(t, cmd.Flag("server"))
	assert.NotNil(t, cmd.Flag("api-key"))
	assert.Empty(t, cmd.Aliases)
}

func TestCommandUsesInjectedRunner(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "reconcile"}
	a := app.New(ctx, parent)
	called := false
	cmd := newCommand(ctx, a, func(ctx context.Context, a *app.Application, client *app.Client, cleanupMigrationTags bool) error {
		called = true
		return nil
	})

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
}
