package auctionrunner

import "code.cloudfoundry.org/rep"

type ScoringFunc func(*Cell, *rep.LRP, float64) (float64, error)
type ScoringFuncTask func(*Cell, *rep.Task, float64) (float64, error)
type AuctionTypeFunc func(*AuctionType)

type AuctionType struct {
	ScoreForLRP        ScoringFunc
	AuctionFilters     []*AuctionFilter
	ScoreForTask       ScoringFuncTask
	AuctionTaskFilters []*AuctionTaskFilter
}

func NewAuctionType(atf AuctionTypeFunc) *AuctionType {
	var at AuctionType
	atf(&at)
	return &at
}

func DefaultAuction(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	defaultTaskFilter := NewAuctionTaskFilter(DefaultTaskFilter)
	filters := []*AuctionFilter{defaultFilter}
	taskFilters := []*AuctionTaskFilter{defaultTaskFilter}
	at.ScoreForLRP = scoreForLRP
	at.AuctionFilters = filters
	at.ScoreForTask = scoreForTask
	at.AuctionTaskFilters = taskFilters
}

func BestFit(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	defaultTaskFilter := NewAuctionTaskFilter(DefaultTaskFilter)
	filters := []*AuctionFilter{defaultFilter}
	taskFilters := []*AuctionTaskFilter{defaultTaskFilter}
	at.ScoreForLRP = bestFit
	at.AuctionFilters = filters
	at.ScoreForTask = scoreForTask
	at.AuctionTaskFilters = taskFilters
}

func (c *Cell) CallForLRPBid(lrp *rep.LRP, startingContainerWeight float64, sf ScoringFunc) (float64, error) {
	score, err := sf(c, lrp, startingContainerWeight)
	return score, err
}

func (c *Cell) CallForTaskBid(task *rep.Task, startingContainerWeight float64, sf ScoringFuncTask) (float64, error) {
	score, err := sf(c, task, startingContainerWeight)
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

func scoreForTask(c *Cell, task *rep.Task, startingContainerWeight float64) (float64, error) {
	err := c.state.ResourceMatch(&task.Resource)
	if err != nil {
		return 0, err
	}

	localityScore := LocalityOffset * len(c.state.Tasks)
	resourceScore := c.state.ComputeScore(&task.Resource, startingContainerWeight)
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
