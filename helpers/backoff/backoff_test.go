package backoff_test

// sanity check

// import (
// 	"context"
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/simulot/immich-go/helpers/backoff"
// )

// func Test_Wait(t *testing.T) {
// 	b := backoff.DefaultExponentialBackoff()

// 	for {
// 		d := time.Duration(b.NextDelay()).String()
// 		if err := b.Wait(context.TODO()); err != nil {
// 			break
// 		}
// 		fmt.Println(d)
// 	}

// 	if b.GetAttempt() != b.MaxAttempts {
// 		t.Fatalf("attempts: want %d, got %d", b.MaxAttempts, b.GetAttempt())
// 	}
// }
