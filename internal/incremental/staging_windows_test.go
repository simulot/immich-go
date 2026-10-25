package incremental

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsStagingRefusedAndNonstagingRemainsAvailable(t *testing.T) {
	base := t.TempDir()
	state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
	_, err := Open(state, scope(), &fakeClient{}, WithStaging(directory, 100))
	require.ErrorContains(t, err, "staging is unsupported on Windows")
	require.NoDirExists(t, directory)
	i, err := Open(state, scope(), &fakeClient{})
	require.NoError(t, err)
	require.NoError(t, i.Reconcile(context.Background()))
	// A persisted staging config must also fail explicitly when flags are omitted.
	i.state.Staging = &Staging{Directory: directory, StateFile: i.filename, ID: "ABCDEFGHIJKLMNOPQRSTUVWXYZ", MaxBytes: 100}
	require.NoError(t, i.save())
	require.NoError(t, i.Close())
	before, err := os.ReadFile(state)
	require.NoError(t, err)
	_, err = Open(state, scope(), &fakeClient{})
	require.ErrorContains(t, err, "staging is unsupported on Windows")
	after, err := os.ReadFile(state)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
