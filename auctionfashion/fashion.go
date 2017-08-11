package auctionfashion

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
