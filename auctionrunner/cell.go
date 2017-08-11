package auctionrunner

import (
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/rep"
)

const LocalityOffset = 1000

type Cell struct {
	logger lager.Logger
	Guid   string
	client rep.Client
	State  rep.CellState

	workToCommit rep.Work
}

func NewCell(logger lager.Logger, guid string, client rep.Client, state rep.CellState) *Cell {
	return &Cell{
		logger: logger,
		Guid:   guid,
		client: client,
		State:  state,
	}
}

func (c *Cell) StartingContainerCount() int {
	return c.State.StartingContainerCount
}

func (c *Cell) MatchRootFS(rootFS string) bool {
	return c.State.MatchRootFS(rootFS)
}

func (c *Cell) MatchVolumeDrivers(volumeDrivers []string) bool {
	return c.State.MatchVolumeDrivers(volumeDrivers)
}

func (c *Cell) MatchPlacementTags(placementTags []string) bool {
	// fmt.Printf("cell to match against is [%s]\n", c.Guid)
	return c.State.MatchPlacementTags(placementTags)
}

func (c *Cell) ReserveLRP(lrp *rep.LRP) error {
	err := c.State.ResourceMatch(&lrp.Resource)
	if err != nil {
		return err
	}

	c.State.AddLRP(lrp)
	c.workToCommit.LRPs = append(c.workToCommit.LRPs, *lrp)
	return nil
}

func (c *Cell) ReserveTask(task *rep.Task) error {
	err := c.State.ResourceMatch(&task.Resource)
	if err != nil {
		return err
	}

	c.State.AddTask(task)
	c.workToCommit.Tasks = append(c.workToCommit.Tasks, *task)
	return nil
}

func (c *Cell) Commit() rep.Work {
	if len(c.workToCommit.LRPs) == 0 && len(c.workToCommit.Tasks) == 0 {
		return rep.Work{}
	}

	failedWork, err := c.client.Perform(c.logger, c.workToCommit)
	if err != nil {
		c.logger.Error("failed-to-commit", err, lager.Data{"cell-guid": c.Guid})
		//an error may indicate partial failure
		//in this case we don't reschedule work in order to make sure we don't
		//create duplicates of things -- we'll let the converger figure things out for us later
		return rep.Work{}
	}
	return failedWork
}
