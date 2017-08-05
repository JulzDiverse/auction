package auctionrunner

import (
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"
)

type LrpZoneFilter func([]lrpByZone, *auctiontypes.LRPAuction, LrpCellFilter) ([]lrpByZone, error)
type LrpCellFilter func(rep.PlacementConstraint, *Zone) ([]*Cell, error)
type FilterTypeFunc func(*Selector)

type Selector struct {
	ZoneFilter LrpZoneFilter
	CellFilter LrpCellFilter
}

func NewSelector(option FilterTypeFunc) *Selector {
	var s Selector
	option(&s)
	return &s
}

func UsingClassicFilter(s *Selector) {
	s.ZoneFilter = filterZones
	s.CellFilter = filterCells
}

func applyFilters(zones []lrpByZone, lrpAuction *auctiontypes.LRPAuction, selectors ...*Selector) ([]lrpByZone, error) {
	filteredZones := zones
	for _, selector := range selectors {
		tmpFilteredZones, err := selector.ZoneFilter(filteredZones, lrpAuction, selector.CellFilter)
		if err != nil {
			return nil, err
		}
		filteredZones = tmpFilteredZones
	}
	return filteredZones, nil
}

func filterZones(zones []lrpByZone, lrpAuction *auctiontypes.LRPAuction, filterCells LrpCellFilter) ([]lrpByZone, error) {
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

	/*
	  moved sorting from scheduleLRPAuction() to filterZones as the filter
	  should deside how zones/cells are sorted
	*/
	filteredZones = sortZonesByInstances(filteredZones)
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
