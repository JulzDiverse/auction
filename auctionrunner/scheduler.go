package auctionrunner

import (
	"sort"
	"sync"

	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/workpool"
)

type Zone []*Cell

type Scheduler struct {
	workPool                      *workpool.WorkPool
	zones                         map[string]Zone
	clock                         clock.Clock
	logger                        lager.Logger
	startingContainerWeight       float64
	startingContainerCountMaximum int // <=0 means no limit
	auctionType                   *AuctionType
}

func NewScheduler(
	workPool *workpool.WorkPool,
	zones map[string]Zone,
	clock clock.Clock,
	logger lager.Logger,
	startingContainerWeight float64,
	startingContainerCountMaximum int,
	auctionType *AuctionType,
) *Scheduler {
	return &Scheduler{
		workPool:                      workPool,
		zones:                         zones,
		clock:                         clock,
		logger:                        logger,
		startingContainerWeight:       startingContainerWeight,
		startingContainerCountMaximum: startingContainerCountMaximum,
		auctionType:                   auctionType, //CHANGE
	}
}

/*
Schedule takes in a set of job requests (LRP start auctions and task starts) and
assigns the work to available cells according to the diego scoring algorithm. The
scheduler is single-threaded.  It determines scheduling of jobs one at a time so
that each calculation reflects available resources correctly.  It commits the
work in batches at the end, for better network performance.  Schedule returns
AuctionResults, indicating the success or failure of each requested job.
*/
func (s *Scheduler) Schedule(auctionRequest auctiontypes.AuctionRequest) auctiontypes.AuctionResults {
	results := auctiontypes.AuctionResults{}

	if len(s.zones) == 0 {
		results.FailedLRPs = auctionRequest.LRPs
		for i, _ := range results.FailedLRPs {
			results.FailedLRPs[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		results.FailedTasks = auctionRequest.Tasks
		for i, _ := range results.FailedTasks {
			results.FailedTasks[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		return s.markResults(results)
	}

	var successfulLRPs = map[string]*auctiontypes.LRPAuction{}
	var lrpStartAuctionLookup = map[string]*auctiontypes.LRPAuction{}
	var successfulTasks = map[string]*auctiontypes.TaskAuction{}
	var taskAuctionLookup = map[string]*auctiontypes.TaskAuction{}
	var currentInflightContainerStarts int

	for _, zone := range s.zones {
		for _, cell := range zone {
			currentInflightContainerStarts += cell.StartingContainerCount()
		}
	}

	sort.Sort(SortableLRPAuctions(auctionRequest.LRPs))
	sort.Sort(SortableTaskAuctions(auctionRequest.Tasks))

	lrpsBeforeTasks, lrpsAfterTasks := splitLRPS(auctionRequest.LRPs)

	auctionLRP := func(lrpsToAuction []auctiontypes.LRPAuction) {
		for i := range lrpsToAuction {
			lrpAuction := &lrpsToAuction[i]
			lrpStartAuctionLookup[lrpAuction.Identifier()] = lrpAuction

			if s.exceededInflightContainerCreation(currentInflightContainerStarts) {
				s.logger.Info(
					"exceeded-max-inflight-container-creation",
					lager.Data{
						"max-inflight": s.startingContainerCountMaximum,
						"lrp-guid":     lrpAuction.Identifier(),
					},
				)
				lrpAuction.PlacementError = auctiontypes.ErrorExceededInflightCreation.Error()
				results.FailedLRPs = append(results.FailedLRPs, *lrpAuction)
				continue
			}

			successfulStart, err := s.scheduleLRPAuction(lrpAuction)
			if err != nil {
				lrpAuction.PlacementError = err.Error()
				results.FailedLRPs = append(results.FailedLRPs, *lrpAuction)
			} else {
				successfulLRPs[successfulStart.Identifier()] = successfulStart
				currentInflightContainerStarts++
			}
		}
	}

	auctionLRP(lrpsBeforeTasks)

	for i := range auctionRequest.Tasks {
		taskAuction := &auctionRequest.Tasks[i]
		taskAuctionLookup[taskAuction.Identifier()] = taskAuction

		if s.exceededInflightContainerCreation(currentInflightContainerStarts) {
			s.logger.Info(
				"exceeded-max-inflight-container-creation",
				lager.Data{
					"max-inflight": s.startingContainerCountMaximum,
					"task-guid":    taskAuction.Identifier(),
				},
			)
			taskAuction.PlacementError = auctiontypes.ErrorExceededInflightCreation.Error()
			results.FailedTasks = append(results.FailedTasks, *taskAuction)
			continue
		}

		successfulTask, err := s.scheduleTaskAuction(taskAuction, s.startingContainerWeight)
		if err != nil {
			taskAuction.PlacementError = err.Error()
			results.FailedTasks = append(results.FailedTasks, *taskAuction)
		} else {
			successfulTasks[successfulTask.Identifier()] = successfulTask
			currentInflightContainerStarts++
		}
	}

	auctionLRP(lrpsAfterTasks)

	failedWorks := s.commitCells()
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.LRPs {
			identifier := failedStart.Identifier()
			delete(successfulLRPs, identifier)

			s.logger.Info("lrp-failed-to-be-placed", lager.Data{"lrp-guid": failedStart.Identifier()})
			results.FailedLRPs = append(results.FailedLRPs, *lrpStartAuctionLookup[identifier])
		}

		for _, failedTask := range failedWork.Tasks {
			identifier := failedTask.Identifier()
			delete(successfulTasks, identifier)

			s.logger.Info("task-failed-to-be-placed", lager.Data{"task-guid": failedTask.Identifier()})
			results.FailedTasks = append(results.FailedTasks, *taskAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPs {
		s.logger.Info("lrp-added-to-cell", lager.Data{"lrp-guid": successfulStart.Identifier(), "cell-guid": successfulStart.Winner})
		results.SuccessfulLRPs = append(results.SuccessfulLRPs, *successfulStart)
	}
	for _, successfulTask := range successfulTasks {
		s.logger.Info("task-added-to-cell", lager.Data{"task-guid": successfulTask.Identifier(), "cell-guid": successfulTask.Winner})
		results.SuccessfulTasks = append(results.SuccessfulTasks, *successfulTask)
	}
	return s.markResults(results)
}

func (s *Scheduler) markResults(results auctiontypes.AuctionResults) auctiontypes.AuctionResults {
	now := s.clock.Now()
	for i := range results.FailedLRPs {

		results.FailedLRPs[i].Attempts++
	}
	for i := range results.FailedTasks {
		results.FailedTasks[i].Attempts++
	}
	for i := range results.SuccessfulLRPs {
		results.SuccessfulLRPs[i].Attempts++
		results.SuccessfulLRPs[i].WaitDuration = now.Sub(results.SuccessfulLRPs[i].QueueTime)
	}
	for i := range results.SuccessfulTasks {
		results.SuccessfulTasks[i].Attempts++
		results.SuccessfulTasks[i].WaitDuration = now.Sub(results.SuccessfulTasks[i].QueueTime)
	}

	return results
}

func splitLRPS(lrps []auctiontypes.LRPAuction) ([]auctiontypes.LRPAuction, []auctiontypes.LRPAuction) {
	const pivot = 0

	for idx, lrp := range lrps {
		if lrp.Index > pivot {
			return lrps[:idx], lrps[idx:]
		}
	}

	return lrps[:0], lrps[0:]
}

func (s *Scheduler) commitCells() []rep.Work {
	wg := &sync.WaitGroup{}
	for _, cells := range s.zones {
		wg.Add(len(cells))
	}

	lock := &sync.Mutex{}
	failedWorks := []rep.Work{}

	for _, cells := range s.zones {
		for _, cell := range cells {
			cell := cell
			s.workPool.Submit(func() {
				defer wg.Done()
				failedWork := cell.Commit()

				lock.Lock()
				failedWorks = append(failedWorks, failedWork)
				lock.Unlock()
			})
		}
	}

	wg.Wait()
	return failedWorks
}

func (s *Scheduler) scheduleLRPAuction(lrpAuction *auctiontypes.LRPAuction) (*auctiontypes.LRPAuction, error) {
	var winnerCell *Cell

	zones := accumulateZonesByInstances(s.zones, lrpAuction.ProcessGuid)

	//*******START JULZ ******
	filteredZones, err := applyFilters(zones, lrpAuction, s.auctionType.AuctionFilters...)
	if err != nil {
		return nil, err
	}

	winnerCell, problems := s.runLRPAuction(filteredZones, lrpAuction)
	//*******END JULZ**********

	if winnerCell == nil {
		return nil, &rep.InsufficientResourcesError{Problems: problems}
	}

	err = winnerCell.ReserveLRP(&lrpAuction.LRP)
	if err != nil {
		s.logger.Error("lrp-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "lrp-guid": lrpAuction.Identifier()})
		return nil, err
	}

	winningAuction := lrpAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}

func (s *Scheduler) runLRPAuction(filteredZones []lrpByZone, lrpAuction *auctiontypes.LRPAuction) (*Cell, map[string]struct{}) {
	var winnerCell *Cell
	winnerScore := 1e20

	problems := map[string]struct{}{"disk": struct{}{}, "memory": struct{}{}, "containers": struct{}{}}

	s.logger.Info("schedule-lrp-auction", lager.Data{"problems": problems})

	for zoneIndex, lrpByZone := range filteredZones {
		for _, cell := range lrpByZone.zone {
			score, err := cell.CallForBid(&lrpAuction.LRP, s.startingContainerWeight, s.auctionType.ScoreForLRP)
			if err != nil {
				removeNonApplicableProblems(problems, err)
				s.logger.Info("schedule-lrp-auction-after-error", lager.Data{"problems": problems, "error": err})
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}

		// if (not last zone) && (this zone has the same # of instances as the next sorted zone)
		// acts as a tie breaker
		if zoneIndex+1 < len(filteredZones) &&
			lrpByZone.instances == filteredZones[zoneIndex+1].instances {
			continue
		}

		if winnerCell != nil {
			break
		}
	}
	return winnerCell, problems
}
func (s *Scheduler) scheduleTaskAuction(taskAuction *auctiontypes.TaskAuction, startingContainerWeight float64) (*auctiontypes.TaskAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	filteredZones := []Zone{}
	var zoneError error

	for _, zone := range s.zones {
		cells, err := filterCells(taskAuction.PlacementConstraint, &zone)
		if err != nil {
			_, isZoneErrorPlacementTagMismatchError := zoneError.(auctiontypes.PlacementTagMismatchError)
			_, isErrPlacementTagMismatchError := err.(auctiontypes.PlacementTagMismatchError)

			if isZoneErrorPlacementTagMismatchError ||
				(zoneError == auctiontypes.ErrorVolumeDriverMismatch && isErrPlacementTagMismatchError) ||
				zoneError == auctiontypes.ErrorCellMismatch || zoneError == nil {
				zoneError = err
			}
			continue
		}

		filteredZones = append(filteredZones, Zone(cells))
	}

	if len(filteredZones) == 0 {
		return nil, zoneError
	}

	problems := map[string]struct{}{"disk": struct{}{}, "memory": struct{}{}, "containers": struct{}{}}

	for _, zone := range filteredZones {
		for _, cell := range zone {
			score, err := cell.ScoreForTask(&taskAuction.Task, startingContainerWeight)
			if err != nil {
				removeNonApplicableProblems(problems, err)
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}
	}

	if winnerCell == nil {
		return nil, &rep.InsufficientResourcesError{Problems: problems}
	}

	err := winnerCell.ReserveTask(&taskAuction.Task)
	if err != nil {
		s.logger.Error("task-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "task-guid": taskAuction.Identifier()})
		return nil, err
	}

	winningAuction := taskAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}

// removeNonApplicableProblems modifies the 'problems' map to remove any problems that didn't show up on err.
//
// The list of problems to report should only consist of the problems that exist on every cell
// For example, if there is not enough memory on one cell and not enough disk on another, we should
// not call out memory or disk as being a specific problem.
func removeNonApplicableProblems(problems map[string]struct{}, err error) {
	if ierr, ok := err.(rep.InsufficientResourcesError); ok {
		for problem, _ := range problems {
			if _, ok := ierr.Problems[problem]; !ok {
				delete(problems, problem)
			}
		}
	}
}

func (s *Scheduler) exceededInflightContainerCreation(currentInflight int) bool {
	return s.startingContainerCountMaximum > 0 && currentInflight >= s.startingContainerCountMaximum
}
