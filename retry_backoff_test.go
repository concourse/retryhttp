package retryhttp_test

import (
	"net/http"
	"net/url"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/concourse/retryhttp"
	"github.com/concourse/retryhttp/retryhttpfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryBackoffFactory", func() {
	var (
		fakeRoundTripper  *retryhttpfakes.FakeRoundTripper
		testLogger        lager.Logger
		retryRoundTripper *retryhttp.RetryRoundTripper
		roundTripErr      error
		retryableError    error
		request           *http.Request
	)

	BeforeEach(func() {
		retryableError = syscall.ECONNRESET // "connection reset by peer"
		fakeRoundTripper = new(retryhttpfakes.FakeRoundTripper)
		fakeRoundTripper.RoundTripReturns(nil, retryableError)
		testLogger = lager.NewLogger("test")
		request = &http.Request{URL: &url.URL{Path: "some-path"}}
	})

	Context("when using the exponential backoff factory", func() {
		BeforeEach(func() {
			retryRoundTripper = &retryhttp.RetryRoundTripper{
				Logger:         testLogger,
				BackOffFactory: retryhttp.NewExponentialBackOffFactory(3 * time.Second),
				RoundTripper:   fakeRoundTripper,
				Retryer:        &retryhttp.DefaultRetryer{},
			}
		})

		It("it respects the timeout", func() {
			_, roundTripErr = retryRoundTripper.RoundTrip(request)
			Expect(roundTripErr).To(Equal(retryableError))
			Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(2))
			// retries twice, after 1 and 2 seconds -> total time of 3 seconds, add
			// extra second to timeout just to be safe
		})
	})
})
