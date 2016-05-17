package retryhttp_test

import (
	"errors"
	"net"
	"net/http"

	"github.com/concourse/retryhttp"
	"github.com/concourse/retryhttp/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BasicHijackableClient", func() {
	var (
		fakeDoHijackCloserFactory *fakes.FakeDoHijackCloserFactory
		hijackableClient          *retryhttp.BasicHijackableClient
	)

	BeforeEach(func() {
		fakeDoHijackCloserFactory = new(fakes.FakeDoHijackCloserFactory)
		hijackableClient = nil
	})

	Context("when dialing returns an error", func() {
		var (
			disaster      error
			actualNetwork string
			actualAddr    string
		)

		BeforeEach(func() {
			disaster = errors.New("oh no")
			actualNetwork = ""
			actualAddr = ""
			hijackableClient = &retryhttp.BasicHijackableClient{
				Dial: func(network, addr string) (net.Conn, error) {
					actualNetwork = network
					actualAddr = addr
					return nil, disaster
				},
				DoHijackCloserFactory: fakeDoHijackCloserFactory,
			}
		})

		It("returns the error", func() {
			request, err := http.NewRequest("GET", "http://example.com", nil)
			Expect(err).NotTo(HaveOccurred())
			_, _, err = hijackableClient.Do(request)
			Expect(err).To(Equal(disaster))
			Expect(fakeDoHijackCloserFactory.NewDoHijackCloserCallCount()).To(BeZero())
		})

		Context("when the url has no port", func() {
			It("uses port 80", func() {
				request, err := http.NewRequest("GET", "http://example.com", nil)
				Expect(err).NotTo(HaveOccurred())
				_, _, err = hijackableClient.Do(request)
				Expect(actualNetwork).To(Equal("tcp"))
				Expect(actualAddr).To(Equal("example.com:80"))
			})
		})

		Context("when the url has a port", func() {
			It("uses that port", func() {
				request, err := http.NewRequest("GET", "http://example.com:75", nil)
				Expect(err).NotTo(HaveOccurred())
				_, _, err = hijackableClient.Do(request)
				Expect(actualNetwork).To(Equal("tcp"))
				Expect(actualAddr).To(Equal("example.com:75"))
			})
		})
	})

	Context("when dialing succeeds", func() {
		var (
			fakeDoHijackCloser *fakes.FakeDoHijackCloser
			fakeConn           *fakes.FakeConn
			request            *http.Request
		)

		BeforeEach(func() {
			fakeDoHijackCloser = new(fakes.FakeDoHijackCloser)
			fakeDoHijackCloserFactory.NewDoHijackCloserReturns(fakeDoHijackCloser)
			fakeConn = new(fakes.FakeConn)
			hijackableClient = &retryhttp.BasicHijackableClient{
				Dial: func(string, string) (net.Conn, error) {
					return fakeConn, nil
				},
				DoHijackCloserFactory: fakeDoHijackCloserFactory,
			}
			var err error
			request, err = http.NewRequest("GET", "http://example.com", nil)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when making the http request fails", func() {
			var (
				disaster              error
				lessImportantDisaster error
			)

			BeforeEach(func() {
				disaster = errors.New("oh no")
				lessImportantDisaster = errors.New("oh well")
				fakeDoHijackCloser.DoReturns(nil, disaster)
				fakeDoHijackCloser.CloseReturns(lessImportantDisaster)
			})

			It("returns the error", func() {
				_, _, err := hijackableClient.Do(request)
				Expect(err).To(Equal(disaster))
			})

			It("creates the client using the connection", func() {
				hijackableClient.Do(request)
				Expect(fakeDoHijackCloserFactory.NewDoHijackCloserCallCount()).To(Equal(1))
				actualConn, actualReader := fakeDoHijackCloserFactory.NewDoHijackCloserArgsForCall(0)
				Expect(actualConn).To(Equal(fakeConn))
				Expect(actualReader).To(BeNil())
			})

			It("closes the client", func() {
				hijackableClient.Do(request)
				Expect(fakeDoHijackCloser.CloseCallCount()).To(Equal(1))
			})

			It("calls Do with the request", func() {
				hijackableClient.Do(request)
				Expect(fakeDoHijackCloser.DoCallCount()).To(Equal(1))
				Expect(fakeDoHijackCloser.DoArgsForCall(0)).To(Equal(request))
			})
		})

		Context("when making the http request succeeds", func() {
			var response *http.Response

			BeforeEach(func() {
				response = &http.Response{StatusCode: http.StatusOK}
				fakeDoHijackCloser.DoReturns(response, nil)
			})

			It("does not close the client", func() {
				hijackableClient.Do(request)
				Expect(fakeDoHijackCloser.CloseCallCount()).To(BeZero())
			})

			It("returns the response", func() {
				actualResponse, actualHijackCloser, err := hijackableClient.Do(request)
				Expect(err).NotTo(HaveOccurred())
				Expect(actualResponse).To(Equal(response))
				Expect(actualHijackCloser).To(Equal(fakeDoHijackCloser))
			})
		})
	})
})
