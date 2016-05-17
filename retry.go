package backoff

import (
	"time"

	"golang.org/x/net/context"
)

// An Operation is executing by Retry() or RetryNotify().
// The operation will be retried using a backoff policy if it returns an error.
type Operation func() error

// Notify is a notify-on-error function. It receives an operation error and
// backoff delay if the operation failed (with an error).
//
// NOTE that if the backoff policy stated to stop retrying,
// the notify function isn't called.
type Notify func(error, time.Duration)

// Retry the function f until it does not return error or BackOff stops.
// f is guaranteed to be run at least once.
// It is the caller's responsibility to reset b after Retry returns.
//
// Retry sleeps the goroutine for the duration returned by BackOff after a
// failed operation returns.
func Retry(o Operation, b BackOff) error { return RetryNotify(o, b, nil) }

// RetryNotify calls notify function with the error and wait duration
// for each failed attempt before sleep.
func RetryNotify(operation Operation, b BackOff, notify Notify) error {
	return RetryNotifyWithContext(nil, operation, b, notify)
}

// RetryNotifyWithContext calls notify function with the error and
// wait duration for each failed attempt before sleep. If ctx is
// non-nil, it will return early from a sleep when it's Done channel
// is closed.
func RetryNotifyWithContext(ctx context.Context, operation Operation,
	b BackOff, notify Notify) error {
	// If context is already canceled, return immediately.
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	var err error
	var next time.Duration

	b.Reset()
	for {
		if err = operation(); err == nil {
			return nil
		}

		if next = b.NextBackOff(); next == Stop {
			return err
		}

		if notify != nil {
			notify(err, next)
		}

		if ctx != nil {
			select {
			case <-time.After(next):
			case <-ctx.Done():
				return ctx.Err()
			}
		} else {
			time.Sleep(next)
		}
	}
}
