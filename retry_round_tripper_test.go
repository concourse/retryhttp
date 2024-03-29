package retryhttp_test

import (
	"context"
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
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("RetryRoundTripper", func() {
	var (
		fakeRoundTripper  *retryhttpfakes.FakeRoundTripper
		fakeBackOff       *retryhttpfakes.FakeBackOff
		testLogger        lager.Logger
		retryRoundTripper *retryhttp.RetryRoundTripper
		response          *http.Response
		roundTripErr      error
		request           *http.Request
	)

	BeforeEach(func() {
		fakeRoundTripper = new(retryhttpfakes.FakeRoundTripper)
		fakeBackOffFactory := new(retryhttpfakes.FakeBackOffFactory)
		fakeBackOff = new(retryhttpfakes.FakeBackOff)
		fakeBackOffFactory.NewBackOffReturns(fakeBackOff)
		testLogger = lager.NewLogger("test")

		retryRoundTripper = &retryhttp.RetryRoundTripper{
			Logger:         testLogger,
			BackOffFactory: fakeBackOffFactory,
			RoundTripper:   fakeRoundTripper,
			Retryer:        &retryhttp.DefaultRetryer{},
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
		response, roundTripErr = retryRoundTripper.RoundTrip(request)
	})

	for _, retryableError := range retryableErrors {
		Context("when the error is "+retryableError.Error(), func() {
			BeforeEach(func() {
				fakeRoundTripper.RoundTripReturns(nil, retryableError)
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
				Expect(roundTripErr).To(Equal(retryableError))
				Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(10))
			})

			Context("when request body was already read from (streaming request)", func() {
				BeforeEach(func() {
					fakeRoundTripper.RoundTripStub = func(request *http.Request) (*http.Response, error) {
						request.Body.Read(make([]byte, 1))
						return &http.Response{StatusCode: http.StatusTeapot}, retryableError
					}
					requestBody := gbytes.NewBuffer()
					requestBody.Write([]byte("hello world"))
					request.Body = requestBody
					buf := make([]byte, 1)
					request.Body.Read(buf)
				})

				It("does not retry", func() {
					Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(1))
					Expect(roundTripErr).To(Equal(retryableError))
				})
			})
		})
	}

	Context("when the error is not retryable", func() {
		var disaster error

		BeforeEach(func() {
			disaster = errors.New("oh no!")
			fakeRoundTripper.RoundTripReturns(nil, disaster)
		})

		It("propagates the error", func() {
			Expect(roundTripErr).To(Equal(disaster))
		})

		It("does not retry", func() {
			Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(1))
		})
	})

	Context("when there is no error", func() {
		BeforeEach(func() {
			fakeRoundTripper.RoundTripReturns(
				&http.Response{StatusCode: http.StatusTeapot},
				nil,
			)
		})

		It("sends the request", func() {
			Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(fakeRoundTripper.RoundTripArgsForCall(0)).To(Equal(
				&http.Request{URL: &url.URL{Path: "some-path"}, Body: nil},
			))
		})

		It("returns the response", func() {
			Expect(response).To(Equal(&http.Response{StatusCode: http.StatusTeapot}))
		})

		It("does not error", func() {
			Expect(roundTripErr).NotTo(HaveOccurred())
		})
	})

	Context("when the context is canceled", func() {
		var innerErr = errors.New("oh no")

		BeforeEach(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			fakeRoundTripper.RoundTripReturns(nil, innerErr)

			request = request.WithContext(ctx)
		})

		It("does not retry and returns the error", func() {
			Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripErr).To(Equal(innerErr))
		})
	})

	Context("when the context times out", func() {
		var innerErr = errors.New("oh no")

		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 0)
			cancel()

			fakeRoundTripper.RoundTripReturns(nil, innerErr)

			request = request.WithContext(ctx)
		})

		It("does not retry and returns the error", func() {
			Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(1))
			Expect(roundTripErr).To(Equal(innerErr))
		})
	})
})
