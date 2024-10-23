package backoff

import (
	"context"
	"fmt"
	"math"
	"time"
)

type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Factor       float64
	MaxAttempts  int
	attempt      int
}

// NewExponentialBackoff creates a new ExponentialBackoff with default values
func DefaultExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Factor:       3.5,
		MaxAttempts:  5,
	}
}

func (b *ExponentialBackoff) Wait(ctx context.Context) error {
	if b.attempt >= b.MaxAttempts {
		return fmt.Errorf("exceeded max attempts: %d", b.MaxAttempts)
	}

	// Create timer and wait
	timer := time.NewTimer(b.NextDelay())
	defer timer.Stop()
	b.attempt += 1

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (b *ExponentialBackoff) GetAttempt() int {
	return b.attempt
}

func (b *ExponentialBackoff) NextDelay() time.Duration {
	delay := float64(b.InitialDelay) * math.Pow(b.Factor, float64(b.attempt))
	if delay > float64(b.MaxDelay) {
		return b.MaxDelay
	}

	return time.Duration(delay)
}
