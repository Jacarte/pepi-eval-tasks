// Command demo runs a flaky job and prints each wait between attempts.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jobq"
)

func main() {
	cfg := jobq.Config{
		Name:        "export",
		MaxAttempts: 4,
		RetryPolicy: "linear",
		RetryDelay:  250 * time.Millisecond,
		Sleep: func(_ context.Context, d time.Duration) error {
			fmt.Println("waiting", d)
			return nil
		},
	}
	fmt.Printf("policy %s, delay %s\n", cfg.RetryPolicy, cfg.RetryDelay)
	calls := 0
	err := jobq.New(cfg).Run(context.Background(), func(context.Context) error {
		calls++
		fmt.Println("attempt", calls)
		if calls < 4 {
			return errors.New("temporary failure")
		}
		return nil
	})
	fmt.Println("result:", err)
}
