package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterruptCancelsContextOnFirstSignal(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(nil) })

	signals := make(chan os.Signal, 2)
	t.Cleanup(func() { close(signals) })
	forced := make(chan int, 1)
	startInterruptHandler(cancel, signals, func(code int) { forced <- code })

	signals <- os.Interrupt

	select {
	case <-ctx.Done():
		require.ErrorIs(t, context.Cause(ctx), errInterrupt)
	case <-time.After(time.Second):
		t.Fatal("interrupt did not cancel the context")
	}

	select {
	case code := <-forced:
		t.Fatalf("unexpected forced exit with code %d", code)
	default:
	}
}

func TestInterruptForcesExitOnSecondSignal(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() { cancel(nil) })

	signals := make(chan os.Signal, 2)
	t.Cleanup(func() { close(signals) })
	forced := make(chan int, 1)
	startInterruptHandler(cancel, signals, func(code int) { forced <- code })

	signals <- os.Interrupt
	signals <- os.Interrupt

	select {
	case <-ctx.Done():
		require.True(t, errors.Is(context.Cause(ctx), errInterrupt))
	case <-time.After(time.Second):
		t.Fatal("first interrupt did not cancel the context")
	}

	select {
	case code := <-forced:
		assert.Equal(t, 130, code)
	case <-time.After(time.Second):
		t.Fatal("second interrupt did not force exit")
	}
}

func startInterruptHandler(cancel context.CancelCauseFunc, signals <-chan os.Signal, exitFn func(int)) {
	go func() {
		if _, ok := <-signals; !ok {
			return
		}
		cancel(errInterrupt)
		if _, ok := <-signals; !ok {
			return
		}
		exitFn(130)
	}()
}
