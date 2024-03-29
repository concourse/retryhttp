package retryhttp

import (
	"net/http"

	"github.com/cenkalti/backoff/v4"

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

	backoff.Retry(func() error {
		response, hijackCloser, err = d.HijackableClient.Do(request)
		if err != nil && d.Retryer.IsRetryable(err) {
			failedAttempts++
			d.Logger.Info("retrying", lager.Data{
				"failed-attempts": failedAttempts,
				"ran-for":         backOff.GetElapsedTime().String(),
				"error":           err.Error(),
			})
			return err
		}

		return nil
	}, backOff)

	return response, hijackCloser, err
}
