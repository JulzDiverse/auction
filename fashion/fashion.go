package fashion

import (
	ar "code.cloudfoundry.org/auction/auctionrunner"
)

func NewAuctionType(atf ar.AuctionTypeFunc) *ar.AuctionType {
	var at ar.AuctionType
	atf(&at)
	return &at
}

func newAuctionFilter(option ar.FilterTypeFunc) *ar.AuctionFilter {
	var s ar.AuctionFilter
	option(&s)
	return &s
}

func newAuctionTaskFilter(option ar.TaskFilterTypeFunc) *ar.AuctionTaskFilter {
	var s ar.AuctionTaskFilter
	option(&s)
	return &s
}

//*******************  Auction Types  ***********************************
func DefaultAuction(at *ar.AuctionType) {
	df := newAuctionFilter(defaultFilter)
	dtf := newAuctionTaskFilter(defaultTaskFilter)
	filters := []*AuctionFilter{df}
	taskFilters := []*AuctionTaskFilter{dtf}
	at.ScoreForLRP = scoreForLRP
	at.AuctionFilters = filters
	at.ScoreForTask = scoreForTask
	at.AuctionTaskFilters = taskFilters
}

func BestFit(at *ar.AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	defaultTaskFilter := NewAuctionTaskFilter(DefaultTaskFilter)
	filters := []*AuctionFilter{defaultFilter}
	taskFilters := []*AuctionTaskFilter{defaultTaskFilter}
	at.ScoreForLRP = bestFit
	at.AuctionFilters = filters
	at.ScoreForTask = scoreForTask
	at.AuctionTaskFilters = taskFilters
}
