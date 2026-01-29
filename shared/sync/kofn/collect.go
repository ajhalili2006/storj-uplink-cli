// Copyright (C) 2026 Storj Labs, Inc.
// See LICENSE for copying information.

// Package kofn provides concurrent collection of K successful results out of N attempts.
// This pattern is commonly used in erasure coding where K pieces are needed to reconstruct data.
package kofn

import (
	"context"
	"sync"

	"storj.io/common/sync2"
)

// Result represents the outcome of a single operation.
type Result[T any] struct {
	Index int
	Value T
	Error error
}

// Config controls the collection behavior.
type Config struct {
	// Concurrency is the maximum number of concurrent operations.
	Concurrency int
	// RequiredSuccesses is the number of successful results required before stopping (K).
	RequiredSuccesses int
	// RequiredFailures is the number of failed results required before stopping.
	// Set to 0 to not wait for any failures.
	RequiredFailures int
}

// Collect runs concurrent operations, collecting K successful results out of N attempts.
//
// - items: the inputs to process (items where skip returns true are skipped)
// - do: called for each non-skipped item; should respect ctx cancellation
//
// Returns all results collected before stopping (successes + failures).
func Collect[Req, Resp any](
	ctx context.Context,
	config Config,
	items []Req,
	skip func(Req) bool,
	do func(ctx context.Context, index int, item Req) (Resp, error),
) (successes []Result[Resp], failures []Result[Resp]) {
	var pending int
	for _, item := range items {
		if !skip(item) {
			pending++
		}
	}

	if pending == 0 {
		return nil, nil
	}

	state := &collectState[Req, Resp]{
		requiredSuccesses: config.RequiredSuccesses,
		requiredFailures:  config.RequiredFailures,
		pending:           pending,
		cancelFuncs:       make(map[int]context.CancelFunc),
		do:                do,
	}
	state.cond.L = &state.mu

	concurrency := min(config.Concurrency, pending)
	limiter := sync2.NewLimiter(concurrency)

	for index, item := range items {
		if skip(item) {
			continue
		}
		index, item := index, item
		limiter.Go(ctx, func() {
			state.run(ctx, index, item)
		})
	}

	limiter.Wait()

	return state.successes, state.failures
}

type collectState[Req, Resp any] struct {
	mu   sync.Mutex
	cond sync.Cond

	requiredSuccesses int
	requiredFailures  int

	successCount int
	failureCount int
	active       int
	pending      int

	successes   []Result[Resp]
	failures    []Result[Resp]
	cancelFuncs map[int]context.CancelFunc

	do func(ctx context.Context, index int, item Req) (Resp, error)
}

func (s *collectState[Req, Resp]) run(ctx context.Context, index int, item Req) {
	s.mu.Lock()
	defer s.cond.Signal()
	defer s.mu.Unlock()

	for {
		// Check if we're done
		if s.successCount >= s.requiredSuccesses && s.failureCount >= s.requiredFailures {
			// Cancel all remaining does
			for _, cancel := range s.cancelFuncs {
				cancel()
			}
			s.cond.Broadcast()
			return
		}

		// Check if completion is impossible
		if s.successCount+s.active+s.pending < s.requiredSuccesses ||
			s.failureCount+s.active+s.pending < s.requiredFailures {
			s.cond.Broadcast()
			return
		}

		// Check if active workers could satisfy remaining requirements.
		// If so, wait rather than starting more work unnecessarily.
		if s.successCount+s.active >= s.requiredSuccesses &&
			s.failureCount+s.active >= s.requiredFailures {
			s.cond.Wait()
			continue
		}

		// Claim work
		s.pending--
		s.active++
		doCtx, cancel := context.WithCancel(ctx)
		s.cancelFuncs[index] = cancel
		s.mu.Unlock()

		// Do the collection (unlocked)
		value, err := s.do(doCtx, index, item)
		cancel()

		// Record result
		s.mu.Lock()
		s.active--
		delete(s.cancelFuncs, index)

		if err != nil {
			s.failures = append(s.failures, Result[Resp]{Index: index, Error: err})
			s.failureCount++
		} else {
			s.successes = append(s.successes, Result[Resp]{Index: index, Value: value})
			s.successCount++
		}
		return
	}
}
