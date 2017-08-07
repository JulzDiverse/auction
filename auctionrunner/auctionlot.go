package auctionrunner

import (
	"code.cloudfoundry.org/rep"
)

type ScoringFunc func(*Cell, *rep.LRP, float64) (float64, error)
type AuctionLotType func(*AuctionLot)

func (c *Cell) CallForBid(lrp *rep.LRP, startingContainerWeight float64, sf ScoringFunc) (float64, error) {
	score, err := sf(c, lrp, startingContainerWeight)
	return score, err
}

/*
  A "Lot" is the item for sale in an auction. In this case the item could be
  - "Utilization" (standard)
  - "Fitness" (For each job find the cell with less workload)
  - or other
*/
type AuctionLot struct {
	ScoreForLRP ScoringFunc
}

func NewAuctionLot(alt AuctionLotType) *AuctionLot {
	var at AuctionLot
	alt(&at)
	return &at
}

func UtilizationLot(at *AuctionLot) {
	at.ScoreForLRP = scoreForLRP
}

func scoreForLRP(c *Cell, lrp *rep.LRP, startingContainerWeight float64) (float64, error) {
	err := c.state.ResourceMatch(&lrp.Resource)
	if err != nil {
		return 0, err
	}

	numberOfInstancesWithMatchingProcessGuid := 0
	for i := range c.state.LRPs {
		if c.state.LRPs[i].ProcessGuid == lrp.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	localityScore := LocalityOffset * numberOfInstancesWithMatchingProcessGuid

	resourceScore := c.state.ComputeScore(&lrp.Resource, startingContainerWeight)
	return resourceScore + float64(localityScore), nil
}
