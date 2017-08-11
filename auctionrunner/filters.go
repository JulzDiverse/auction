package auctionrunner

import (
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"
)

type TaskZoneFilter func(map[string]Zone, *auctiontypes.TaskAuction, CellFilter) ([]Zone, error)
type LrpZoneFilter func([]LrpByZone, *auctiontypes.LRPAuction, CellFilter) ([]LrpByZone, error)
type CellFilter func(rep.PlacementConstraint, *Zone) ([]*Cell, error)

type FilterTypeFunc func(*AuctionFilter)
type TaskFilterTypeFunc func(*AuctionTaskFilter)

type AuctionFilter struct {
	ZoneFilter     LrpZoneFilter
	CellFilter     CellFilter
	TaskZoneFilter TaskZoneFilter
}

type AuctionTaskFilter struct {
	CellFilter CellFilter
	ZoneFilter TaskZoneFilter
}

func NewAuctionFilter(option FilterTypeFunc) *AuctionFilter {
	var s AuctionFilter
	option(&s)
	return &s
}

func NewAuctionTaskFilter(option TaskFilterTypeFunc) *AuctionTaskFilter {
	var s AuctionTaskFilter
	option(&s)
	return &s
}

func DefaultFilter(s *AuctionFilter) {
	s.ZoneFilter = filterZones
	s.CellFilter = filterCells
}

func DefaultTaskFilter(s *AuctionTaskFilter) {
	s.ZoneFilter = filterZonesForTaks
	s.CellFilter = filterCells
}

func applyLRPFilters(zones []LrpByZone, lrpAuction *auctiontypes.LRPAuction, auctionFilters ...*AuctionFilter) ([]LrpByZone, error) {
	filteredZones := zones
	for _, filter := range auctionFilters {
		tmpFilteredZones, err := filter.ZoneFilter(filteredZones, lrpAuction, filter.CellFilter)
		if err != nil {
			return nil, err
		}
		filteredZones = tmpFilteredZones
	}
	return filteredZones, nil
}

func applyTaskFilters(zones map[string]Zone, lrpAuction *auctiontypes.TaskAuction, auctionFilters ...*AuctionTaskFilter) ([]Zone, error) {
	filteredZones := []Zone{}
	for _, filter := range auctionFilters {
		tmpFilteredZones, err := filter.ZoneFilter(zones, lrpAuction, filter.CellFilter)
		if err != nil {
			return nil, err
		}
		filteredZones = tmpFilteredZones
	}
	return filteredZones, nil
}

func filterZones(zones []LrpByZone, lrpAuction *auctiontypes.LRPAuction, filterCells CellFilter) ([]LrpByZone, error) {
	filteredZones := []LrpByZone{}
	var zoneError error

	for _, lrpZone := range zones {
		cells, err := filterCells(lrpAuction.PlacementConstraint, &lrpZone.Zone)
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

		filteredZone := LrpByZone{
			Zone:      Zone(cells),
			Instances: lrpZone.Instances,
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

func filterZonesForTaks(zones map[string]Zone, taskAuction *auctiontypes.TaskAuction, filterCells CellFilter) ([]Zone, error) {
	filteredZones := []Zone{}
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
