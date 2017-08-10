package auctionrunner

import "code.cloudfoundry.org/rep"

type LRPScoringFunc func(*Cell, *rep.LRP, float64) (float64, error)
type AuctionTypeFunc func(*AuctionType)

type AuctionType struct {
	ScoreForLRP    LRPScoringFunc
	AuctionFilters []*AuctionFilter
}

func NewAuctionType(atf AuctionTypeFunc) *AuctionType {
	var at AuctionType
	atf(&at)
	return &at
}

func DefaultAuction(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	filters := []*AuctionFilter{defaultFilter}
	at.ScoreForLRP = scoreForLRP
	at.AuctionFilters = filters
}

func BestFit(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	filters := []*AuctionFilter{defaultFilter}
	at.ScoreForLRP = bestFit
	at.AuctionFilters = filters
}

func (c *Cell) CallForBid(lrp *rep.LRP, startingContainerWeight float64, sf LRPScoringFunc) (float64, error) {
	score, err := sf(c, lrp, startingContainerWeight)
	return score, err
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
	score := rep.NewScoreType(rep.WorstFitFashion)

	resourceScore := score.Compute(&c.state, &lrp.Resource, startingContainerWeight)

	return resourceScore + float64(localityScore), nil
}

func bestFit(c *Cell, lrp *rep.LRP, startingContainerWeight float64) (float64, error) {
	err := c.state.ResourceMatch(&lrp.Resource)
	if err != nil {
		return 0, err
	}

	score := rep.NewScoreType(rep.BestFitFashion)

	resourceScore := score.Compute(&c.state, &lrp.Resource, startingContainerWeight)
	return resourceScore, nil
}
