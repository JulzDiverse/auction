package auctionrunner

import (
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/lager"
)

type lrpAuctionFunc func(*Scheduler, []lrpByZone, *auctiontypes.LRPAuction) (*Cell, map[string]struct{})

func applyScoring() {

}

func runLRPAuction(s *Scheduler, filteredZones []lrpByZone, lrpAuction *auctiontypes.LRPAuction) (*Cell, map[string]struct{}) {
	var winnerCell *Cell
	winnerScore := 1e20

	problems := map[string]struct{}{"disk": struct{}{}, "memory": struct{}{}, "containers": struct{}{}}

	s.logger.Info("schedule-lrp-auction", lager.Data{"problems": problems})

	for zoneIndex, lrpByZone := range filteredZones {
		for _, cell := range lrpByZone.zone {
			score, err := cell.ScoreForLRP(&lrpAuction.LRP, s.startingContainerWeight)
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
