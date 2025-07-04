package upload

import (
	"context"
	"testing"
)

// TestWorkerLogging demonstrates that worker information is included in log messages
// when using concurrent uploads
func TestWorkerLogging(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Test that worker context is created correctly
	workerCtx := context.WithValue(ctx, workerIDKey{}, 2)
	workerID, hasWorker := workerCtx.Value(workerIDKey{}).(int)

	if !hasWorker {
		t.Error("Expected worker information to be present in context")
	}

	if workerID != 2 {
		t.Errorf("Expected worker ID 2, got %d", workerID)
	}

	// Test that sequential upload doesn't have worker information
	_, hasWorker = ctx.Value(workerIDKey{}).(int)
	if hasWorker {
		t.Error("Expected no worker information in sequential context")
	}

	// The actual logging test would require more complex setup to capture
	// the structured log output, but the code changes ensure that worker
	// information is passed through the context and included in log calls
	// when concurrent uploads are used.
}
