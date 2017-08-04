package auctionrunner

import (
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"
)

type LrpZoneFilter func([]lrpByZone, *auctiontypes.LRPAuction) ([]lrpByZone, error)
type LrpCellFilter func(rep.PlacementConstraint, *Zone) ([]*Cell, error)

type Selector struct {
	ZoneFilter LrpZoneFilter
	CellFilter LrpCellFilter
}

func applyFilters(zones []lrpByZone, lrpAuction *auctiontypes.LRPAuction, selectors ...Selector) ([]lrpByZone, error) {
	filteredZones := zones
	for _, selector := range selectors {
		tmpFilteredZones, err := selector.ZoneFilter(filteredZones, lrpAuction)
		if err != nil {
			return nil, err
		}
		filteredZones = tmpFilteredZones
	}
	return filteredZones, nil
}

func filterZonesDifferent(zones []lrpByZone, lrpAuction *auctiontypes.LRPAuction) ([]lrpByZone, error) {

	//just a test
	return zones, nil
}

func filterZones(zones []lrpByZone, lrpAuction *auctiontypes.LRPAuction) ([]lrpByZone, error) {
	filteredZones := []lrpByZone{}
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

	return filteredZones, nil
}

func filterCells(pc rep.PlacementConstraint, z *Zone) ([]*Cell, error) {
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
