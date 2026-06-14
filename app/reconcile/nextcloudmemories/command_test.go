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
	assert.Empty(t, cmd.Aliases)
}

func TestCommandReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "reconcile"}
	a := app.New(ctx, parent)
	cmd := NewCommand(ctx, a)

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not implemented")
}
