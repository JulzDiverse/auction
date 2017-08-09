package auctionrunner

type AuctionTypeFunc func(*AuctionType)

type AuctionType struct {
	AuctionLot     *AuctionLot
	AuctionFilters []*AuctionFilter
}

func NewAuctionType(atf AuctionTypeFunc) *AuctionType {
	var at AuctionType
	atf(&at)
	return &at
}

func DefaultAuction(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	filters := []*AuctionFilter{defaultFilter}
	at.AuctionLot = NewAuctionLot(UtilizationLot)
	at.AuctionFilters = filters
}

func BestFitFashion(at *AuctionType) {
	defaultFilter := NewAuctionFilter(DefaultFilter)
	filters := []*AuctionFilter{defaultFilter}
	at.AuctionLot = NewAuctionLot(BestFit)
	at.AuctionFilters = filters
}
