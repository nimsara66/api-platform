/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package retry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// PropagationCeiling is the minimum timeout for framework waits.
const PropagationCeiling = 60 * time.Second

// BaseInterval is the minimum polling interval.
const BaseInterval = 750 * time.Millisecond

// pacing widens the interval as a wait progresses.
func pacing(elapsed, base time.Duration) time.Duration {
	switch {
	case elapsed < 20*time.Second:
		return base
	case elapsed < 40*time.Second:
		return max(base, 1500*time.Millisecond)
	default:
		return max(base, 3*time.Second)
	}
}

// Until polls for a result that satisfies an acceptance function.

// Attempt produces one result.
type Attempt[T any] func(ctx context.Context) (T, error)

// Accept reports whether a result is the one being waited for.
type Accept[T any] func(T) bool

// Options configure a wait.
type Options struct {
	// Timeout is floored at PropagationCeiling.
	Timeout time.Duration
	// Interval is the base polling cadence.
	Interval time.Duration
	// Retryable classifies an attempt error as transient. Nil means only errors matching
	// IsTransient are retried.
	Retryable func(error) bool
	// Logger receives self-heal and diagnostic lines.
	Logger *slog.Logger

	// subBaseIntervalForTests permits shorter intervals in package tests.
	subBaseIntervalForTests bool

	// subCeilingTimeoutForTests permits shorter timeouts in package tests.
	subCeilingTimeoutForTests bool
}

func (o Options) deadline(now time.Time) time.Time {
	timeout := o.Timeout
	if timeout < PropagationCeiling && !o.subCeilingTimeoutForTests {
		timeout = PropagationCeiling
	}
	return now.Add(timeout)
}

func (o Options) interval() time.Duration {
	if o.Interval <= 0 {
		return BaseInterval
	}
	// Floored, so a step cannot poll faster than the suite's agreed cadence. The only way
	// below it is subBaseIntervalForTests, which is unexported.
	if o.Interval < BaseInterval && !o.subBaseIntervalForTests {
		return BaseInterval
	}
	return o.Interval
}

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// Until polls until accept is satisfied and returns the last result.
func Until[T any](ctx context.Context, opts Options, attempt Attempt[T], accept Accept[T]) (T, error) {
	var last T
	var lastErr error

	start := time.Now()
	deadline := opts.deadline(start)
	attempts := 0

	for {
		attempts++
		result, err := attempt(ctx)
		switch {
		case err == nil:
			last = result
			lastErr = nil
			if accept(result) {
				return result, nil
			}
		case isRetryable(err, opts.Retryable):
			lastErr = err
		default:
			return last, fmt.Errorf("retry: attempt failed with a non-retryable error after %s: %w",
				time.Since(start).Round(time.Millisecond), err)
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return last, fmt.Errorf("retry: cancelled after %d attempt(s): %w", attempts, ctxErr)
		}
		if !time.Now().Before(deadline) {
			break
		}

		wait := pacing(time.Since(start), opts.interval())
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if wait > 0 {
			select {
			case <-ctx.Done():
				return last, fmt.Errorf("retry: cancelled while waiting: %w", ctx.Err())
			case <-time.After(wait):
			}
		}
	}

	if lastErr != nil {
		return last, fmt.Errorf("retry: never succeeded within %s (%d attempt(s)); last error: %w",
			time.Since(start).Round(time.Second), attempts, lastErr)
	}
	return last, nil
}

// Await polls until accept holds and returns an error when it does not.
func Await[T any](
	ctx context.Context, opts Options, attempt Attempt[T], accept Accept[T], what string,
) error {
	last, err := Until(ctx, opts, attempt, accept)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	if accept(last) {
		return nil
	}
	if d, ok := any(last).(interface{ Describe() string }); ok {
		return fmt.Errorf("%s: the condition never held; the last response was %s", what, d.Describe())
	}
	return fmt.Errorf("%s: the condition never held; the last result was %v", what, last)
}

// SettledCount waits for a monotonic counter's final value.

// Counter reports a monotonically non-decreasing value.
type Counter func(ctx context.Context) (int, error)

// Settled is the outcome of waiting for a counter to go quiet.
type Settled struct {
	// Value is the last observed value.
	Value int
	// Quiet reports whether the value stopped changing for the required quiet period. When
	// false, the value was still moving when the deadline passed.
	Quiet bool
	// Samples is how many observations were taken.
	Samples int
}

