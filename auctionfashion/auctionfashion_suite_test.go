package auctionfashion_test

import (
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var logger lager.Logger

func TestAuctionfashion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auctionfashion Suite")
}
