package auctionfashion_test

import (
	"code.cloudfoundry.org/auction/auctionfashion"
	"code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/rep"
	"code.cloudfoundry.org/rep/repfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BestFit", func() {
	var (
		client          *repfakes.FakeSimClient
		emptyCell, cell *auctionrunner.Cell
		bestFitAuction  *auctionrunner.AuctionType
	)

	BeforeEach(func() {
		client = &repfakes.FakeSimClient{}
		emptyState := BuildCellState("the-zone", 100, 200, 50, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
		emptyCell = auctionrunner.NewCell(logger, "empty-cell", client, emptyState)

		state := BuildCellState("the-zone", 100, 200, 50, false, 10, linuxOnlyRootFSProviders, []rep.LRP{
			*BuildLRP("pg-1", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{}),
			*BuildLRP("pg-1", "domain", 1, linuxRootFSURL, 10, 20, 10, []string{}),
			*BuildLRP("pg-2", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{}),
			*BuildLRP("pg-3", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{}),
			*BuildLRP("pg-4", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{}),
		},
			[]string{},
			[]string{},
			[]string{},
		)
		cell = auctionrunner.NewCell(logger, "the-cell", client, state)
		bestFitAuction = auctionfashion.NewAuctionType(auctionfashion.BestFit)
	})

	Describe("BestFit", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 10, 10, []string{})
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := bestFitAuction.ScoreForLRP(emptyCell, bigInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := bestFitAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)

			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically(">", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := bestFitAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := bestFitAuction.ScoreForLRP(cell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically(">", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{})
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := bestFitAuction.ScoreForLRP(emptyCell, bigInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := bestFitAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically(">", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := bestFitAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := bestFitAuction.ScoreForLRP(cell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically(">", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 20, 10, []string{})

			bigState := BuildCellState("the-zone", 100, 200, 50, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			bigCell := auctionrunner.NewCell(logger, "big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			smallCell := auctionrunner.NewCell(logger, "small-cell", client, smallState)

			bigScore, err := bestFitAuction.ScoreForLRP(bigCell, instance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := bestFitAuction.ScoreForLRP(smallCell, instance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically(">", smallScore), "prefer Cells with less resources")
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10000, 10, 1024, []string{})
					score, err := bestFitAuction.ScoreForLRP(cell, massiveMemoryInstance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: memory"))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10000, 1024, []string{})
					score, err := bestFitAuction.ScoreForLRP(cell, massiveDiskInstance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: disk"))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
					zeroCell := auctionrunner.NewCell(logger, "zero-cell", client, zeroState)
					score, err := bestFitAuction.ScoreForLRP(zeroCell, instance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: containers"))
				})
			})
		})
	})

})