// SettledCount waits until a counter stops changing for the quiet period.
func SettledCount(ctx context.Context, opts Options, quiet time.Duration, count Counter) (Settled, error) {
	if quiet <= 0 {
		quiet = 2 * time.Second
	}

	start := time.Now()
	deadline := opts.deadline(start)

	var (
		out         Settled
		lastValue   = -1
		lastChanged = start
	)

	for {
		value, err := count(ctx)
		if err != nil {
			if !isRetryable(err, opts.Retryable) {
				return out, fmt.Errorf("retry: counting failed with a non-retryable error: %w", err)
			}
		} else {
			out.Samples++
			if value != lastValue {
				if lastValue > value {
					return out, fmt.Errorf("retry: counter decreased from %d to %d, which is not monotonic",
						lastValue, value)
				}
				lastValue = value
				lastChanged = time.Now()
			}
			out.Value = value

			if time.Since(lastChanged) >= quiet {
				out.Quiet = true
				return out, nil
			}
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return out, fmt.Errorf("retry: counting cancelled after %d sample(s): %w", out.Samples, ctxErr)
		}
		if !time.Now().Before(deadline) {
			return out, nil
		}

		wait := pacing(time.Since(start), opts.interval())
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if wait > 0 {
			select {
			case <-ctx.Done():
				return out, fmt.Errorf("retry: counting cancelled: %w", ctx.Err())
			case <-time.After(wait):
			}
		}
	}
}

// Transient error classification.

// transientError marks an error as retryable.
type transientError struct{ err error }

func (t transientError) Error() string { return t.err.Error() }
func (t transientError) Unwrap() error { return t.err }

// Transient marks an error as retryable.
func Transient(err error) error {
	if err == nil {
		return nil
	}
	return transientError{err: err}
}

// IsTransient reports whether an error was marked retryable.
func IsTransient(err error) bool {
	var t transientError
	return errors.As(err, &t)
}

func isRetryable(err error, custom func(error) bool) bool {
	if err == nil {
		return false
	}
	if custom != nil && custom(err) {
		return true
	}
	return IsTransient(err)
}

// LogAuthRejection logs an authentication rejection and returns its message.
func LogAuthRejection(log *slog.Logger, what, credentialKey string, status int, body string, elapsed time.Duration) string {
	if log == nil {
		log = slog.Default()
	}
	const maxBody = 256
	if len(body) > maxBody {
		body = body[:maxBody] + "…"
	}
	msg := fmt.Sprintf("auth-reject: %s was rejected with %d after %s (credential from context key %q): %s",
		what, status, elapsed.Round(time.Millisecond), credentialKey, body)
	log.Warn("auth-reject", "what", what, "status", status, "credentialKey", credentialKey,
		"elapsed", elapsed.Round(time.Millisecond), "body", body)
	return msg
}

// Never verifies that an invariant holds for a complete window.

// Forbidden reports whether an attempt's result violates the invariant.
type Forbidden[T any] func(T) bool

// Never polls for the configured window and fails when forbidden holds.
func Never[T any](
	ctx context.Context, opts Options, window time.Duration, attempt Attempt[T], forbidden Forbidden[T],
) (T, error) {
	var last T
	if window <= 0 {
		window = PropagationCeiling
	}
	start := time.Now()
	deadline := start.Add(window)
	attempts := 0
	for {
		result, err := attempt(ctx)
		switch {
		case err == nil:
			last = result
			attempts++
			if forbidden(result) {
				return last, fmt.Errorf(
					"retry: the invariant was violated after %s (%d attempt(s)); it was expected to hold for %s",
					time.Since(start).Round(time.Millisecond), attempts, window)
			}
		case isRetryable(err, opts.Retryable):
			// A transient error is not a clean sample, so it must not count toward the window.
		default:
			return last, fmt.Errorf("retry: attempt failed with a non-retryable error after %s: %w",
				time.Since(start).Round(time.Millisecond), err)
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return last, fmt.Errorf("retry: cancelled after %d attempt(s): %w", attempts, ctxErr)
		}
		if !time.Now().Before(deadline) {
			if attempts == 0 {
				// No clean sample, so the window proved nothing.
				return last, fmt.Errorf(
					"retry: the invariant could not be checked — no attempt succeeded within %s", window)
			}
			return last, nil
		}
		wait := pacing(time.Since(start), opts.interval())
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if wait > 0 {
			select {
			case <-ctx.Done():
				return last, fmt.Errorf("retry: cancelled while waiting: %w", ctx.Err())
			case <-time.After(wait):
			}
		}
	}
}
