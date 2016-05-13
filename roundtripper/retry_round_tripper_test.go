package roundtripper_test

import (
	"errors"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/concourse/retryhttp/roundtripper"
	"github.com/concourse/retryhttp/roundtripper/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("RetryRoundTripper", func() {
	var (
		fakeRoundTripper  *fakes.FakeRoundTripper
		fakeRetryPolicy   *fakes.FakeRetryPolicy
		fakeSleeper       *fakes.FakeSleeper
		testLogger        lager.Logger
		retryRoundTripper *roundtripper.RetryRoundTripper
		response          *http.Response
		roundTripErr      error
		request           *http.Request
	)

	BeforeEach(func() {
		fakeRoundTripper = new(fakes.FakeRoundTripper)
		fakeRetryPolicy = new(fakes.FakeRetryPolicy)
		fakeSleeper = new(fakes.FakeSleeper)
		testLogger = lager.NewLogger("test")

		retryRoundTripper = &roundtripper.RetryRoundTripper{
			Logger:       testLogger,
			Sleeper:      fakeSleeper,
			RetryPolicy:  fakeRetryPolicy,
			RoundTripper: fakeRoundTripper,
		}
		request = &http.Request{URL: &url.URL{Path: "some-path"}}
	})

	retryableErrors := []error{
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.ETIMEDOUT,
		errors.New("i/o timeout"),
		errors.New("no such host"),
		errors.New("remote error: handshake failure"),
	}

	JustBeforeEach(func() {
		response, roundTripErr = retryRoundTripper.RoundTrip(request)
	})

	for _, retryableError := range retryableErrors {
		Context("when the error is "+retryableError.Error(), func() {
			BeforeEach(func() {
				fakeRoundTripper.RoundTripReturns(
					&http.Response{StatusCode: http.StatusTeapot},
					retryableError,
				)
			})

			Context("as long as the backoff policy returns true", func() {
				BeforeEach(func() {
					durations := make(chan time.Duration, 3)
					durations <- time.Second
					durations <- 2 * time.Second
					durations <- 1000 * time.Second
					close(durations)

					fakeRetryPolicy.DelayForStub = func(failedAttempts uint) (time.Duration, bool) {
						Expect(fakeRoundTripper.RoundTripCallCount()).To(Equal(int(failedAttempts)))

						select {
						case d, ok := <-durations:
							return d, ok
						}
					}
				})

				It("continuously retries with an increasing attempt count", func() {
					Expect(fakeRetryPolicy.DelayForCallCount()).To(Equal(4))
					Expect(fakeSleeper.SleepCallCount()).To(Equal(3))

					Expect(fakeRetryPolicy.DelayForArgsForCall(0)).To(Equal(uint(1)))
					Expect(fakeSleeper.SleepArgsForCall(0)).To(Equal(time.Second))

					Expect(fakeRetryPolicy.DelayForArgsForCall(1)).To(Equal(uint(2)))
					Expect(fakeSleeper.SleepArgsForCall(1)).To(Equal(2 * time.Second))

					Expect(fakeRetryPolicy.DelayForArgsForCall(2)).To(Equal(uint(3)))
					Expect(fakeSleeper.SleepArgsForCall(2)).To(Equal(1000 * time.Second))

					Expect(roundTripErr).To(Equal(retryableError))
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
		})
	}

	Context("when the error is not retryable", func() {
		var disaster error

		BeforeEach(func() {
			fakeRetryPolicy.DelayForReturns(0, true)

			disaster = errors.New("oh no!")
			fakeRoundTripper.RoundTripReturns(
				&http.Response{StatusCode: http.StatusTeapot},
				disaster,
			)
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
				&http.Request{URL: &url.URL{Path: "some-path"}, Body: &roundtripper.RetryReadCloser{}},
			))
		})

		It("returns the response", func() {
			Expect(response).To(Equal(&http.Response{StatusCode: http.StatusTeapot}))
		})

		It("does not error", func() {
			Expect(roundTripErr).NotTo(HaveOccurred())
		})
	})
})
