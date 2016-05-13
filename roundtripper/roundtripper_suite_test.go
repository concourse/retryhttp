package roundtripper_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRoundtripper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Roundtripper Suite")
}
