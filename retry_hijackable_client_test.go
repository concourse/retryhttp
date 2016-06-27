package retryhttp_test

import (
	"errors"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/concourse/retryhttp"
	"github.com/concourse/retryhttp/retryhttpfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("RetryHijackableClient", func() {
	var (
		fakeHijackableClient  *retryhttpfakes.FakeHijackableClient
		fakeRetryPolicy       *retryhttpfakes.FakeRetryPolicy
		fakeSleeper           *retryhttpfakes.FakeSleeper
		testLogger            lager.Logger
		retryHijackableClient *retryhttp.RetryHijackableClient
		response              *http.Response
		hijackCloser          retryhttp.HijackCloser
		clientError           error
		request               *http.Request
	)

	BeforeEach(func() {
		fakeHijackableClient = new(retryhttpfakes.FakeHijackableClient)
		fakeRetryPolicy = new(retryhttpfakes.FakeRetryPolicy)
		fakeSleeper = new(retryhttpfakes.FakeSleeper)
		testLogger = lager.NewLogger("test")

		retryHijackableClient = &retryhttp.RetryHijackableClient{
			Logger:           testLogger,
			Sleeper:          fakeSleeper,
			RetryPolicy:      fakeRetryPolicy,
			HijackableClient: fakeHijackableClient,
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
		response, hijackCloser, clientError = retryHijackableClient.Do(request)
	})

	for _, retryableError := range retryableErrors {
		Context("when the error is "+retryableError.Error(), func() {
			BeforeEach(func() {
				fakeHijackableClient.DoReturns(nil, nil, retryableError)
			})

			Context("as long as the backoff policy returns true", func() {
				BeforeEach(func() {
					durations := make(chan time.Duration, 3)
					durations <- time.Second
					durations <- 2 * time.Second
					durations <- 1000 * time.Second
					close(durations)

					fakeRetryPolicy.DelayForStub = func(failedAttempts uint) (time.Duration, bool) {
						Expect(fakeHijackableClient.DoCallCount()).To(Equal(int(failedAttempts)))

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

					Expect(clientError).To(Equal(retryableError))
				})
			})
		})
	}

	Context("when the error is not retryable", func() {
		var disaster error

		BeforeEach(func() {
			fakeRetryPolicy.DelayForReturns(0, true)

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
