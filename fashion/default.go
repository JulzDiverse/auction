package fashion //Default: Classic Diego Auction-Algorithm

import (
	ar "code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"
)

func defaultFilter(s *ar.AuctionFilter) {
	s.ZoneFilter = filterZones
	s.CellFilter = filterCells
}

func defaultTaskFilter(s *ar.AuctionTaskFilter) {
	s.ZoneFilter = filterZonesForTaks
	s.CellFilter = filterCells
}

func scoreForLRP(c *ar.Cell, lrp *rep.LRP, startingContainerWeight float64) (float64, error) {
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

	localityScore := ar.LocalityOffset * numberOfInstancesWithMatchingProcessGuid
	score := rep.NewScoreType(rep.WorstFitFashion)

	resourceScore := score.Compute(&c.state, &lrp.Resource, startingContainerWeight)

	return resourceScore + float64(localityScore), nil
}

func scoreForTask(c *ar.Cell, task *rep.Task, startingContainerWeight float64) (float64, error) {
	err := c.state.ResourceMatch(&task.Resource)
	if err != nil {
		return 0, err
	}

	localityScore := ar.LocalityOffset * len(c.state.Tasks)
	resourceScore := c.state.ComputeScore(&task.Resource, startingContainerWeight)
	return resourceScore + float64(localityScore), nil
}

func filterZones(zones []ar.LrpByZone, lrpAuction *auctiontypes.LRPAuction, filterCells ar.CellFilter) ([]ar.LrpByZone, error) {
	filteredZones := []LrpByZone{}
	var zoneError error

	for _, lrpZone := range zones {
		cells, err := filterCells(lrpAuction.PlacementConstraint, &lrpZone.zone)
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

		filteredZone := lrpByZone{
			zone:      Zone(cells),
			instances: lrpZone.instances,
		}
		filteredZones = append(filteredZones, filteredZone)
	}

	if len(filteredZones) == 0 {
		return nil, zoneError
	}

	/*
	  moved sorting from scheduleLRPAuction() to filterZones as the filter
	  should deside how zones/cells are sorted
	*/
	filteredZones = sortZonesByInstances(filteredZones)
	return filteredZones, nil
}

func filterCells(pc rep.PlacementConstraint, z *ar.Zone) ([]*ar.Cell, error) {
	var cells = make([]*Cell, 0, len(*z))
	err := auctiontypes.ErrorCellMismatch

	for _, cell := range *z {
		if cell.MatchRootFS(pc.RootFs) {
			if err == auctiontypes.ErrorCellMismatch {
				err = auctiontypes.ErrorVolumeDriverMismatch
			}

			if cell.MatchVolumeDrivers(pc.VolumeDrivers) {
				if err == auctiontypes.ErrorVolumeDriverMismatch {
					err = auctiontypes.NewPlacementTagMismatchError(pc.PlacementTags)
				}

				if cell.MatchPlacementTags(pc.PlacementTags) {
					err = nil
					cells = append(cells, cell)
				}
			}
		}
	}

	return cells, err
}

func filterZonesForTaks(zones map[string]ar.Zone, taskAuction *auctiontypes.TaskAuction, filterCells ar.CellFilter) ([]ar.Zone, error) {
	filteredZones := []ar.Zone{}
	var zoneError error
	for _, zone := range zones {
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
	return filteredZones, nil
}
