package auctionrunner_test

import (
	"errors"

	"code.cloudfoundry.org/auction/auctionfashion"
	"code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/rep"
	"code.cloudfoundry.org/rep/repfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cell", func() {
	var (
		client          *repfakes.FakeSimClient
		emptyCell, cell *auctionrunner.Cell
		defaultAuction  *auctionrunner.AuctionType
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
		defaultAuction = auctionfashion.NewAuctionType(auctionfashion.DefaultAuction)
	})

	Describe("ReserveLRP", func() {

		Context("when there is room for the LRP", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})
				instanceToAdd := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

				initialScore, err := defaultAuction.ScoreForLRP(cell, instance, 0.0)
				Expect(err).NotTo(HaveOccurred())

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := defaultAuction.ScoreForLRP(cell, instance, 0.0)
				Expect(err).NotTo(HaveOccurred())
				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the LRP and keep it in mind when handling future requests", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})
				instanceWithMatchingProcessGuid := BuildLRP("pg-new", "domain", 1, linuxRootFSURL, 10, 10, 10, []string{})
				instanceToAdd := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

				initialScore, err := defaultAuction.ScoreForLRP(cell, instance, 0.0)
				Expect(err).NotTo(HaveOccurred())

				initialScoreForInstanceWithMatchingProcessGuid, err := defaultAuction.ScoreForLRP(cell, instanceWithMatchingProcessGuid, 0.0)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("==", initialScoreForInstanceWithMatchingProcessGuid))

				Expect(cell.ReserveLRP(instanceToAdd)).To(Succeed())

				subsequentScore, err := defaultAuction.ScoreForLRP(cell, instance, 0.0)
				Expect(err).NotTo(HaveOccurred())

				subsequentScoreForInstanceWithMatchingProcessGuid, err := defaultAuction.ScoreForLRP(cell, instanceWithMatchingProcessGuid, 0.0)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
				Expect(initialScoreForInstanceWithMatchingProcessGuid).To(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should have gotten worse")

				Expect(subsequentScore).To(BeNumerically("<", subsequentScoreForInstanceWithMatchingProcessGuid), "the score should be substantially worse for the instance with the matching process guid")
			})
		})

		Context("when there is no room for the LRP", func() {
			It("should error", func() {
				instance := BuildLRP("pg-test", "domain", 0, linuxRootFSURL, 10000, 10, 10, []string{})
				err := cell.ReserveLRP(instance)
				Expect(err).To(MatchError("insufficient resources: memory"))
			})
		})
	})

	Describe("ReserveTask", func() {
		Context("when there is room for the task", func() {
			It("should register its resources usage and keep it in mind when handling future requests", func() {
				task := BuildTask("tg-test", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})
				taskToAdd := BuildTask("tg-new", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})

				initialScore, err := defaultAuction.ScoreForTask(cell, task, 0.0)
				Expect(err).NotTo(HaveOccurred())

				Expect(cell.ReserveTask(taskToAdd)).To(Succeed())

				subsequentScore, err := defaultAuction.ScoreForTask(cell, task, 0.0)
				Expect(err).NotTo(HaveOccurred())
				Expect(initialScore).To(BeNumerically("<", subsequentScore), "the score should have gotten worse")
			})

			It("should register the Task and keep it in mind when handling future requests", func() {
				task := BuildTask("tg-test", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})
				taskToAdd := BuildTask("tg-new", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})

				initialScore, err := defaultAuction.ScoreForTask(cell, task, 0.25)
				Expect(err).NotTo(HaveOccurred())

				initialScoreForTaskToAdd, err := defaultAuction.ScoreForTask(cell, taskToAdd, 0.25)
				Expect(err).NotTo(HaveOccurred())

				Expect(initialScore).To(BeNumerically("==", initialScoreForTaskToAdd))

				Expect(cell.ReserveTask(taskToAdd)).To(Succeed())

				subsequentScore, err := defaultAuction.ScoreForTask(cell, task, 0.25)
				Expect(err).NotTo(HaveOccurred())

				Expect(subsequentScore).To(BeNumerically(">", initialScore+auctionrunner.LocalityOffset), "the score should have gotten worse by at least 1")
			})
		})

		Context("when there is no room for the Task", func() {
			It("should error", func() {
				task := BuildTask("tg-test", "domain", linuxRootFSURL, 10000, 10, 10, []string{}, []string{})
				err := cell.ReserveTask(task)
				Expect(err).To(MatchError("insufficient resources: memory"))
			})
		})
	})

	Describe("Commit", func() {
		Context("with nothing to commit", func() {
			It("does nothing and returns empty", func() {
				failedWork := cell.Commit()
				Expect(failedWork).To(BeZero())
				Expect(client.PerformCallCount()).To(Equal(0))
			})
		})

		Context("with work to commit", func() {
			var lrp rep.LRP

			BeforeEach(func() {
				lrp = *BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 20, 10, 10, []string{})
				Expect(cell.ReserveLRP(&lrp)).To(Succeed())
			})

			It("asks the client to perform", func() {
				cell.Commit()
				Expect(client.PerformCallCount()).To(Equal(1))
				_, work := client.PerformArgsForCall(0)
				Expect(work).To(Equal(rep.Work{LRPs: []rep.LRP{lrp}}))
			})

			Context("when the client returns some failed work", func() {
				It("forwards the failed work", func() {
					failedWork := rep.Work{
						LRPs: []rep.LRP{lrp},
					}
					client.PerformReturns(failedWork, nil)
					Expect(cell.Commit()).To(Equal(failedWork))
				})
			})

			Context("when the client returns an error", func() {
				It("does not return any failed work", func() {
					client.PerformReturns(rep.Work{}, errors.New("boom"))
					Expect(cell.Commit()).To(BeZero())
				})
			})
		})
	})
})
