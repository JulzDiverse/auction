package fashion_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFashion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fashion Suite")
}
