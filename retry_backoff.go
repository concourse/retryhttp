package retryhttp

import (
	"time"

	"github.com/cenkalti/backoff/v5"
)

//counterfeiter:generate . BackOff

type BackOff interface {
	NextBackOff() time.Duration
	Reset()
}

//counterfeiter:generate . BackOffFactory

type BackOffFactory interface {
	NewBackOff() BackOff
	WithMaxElapsedTime() backoff.RetryOption
}

type exponentialBackOffFactory struct {
	timeout time.Duration
}

func NewExponentialBackOffFactory(timeout time.Duration) BackOffFactory {
	return &exponentialBackOffFactory{
		timeout: timeout,
	}
}

func (f *exponentialBackOffFactory) NewBackOff() BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 1 * time.Second
	b.MaxInterval = 16 * time.Second

	return b
}

func (f *exponentialBackOffFactory) WithMaxElapsedTime() backoff.RetryOption {
	return backoff.WithMaxElapsedTime(f.timeout)
}
