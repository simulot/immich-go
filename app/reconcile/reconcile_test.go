package reconcile

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReconcileCommandMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "immich-go"}
	a := app.New(ctx, parent)
	cmd := NewReconcileCommand(ctx, a)

	assert.Equal(t, "reconcile", cmd.Use)
	require.NotNil(t, cmd.Args)
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.Contains(t, cmd.Short, "post-import")
	assert.Len(t, cmd.Commands(), 1)
	assert.Equal(t, "nextcloud-memories [flags]", cmd.Commands()[0].Use)
}

func TestReconcileCommandRequiresSubcommand(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "immich-go"}
	a := app.New(ctx, parent)
	cmd := NewReconcileCommand(ctx, a)

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "must specify a subcommand")
}
