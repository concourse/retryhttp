package retryhttp

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

//go:generate counterfeiter . BackOff

type BackOff interface {
	NextBackOff() time.Duration
	GetElapsedTime() time.Duration
	Reset()
}

//go:generate counterfeiter . BackOffFactory

type BackOffFactory interface {
	NewBackOff() BackOff
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
	b.RandomizationFactor = 0
	b.Multiplier = 2
	b.MaxInterval = 16 * time.Second
	b.MaxElapsedTime = f.timeout
	b.Clock = backoff.SystemClock

	return b
}
