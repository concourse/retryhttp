package retryhttp

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v5"

	"code.cloudfoundry.org/lager/v3"
)

//counterfeiter:generate . Sleeper

type Sleeper interface {
	Sleep(time.Duration)
}

//counterfeiter:generate . RoundTripper

type RoundTripper interface {
	RoundTrip(request *http.Request) (*http.Response, error)
}

type RetryRoundTripper struct {
	Logger         lager.Logger
	BackOffFactory BackOffFactory
	RoundTripper   RoundTripper
	Retryer        Retryer
}

type RetryReadCloser struct {
	io.ReadCloser
	IsRead bool
}

func (rrc *RetryReadCloser) Read(p []byte) (n int, err error) {
	rrc.IsRead = true
	return rrc.ReadCloser.Read(p)
}

func (d *RetryRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	retryReadCloser := &RetryReadCloser{request.Body, false}

	if request.Body != nil {
		request.Body = retryReadCloser
	}

	var response *http.Response
	var err error
	var failedAttempts uint

	backOff := d.BackOffFactory.NewBackOff()
	start := time.Now()

	backoff.Retry(context.TODO(), func() (bool, error) {
		response, err = d.RoundTripper.RoundTrip(request)
		retryer := d.Retryer
		if retryer == nil {
			retryer = &DefaultRetryer{}
		}
		if err != nil && !retryReadCloser.IsRead && retryer.IsRetryable(err) {
			if request.Context().Err() != nil {
				return false, backoff.Permanent(err)
			}

			failedAttempts++
			d.Logger.Info("retrying", lager.Data{
				"failed-attempts": failedAttempts,
				"ran-for":         time.Since(start).String(),
				"error":           err.Error(),
			})
			return false, err
		}

		return true, nil
	}, backoff.WithBackOff(backOff), d.BackOffFactory.WithMaxElapsedTime())

	return response, err
}
