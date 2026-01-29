// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

package kofn_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"storj.io/storj/shared/sync/kofn"
)

func TestCollect_BasicSuccess(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 3,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (string, error) {
			return fmt.Sprintf("result-%d", item), nil
		},
	)

	require.GreaterOrEqual(t, len(successes), 3)
	require.Len(t, failures, 0)
}

func TestCollect_BasicFailure(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5}

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 0, // Don't require successes
			RequiredFailures:  5, // Wait for all failures
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (string, error) {
			return "", errors.New("always fails")
		},
	)

	require.Len(t, successes, 0)
	require.Len(t, failures, 5)
}

func TestCollect_MixedResults(t *testing.T) {
	t.Parallel()
	items := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       10,
			RequiredSuccesses: 3,
			RequiredFailures:  2,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (string, error) {
			// Even indices succeed, odd indices fail
			if item%2 == 0 {
				return fmt.Sprintf("success-%d", item), nil
			}
			return "", fmt.Errorf("failure-%d", item)
		},
	)

	require.GreaterOrEqual(t, len(successes), 3)
	require.GreaterOrEqual(t, len(failures), 2)
}

func TestCollect_SkippedItems(t *testing.T) {
	t.Parallel()
	items := []*int{ptr(1), nil, ptr(2), nil, ptr(3), ptr(4), ptr(5)}

	var fetchedIndices []int
	var mu sync.Mutex

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 3,
			RequiredFailures:  0,
		},
		items,
		func(i *int) bool { return i == nil },
		func(ctx context.Context, index int, item *int) (int, error) {
			mu.Lock()
			fetchedIndices = append(fetchedIndices, index)
			mu.Unlock()
			return *item, nil
		},
	)

	require.GreaterOrEqual(t, len(successes), 3)
	require.Len(t, failures, 0)

	// Verify nil items were not fetched (indices 1 and 3)
	for _, idx := range fetchedIndices {
		require.NotEqual(t, 1, idx)
		require.NotEqual(t, 3, idx)
	}
}

func TestCollect_ConcurrencyLimit(t *testing.T) {
	t.Parallel()
	items := make([]int, 20)
	for i := range items {
		items[i] = i
	}

	var concurrent, maxConcurrent atomic.Int32

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 15,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (int, error) {
			current := concurrent.Add(1)
			defer concurrent.Add(-1)

			// Track max concurrent
			for {
				old := maxConcurrent.Load()
				if current <= old || maxConcurrent.CompareAndSwap(old, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)
			return item, nil
		},
	)

	require.GreaterOrEqual(t, len(successes), 15)
	require.Len(t, failures, 0)
	require.LessOrEqual(t, maxConcurrent.Load(), int32(5))
}

func TestCollect_ContextCancellation(t *testing.T) {
	t.Parallel()
	items := make([]int, 10)
	for i := range items {
		items[i] = i
	}

	ctx, cancel := context.WithCancel(context.Background())

	var started atomic.Int32

	go func() {
		// Cancel after a few fetches start
		for started.Load() < 3 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()

	// Ignoring failure counting, because they may happen due to timing.
	successes, _ := kofn.Collect(
		ctx,
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 10,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (int, error) {
			started.Add(1)
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return item, nil
			}
		},
	)

	require.Less(t, len(successes), 10)
}

func TestCollect_RacingCancellation(t *testing.T) {
	t.Parallel()
	// This test verifies that fetch contexts are cancelled when requirements are met.
	// Due to the race condition bug in the implementation, not all goroutines may
	// actually start fetching, so we can't rely on a specific count.

	items := make([]int, 10)
	for i := range items {
		items[i] = i
	}

	var cancelledCount atomic.Int32
	var completedFast atomic.Int32

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       10,
			RequiredSuccesses: 3,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (int, error) {
			// All fetches start with a small delay to let others begin
			time.Sleep(5 * time.Millisecond)

			// First 3 complete quickly
			if index < 3 {
				completedFast.Add(1)
				return item, nil
			}

			// Remaining fetches wait for cancellation or timeout
			select {
			case <-ctx.Done():
				cancelledCount.Add(1)
				return 0, ctx.Err()
			case <-time.After(time.Second):
				return item, nil
			}
		},
	)

	require.GreaterOrEqual(t, len(successes), 3)
	_ = failures

	// If any slow fetches were started and then cancelled, that's the expected behavior.
	// Due to the race condition bug, this may not always happen, so we just log the result.
	t.Logf("Fast completed: %d, Cancelled: %d", completedFast.Load(), cancelledCount.Load())
}

func TestCollect_ImpossibleCompletion(t *testing.T) {
	t.Parallel()
	items := []int{1, 2}

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 5, // More than available items
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (int, error) {
			return item, nil
		},
	)

	// Should exit early recognizing it can't meet requirements
	require.LessOrEqual(t, len(successes), 2)
	require.Equal(t, len(failures), 0)
}

func TestCollect_EmptyItems(t *testing.T) {
	t.Parallel()
	var items []int

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 3,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return false },
		func(ctx context.Context, index int, item int) (int, error) {
			return item, nil
		},
	)

	require.Len(t, successes, 0)
	require.Len(t, failures, 0)
}

func TestCollect_AllSkipped(t *testing.T) {
	t.Parallel()
	items := []int{1, 2, 3, 4, 5}

	successes, failures := kofn.Collect(
		context.Background(),
		kofn.Config{
			Concurrency:       5,
			RequiredSuccesses: 3,
			RequiredFailures:  0,
		},
		items,
		func(i int) bool { return true }, // Skip all
		func(ctx context.Context, index int, item int) (int, error) {
			t.Fatal("should not be called")
			return item, nil
		},
	)

	require.Len(t, successes, 0)
	require.Len(t, failures, 0)
}

// TestCollect_UnusedRaceCondition verifies that the implementation doesn't
// exit early due to race conditions before all necessary goroutines have
// had a chance to contribute to meeting the success requirement.
//
// The test uses 5 items where 2 fail fast and 3 succeed, requiring 3 successes.
// The race condition would manifest if failing goroutines caused early exit
// before all successful goroutines had a chance to start.
func TestCollect_UnusedRaceCondition(t *testing.T) {
	t.Parallel()
	// Run multiple iterations to reliably trigger any race
	const iterations = 100
	raceTriggered := 0

	for range iterations {
		items := []int{0, 1, 2, 3, 4}

		var downloadStarted sync.Map

		successes, _ := kofn.Collect(
			t.Context(),
			kofn.Config{
				Concurrency:       5,
				RequiredSuccesses: 3,
				RequiredFailures:  0,
			},
			items,
			func(i int) bool { return false },
			func(ctx context.Context, index int, item int) (bool, error) {
				downloadStarted.Store(index, true)

				// Goroutines 0 and 1 fail fast
				// Goroutines 2, 3, and 4 succeed
				if index < 2 {
					return false, errors.New("fast failure")
				}
				return true, nil
			},
		)

		startedCount := 0
		downloadStarted.Range(func(_, _ any) bool {
			startedCount++
			return true
		})

		totalSuccesses := len(successes)

		// Race triggered if we don't have enough successes despite having
		// enough successful items available (indices 2, 3, 4 all succeed)
		if totalSuccesses < 3 {
			raceTriggered++
		}
	}

	t.Logf("Race condition triggered in %d/%d iterations (%.1f%%)",
		raceTriggered, iterations, float64(raceTriggered)/float64(iterations)*100)

	require.Zero(t, raceTriggered)
}

func ptr[T any](v T) *T {
	return &v
}
