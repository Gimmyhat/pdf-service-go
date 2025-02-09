package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	errTest   = errors.New("test error")
	logger, _ = zap.NewDevelopment()
)

func TestRetrier_Do(t *testing.T) {
	tests := []struct {
		name           string
		maxAttempts    int
		operation      Operation
		expectedError  error
		expectedCalls  int
		retryableError error
	}{
		{
			name:        "success on first attempt",
			maxAttempts: 3,
			operation: func(ctx context.Context) error {
				return nil
			},
			expectedError: nil,
			expectedCalls: 1,
		},
		{
			name:        "success after retry",
			maxAttempts: 3,
			operation: func() func(ctx context.Context) error {
				attempts := 0
				return func(ctx context.Context) error {
					attempts++
					if attempts < 2 {
						return errTest
					}
					return nil
				}
			}(),
			expectedError: nil,
			expectedCalls: 2,
		},
		{
			name:        "max attempts reached",
			maxAttempts: 3,
			operation: func(ctx context.Context) error {
				return errTest
			},
			expectedError: &RetryError{
				Attempt:       3,
				OriginalError: errTest,
			},
			expectedCalls: 3,
		},
		{
			name:        "context cancelled",
			maxAttempts: 3,
			operation: func(ctx context.Context) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return errTest
			},
			expectedError: context.Canceled,
			expectedCalls: 1,
		},
		{
			name:        "non-retryable error",
			maxAttempts: 3,
			operation: func(ctx context.Context) error {
				return errors.New("non-retryable")
			},
			expectedError: &RetryError{
				Attempt:       1,
				OriginalError: errors.New("non-retryable"),
			},
			retryableError: errTest,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			wrappedOp := func(ctx context.Context) error {
				calls++
				return tt.operation(ctx)
			}

			opts := []Option{WithMaxAttempts(tt.maxAttempts)}
			if tt.retryableError != nil {
				opts = append(opts, WithRetryableErrors([]error{tt.retryableError}))
			}

			r := New("test", logger, opts...)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.expectedError == context.Canceled {
				cancel()
			}

			err := r.Do(ctx, wrappedOp)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if retryErr, ok := tt.expectedError.(*RetryError); ok {
					assert.IsType(t, retryErr, err)
					actualErr := err.(*RetryError)
					assert.Equal(t, retryErr.Attempt, actualErr.Attempt)
					if retryErr.OriginalError != nil {
						assert.Equal(t, retryErr.OriginalError.Error(), actualErr.OriginalError.Error())
					}
				} else {
					assert.ErrorIs(t, err, tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedCalls, calls, "unexpected number of calls")
		})
	}
}

func TestRetrier_Delay(t *testing.T) {
	tests := []struct {
		name          string
		initialDelay  time.Duration
		maxDelay      time.Duration
		backoffFactor float64
		attempt       int
		expected      time.Duration
	}{
		{
			name:          "first attempt",
			initialDelay:  100 * time.Millisecond,
			maxDelay:      1 * time.Second,
			backoffFactor: 2.0,
			attempt:       1,
			expected:      100 * time.Millisecond,
		},
		{
			name:          "second attempt",
			initialDelay:  100 * time.Millisecond,
			maxDelay:      1 * time.Second,
			backoffFactor: 2.0,
			attempt:       2,
			expected:      200 * time.Millisecond,
		},
		{
			name:          "max delay reached",
			initialDelay:  100 * time.Millisecond,
			maxDelay:      300 * time.Millisecond,
			backoffFactor: 2.0,
			attempt:       3,
			expected:      300 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New("test", logger,
				WithInitialDelay(tt.initialDelay),
				WithMaxDelay(tt.maxDelay),
				WithBackoffFactor(tt.backoffFactor),
			)

			delay := r.calculateDelay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		retryableErrors []error
		expected        bool
	}{
		{
			name:            "nil error",
			err:             nil,
			retryableErrors: []error{errTest},
			expected:        false,
		},
		{
			name:            "empty retryable errors",
			err:             errTest,
			retryableErrors: nil,
			expected:        true,
		},
		{
			name:            "retryable error",
			err:             errTest,
			retryableErrors: []error{errTest},
			expected:        true,
		},
		{
			name:            "non-retryable error",
			err:             errors.New("other"),
			retryableErrors: []error{errTest},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err, tt.retryableErrors)
			assert.Equal(t, tt.expected, result)
		})
	}
}
