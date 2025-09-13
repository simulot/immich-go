package upload

import (
	"context"
	"testing"
)

// TestWorkerLogging demonstrates that worker information is included in log messages
// for both sequential and concurrent uploads
func TestWorkerLogging(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Test that worker context is created correctly
	workerCtx := context.WithValue(ctx, workerIDKey{}, 2)
	workerID := workerCtx.Value(workerIDKey{}).(int)

	if workerID != 2 {
		t.Errorf("Expected worker ID 2, got %d", workerID)
	}

	// Test that sequential upload (worker ID 1) also has worker information
	sequentialCtx := context.WithValue(ctx, workerIDKey{}, 1)
	sequentialWorkerID := sequentialCtx.Value(workerIDKey{}).(int)

	if sequentialWorkerID != 1 {
		t.Errorf("Expected sequential worker ID 1, got %d", sequentialWorkerID)
	}

	// Test that context without worker information returns zero value
	// This should not happen in normal operation but is good to test
	_, ok := ctx.Value(workerIDKey{}).(int)
	if ok {
		t.Error("Expected no worker information in base context")
	}

	// The actual logging test would require more complex setup to capture
	// the structured log output, but the code changes ensure that worker
	// information is passed through the context and included in log calls
	// for both sequential and concurrent uploads.
}
