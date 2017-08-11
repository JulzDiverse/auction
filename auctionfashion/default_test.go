package auctionfashion_test

import (
	"code.cloudfoundry.org/auction/auctionfashion"
	"code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/rep"
	"code.cloudfoundry.org/rep/repfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DefaultAuction", func() {
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

	Describe("ScoreForLRP", func() {
		It("factors in memory usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 10, 10, []string{})
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := defaultAuction.ScoreForLRP(emptyCell, bigInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := defaultAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := defaultAuction.ScoreForLRP(cell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigInstance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 10, 20, 10, []string{})
			smallInstance := BuildLRP("pg-small", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

			By("factoring in the amount of memory taken up by the instance")
			bigScore, err := defaultAuction.ScoreForLRP(emptyCell, bigInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := defaultAuction.ScoreForLRP(emptyCell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := defaultAuction.ScoreForLRP(cell, smallInstance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			instance := BuildLRP("pg-big", "domain", 0, linuxRootFSURL, 20, 20, 10, []string{})

			bigState := BuildCellState("the-zone", 100, 200, 50, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			bigCell := auctionrunner.NewCell(logger, "big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			smallCell := auctionrunner.NewCell(logger, "small-cell", client, smallState)

			bigScore, err := defaultAuction.ScoreForLRP(bigCell, instance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForLRP(smallCell, instance, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		Context("Starting Containers", func() {
			var instance *rep.LRP
			var busyState, boredState rep.CellState
			var busyCell, boredCell *auctionrunner.Cell

			BeforeEach(func() {
				instance = BuildLRP("pg-busy", "domain", 0, linuxRootFSURL, 20, 20, 10, []string{})

				busyState = BuildCellState(
					"the-zone",
					100,
					200,
					50,
					false,
					10,
					linuxOnlyRootFSProviders,
					[]rep.LRP{{ActualLRPKey: models.ActualLRPKey{ProcessGuid: "not-HA"}}},
					[]string{},
					[]string{},
					[]string{},
				)
				busyCell = auctionrunner.NewCell(logger, "busy-cell", client, busyState)

				boredState = BuildCellState(
					"the-zone",
					100,
					200,
					50,
					false,
					0,
					linuxOnlyRootFSProviders,
					[]rep.LRP{{ActualLRPKey: models.ActualLRPKey{ProcessGuid: "HA"}}},
					[]string{},
					[]string{},
					[]string{},
				)
				boredCell = auctionrunner.NewCell(logger, "bored-cell", client, boredState)
			})

			It("factors in starting containers when a weight is provided", func() {
				startingContainerWeight := 0.25

				busyScore, err := defaultAuction.ScoreForLRP(busyCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				boredScore, err := defaultAuction.ScoreForLRP(boredCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())

				Expect(busyScore).To(BeNumerically(">", boredScore), "prefer Cells that have less starting containers")

				smallerWeightState := BuildCellState(
					"the-zone",
					100,
					200,
					50,
					false,
					10,
					linuxOnlyRootFSProviders,
					nil,
					[]string{},
					[]string{},
					[]string{},
				)
				smallerWeightCell := auctionrunner.NewCell(logger, "busy-cell", client, smallerWeightState)
				smallerWeightScore, err := defaultAuction.ScoreForLRP(smallerWeightCell, instance, startingContainerWeight-0.1)
				Expect(err).NotTo(HaveOccurred())

				Expect(busyScore).To(BeNumerically(">", smallerWeightScore), "the number of starting containers is weighted")
			})

			It("privileges spreading LRPs across cells over starting containers", func() {
				instance = BuildLRP("HA", "domain", 1, linuxRootFSURL, 20, 20, 10, []string{})
				startingContainerWeight := 0.25

				busyScore, err := defaultAuction.ScoreForLRP(busyCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				boredScore, err := defaultAuction.ScoreForLRP(boredCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())

				Expect(busyScore).To(BeNumerically("<", boredScore), "prefer Cells that do not have an instance of self already running")
			})

			It("ignores starting containers when a weight is not provided", func() {
				startingContainerWeight := 0.0

				busyScore, err := defaultAuction.ScoreForLRP(busyCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				boredScore, err := defaultAuction.ScoreForLRP(boredCell, instance, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())

				Expect(busyScore).To(BeNumerically("==", boredScore), "ignore how many starting Containers a cell has")
			})
		})

		It("factors in process-guids that are already present", func() {
			instanceWithTwoMatches := BuildLRP("pg-1", "domain", 2, linuxRootFSURL, 10, 10, 10, []string{})
			instanceWithOneMatch := BuildLRP("pg-2", "domain", 1, linuxRootFSURL, 10, 10, 10, []string{})
			instanceWithNoMatches := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})

			twoMatchesScore, err := defaultAuction.ScoreForLRP(cell, instanceWithTwoMatches, 0.0)
			Expect(err).NotTo(HaveOccurred())
			oneMatchesScore, err := defaultAuction.ScoreForLRP(cell, instanceWithOneMatch, 0.0)
			Expect(err).NotTo(HaveOccurred())
			noMatchesScore, err := defaultAuction.ScoreForLRP(cell, instanceWithNoMatches, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(noMatchesScore).To(BeNumerically("<", oneMatchesScore))
			Expect(oneMatchesScore).To(BeNumerically("<", twoMatchesScore))
		})

		Context("when the LRP does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10000, 10, 1024, []string{})
					score, err := defaultAuction.ScoreForLRP(cell, massiveMemoryInstance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: memory"))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskInstance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10000, 1024, []string{})
					score, err := defaultAuction.ScoreForLRP(cell, massiveDiskInstance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: disk"))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					instance := BuildLRP("pg-new", "domain", 0, linuxRootFSURL, 10, 10, 10, []string{})
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
					zeroCell := auctionrunner.NewCell(logger, "zero-cell", client, zeroState)
					score, err := defaultAuction.ScoreForLRP(zeroCell, instance, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: containers"))
				})
			})
		})
	})

	Describe("ScoreForTask", func() {
		var defaultAuction *auctionrunner.AuctionType
		BeforeEach(func() {
			defaultAuction = auctionrunner.NewAuctionType(auctionrunner.DefaultAuction)
		})

		It("factors in number of tasks currently running", func() {
			bigTask := BuildTask("tg-big", "domain", linuxRootFSURL, 20, 10, 10, []string{}, []string{})
			smallTask := BuildTask("tg-small", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := defaultAuction.ScoreForTask(emptyCell, bigTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := defaultAuction.ScoreForTask(cell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in memory usage", func() {
			bigTask := BuildTask("tg-big", "domain", linuxRootFSURL, 20, 10, 10, []string{}, []string{})
			smallTask := BuildTask("tg-small", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := defaultAuction.ScoreForTask(emptyCell, bigTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := defaultAuction.ScoreForTask(cell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in disk usage", func() {
			bigTask := BuildTask("tg-big", "domain", linuxRootFSURL, 10, 20, 10, []string{}, []string{})
			smallTask := BuildTask("tg-small", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})

			By("factoring in the amount of memory taken up by the task")
			bigScore, err := defaultAuction.ScoreForTask(emptyCell, bigTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())

			Expect(smallScore).To(BeNumerically("<", bigScore))

			By("factoring in the relative emptiness of Cells")
			emptyScore, err := defaultAuction.ScoreForTask(emptyCell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			score, err := defaultAuction.ScoreForTask(cell, smallTask, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(emptyScore).To(BeNumerically("<", score))
		})

		It("factors in container usage", func() {
			task := BuildTask("tg-big", "domain", linuxRootFSURL, 20, 20, 10, []string{}, []string{})

			bigState := BuildCellState("the-zone", 100, 200, 50, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			bigCell := auctionrunner.NewCell(logger, "big-cell", client, bigState)

			smallState := BuildCellState("the-zone", 100, 200, 20, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
			smallCell := auctionrunner.NewCell(logger, "small-cell", client, smallState)

			bigScore, err := defaultAuction.ScoreForTask(bigCell, task, 0.0)
			Expect(err).NotTo(HaveOccurred())
			smallScore, err := defaultAuction.ScoreForTask(smallCell, task, 0.0)
			Expect(err).NotTo(HaveOccurred())
			Expect(bigScore).To(BeNumerically("<", smallScore), "prefer Cells with more resources")
		})

		Context("Starting Containers", func() {
			var task *rep.Task
			var busyState, boredState rep.CellState
			var busyCell, boredCell *auctionrunner.Cell

			BeforeEach(func() {
				task = BuildTask("tg-big", "domain", linuxRootFSURL, 20, 20, 20, []string{}, []string{})

				busyState = BuildCellState("the-zone", 100, 200, 50, false, 10, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
				busyCell = auctionrunner.NewCell(logger, "busy-cell", client, busyState)

				boredState = BuildCellState("the-zone", 100, 200, 50, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
				boredCell = auctionrunner.NewCell(logger, "bored-cell", client, boredState)
			})

			It("factors in starting containers when a weight is provided", func() {
				startingContainerWeight := 0.25
				busyScore, err := defaultAuction.ScoreForTask(busyCell, task, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				boredScore, err := defaultAuction.ScoreForTask(boredCell, task, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				Expect(busyScore).To(BeNumerically(">", boredScore), "prefer Cells that have less starting containers")
			})

			It("ignores starting containers when a weight is not provided", func() {
				startingContainerWeight := 0.0
				busyScore, err := defaultAuction.ScoreForTask(busyCell, task, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				boredScore, err := defaultAuction.ScoreForTask(boredCell, task, startingContainerWeight)
				Expect(err).NotTo(HaveOccurred())
				Expect(busyScore).To(BeNumerically("==", boredScore), "ignore how many starting Containers a cell has")
			})
		})

		Context("when the task does not fit", func() {
			Context("because of memory constraints", func() {
				It("should error", func() {
					massiveMemoryTask := BuildTask("pg-new", "domain", linuxRootFSURL, 10000, 10, 1024, []string{}, []string{})
					score, err := defaultAuction.ScoreForTask(cell, massiveMemoryTask, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: memory"))
				})
			})

			Context("because of disk constraints", func() {
				It("should error", func() {
					massiveDiskTask := BuildTask("pg-new", "domain", linuxRootFSURL, 10, 10000, 1024, []string{}, []string{})
					score, err := defaultAuction.ScoreForTask(cell, massiveDiskTask, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: disk"))
				})
			})

			Context("because of container constraints", func() {
				It("should error", func() {
					task := BuildTask("pg-new", "domain", linuxRootFSURL, 10, 10, 10, []string{}, []string{})
					zeroState := BuildCellState("the-zone", 100, 100, 0, false, 0, linuxOnlyRootFSProviders, nil, []string{}, []string{}, []string{})
					zeroCell := auctionrunner.NewCell(logger, "zero-cell", client, zeroState)
					score, err := defaultAuction.ScoreForTask(zeroCell, task, 0.0)
					Expect(score).To(BeZero())
					Expect(err).To(MatchError("insufficient resources: containers"))
				})
			})
		})
	})
})
