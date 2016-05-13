package retryhttp_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRetryhttp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retryhttp Suite")
}
