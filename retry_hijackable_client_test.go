package retryhttp_test

import (
	"errors"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager/v3"
	"github.com/cenkalti/backoff/v4"
	"github.com/concourse/retryhttp"
	"github.com/concourse/retryhttp/retryhttpfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RetryHijackableClient", func() {
	var (
		fakeHijackableClient  *retryhttpfakes.FakeHijackableClient
		fakeBackOff           *retryhttpfakes.FakeBackOff
		testLogger            lager.Logger
		retryHijackableClient *retryhttp.RetryHijackableClient
		response              *http.Response
		hijackCloser          retryhttp.HijackCloser
		clientError           error
		request               *http.Request
	)

	BeforeEach(func() {
		fakeHijackableClient = new(retryhttpfakes.FakeHijackableClient)
		fakeBackOffFactory := new(retryhttpfakes.FakeBackOffFactory)
		fakeBackOff = new(retryhttpfakes.FakeBackOff)
		fakeBackOffFactory.NewBackOffReturns(fakeBackOff)
		testLogger = lager.NewLogger("test")

		retryHijackableClient = &retryhttp.RetryHijackableClient{
			Logger:           testLogger,
			BackOffFactory:   fakeBackOffFactory,
			HijackableClient: fakeHijackableClient,
			Retryer:          &retryhttp.DefaultRetryer{},
		}
		request = &http.Request{URL: &url.URL{Path: "some-path"}}
	})

	retryableErrors := []error{
		syscall.ECONNREFUSED, // "connection refused"
		syscall.ECONNRESET,   // "connection reset by peer"
		syscall.ETIMEDOUT,    // "operation timed out"
		errors.New("i/o timeout"),
		errors.New("no such host"),
		errors.New("handshake failure"),
		errors.New("handshake timeout"),
		errors.New("timeout awaiting response headers"),
	}

	JustBeforeEach(func() {
		response, hijackCloser, clientError = retryHijackableClient.Do(request)
	})

	for _, retryableError := range retryableErrors {
		Context("when the error is "+retryableError.Error(), func() {
			BeforeEach(func() {
				fakeHijackableClient.DoReturns(nil, nil, retryableError)
			})

			Context("as long as the backoff policy does not stop", func() {
				BeforeEach(func() {
					backOffAttempts := 0
					fakeBackOff.NextBackOffStub = func() time.Duration {
						backOffAttempts++
						if backOffAttempts >= 10 {
							return backoff.Stop
						}

						return 0 * time.Second
					}
				})

				It("continuously retries with an increasing attempt count until backoff policy ends", func() {
					Expect(clientError).To(Equal(retryableError))
					Expect(fakeHijackableClient.DoCallCount()).To(Equal(10))
				})
			})
		})
	}

	Context("when the error is not retryable", func() {
		var disaster error

		BeforeEach(func() {
			disaster = errors.New("oh no!")
			fakeHijackableClient.DoReturns(nil, nil, disaster)
		})

		It("propagates the error", func() {
			Expect(clientError).To(Equal(disaster))
		})

		It("does not retry", func() {
			Expect(fakeHijackableClient.DoCallCount()).To(Equal(1))
		})
	})

	Context("when there is no error", func() {
		var fakeHijackCloser *retryhttpfakes.FakeHijackCloser

		BeforeEach(func() {
			fakeHijackCloser = new(retryhttpfakes.FakeHijackCloser)
			fakeHijackableClient.DoReturns(
				&http.Response{StatusCode: http.StatusTeapot},
				fakeHijackCloser,
				nil,
			)
		})

		It("sends the request", func() {
			Expect(fakeHijackableClient.DoCallCount()).To(Equal(1))
			Expect(fakeHijackableClient.DoArgsForCall(0)).To(Equal(
				&http.Request{URL: &url.URL{Path: "some-path"}},
			))
		})

		It("returns the response", func() {
			Expect(response).To(Equal(&http.Response{StatusCode: http.StatusTeapot}))
		})

		It("returns the hijackCloser", func() {
			Expect(hijackCloser).To(Equal(fakeHijackCloser))
		})

		It("does not error", func() {
			Expect(clientError).NotTo(HaveOccurred())
		})
	})
})
