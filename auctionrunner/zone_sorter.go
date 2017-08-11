package auctionrunner

import (
	"sort"
)

type LrpByZone struct {
	Zone      Zone
	Instances int
}

type zoneSorterByInstances struct {
	zones []LrpByZone
}

func (s zoneSorterByInstances) Len() int           { return len(s.zones) }
func (s zoneSorterByInstances) Swap(i, j int)      { s.zones[i], s.zones[j] = s.zones[j], s.zones[i] }
func (s zoneSorterByInstances) Less(i, j int) bool { return s.zones[i].Instances < s.zones[j].Instances }

func accumulateZonesByInstances(zones map[string]Zone, processGuid string) []LrpByZone {
	lrpZones := []LrpByZone{}

	for _, zone := range zones {
		instances := 0
		for _, cell := range zone {
			for i := range cell.State.LRPs {
				if cell.State.LRPs[i].ProcessGuid == processGuid {
					instances++
				}
			}
		}
		lrpZones = append(lrpZones, LrpByZone{zone, instances})
	}

	return lrpZones
}

func sortZonesByInstances(zones []LrpByZone) []LrpByZone {
	sorter := zoneSorterByInstances{zones: zones}
	sort.Sort(sorter)
	return sorter.zones
}
