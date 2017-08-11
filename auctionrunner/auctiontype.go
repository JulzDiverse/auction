package auctionrunner

import (
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"
)

type ScoringFunc func(*Cell, *rep.LRP, float64) (float64, error)
type ScoringFuncTask func(*Cell, *rep.Task, float64) (float64, error)
type AuctionTypeFunc func(*AuctionType)
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

type AuctionType struct {
	ScoreForLRP        ScoringFunc
	AuctionFilters     []*AuctionFilter
	ScoreForTask       ScoringFuncTask
	AuctionTaskFilters []*AuctionTaskFilter
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
