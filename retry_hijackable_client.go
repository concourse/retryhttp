package retryhttp

import (
	"context"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v5"

	"code.cloudfoundry.org/lager/v3"
)

type RetryHijackableClient struct {
	Logger           lager.Logger
	BackOffFactory   BackOffFactory
	HijackableClient HijackableClient
	Retryer          Retryer
}

func (d *RetryHijackableClient) Do(request *http.Request) (*http.Response, HijackCloser, error) {
	var response *http.Response
	var hijackCloser HijackCloser
	var err error
	var failedAttempts uint

	backOff := d.BackOffFactory.NewBackOff()
	start := time.Now()

	backoff.Retry(context.TODO(), func() (bool, error) {
		response, hijackCloser, err = d.HijackableClient.Do(request)
		retryer := d.Retryer
		if retryer == nil {
			retryer = &DefaultRetryer{}
		}
		if err != nil && retryer.IsRetryable(err) {
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

	return response, hijackCloser, err
}
