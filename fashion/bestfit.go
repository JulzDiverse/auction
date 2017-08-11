package fashion //BestFit

import (
	ar "code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/rep"
)

func bestFit(c *ar.Cell, lrp *rep.LRP, startingContainerWeight float64) (float64, error) {
	err := c.state.ResourceMatch(&lrp.Resource)
	if err != nil {
		return 0, err
	}

	score := rep.NewScoreType(rep.BestFitFashion)

	resourceScore := score.Compute(&c.state, &lrp.Resource, startingContainerWeight)
	return resourceScore, nil
}
