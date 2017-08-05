package auctionrunner

import (
	"code.cloudfoundry.org/rep"
)

type ScoringFunc func(*Cell, *rep.LRP, float64) (float64, error)

func (c *Cell) CallForBid(lrp *rep.LRP, startingContainerWeight float64, sf ScoringFunc) (float64, error) {
	score, err := sf(c, lrp, startingContainerWeight)
	return score, err
}

func utilize(c *Cell, lrp *rep.LRP, startingContainerWeight float64) (float64, error) {
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
