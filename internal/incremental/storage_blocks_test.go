//go:build !windows

package incremental

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAvailableBlockBytesAcrossPlatformTypes(t *testing.T) {
	bytes, err := availableBlockBytes(int64(5), uint64(4096))
	require.NoError(t, err)
	require.EqualValues(t, 20480, bytes)
	bytes, err = availableBlockBytes(uint64(5), int32(4096))
	require.NoError(t, err)
	require.EqualValues(t, 20480, bytes)
	bytes, err = availableBlockBytes(uint64(0), int64(4096))
	require.NoError(t, err)
	require.Zero(t, bytes)
	_, err = availableBlockBytes(int64(-1), int64(4096))
	require.Error(t, err, "negative available blocks must not become huge unsigned free space")
	_, err = availableBlockBytes(uint64(1), int32(0))
	require.Error(t, err)
	_, err = availableBlockBytes(uint64(1), int32(-1))
	require.Error(t, err)
	_, err = availableBlockBytes(uint64(math.MaxUint64), uint64(4096))
	require.Error(t, err, "byte multiplication must not overflow")
}
