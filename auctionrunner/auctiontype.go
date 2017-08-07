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
	filters := []*AuctionFilter{
		NewAuctionFilter(DefaultFilter),
	}
	at.AuctionLot = NewAuctionLot(UtilizationLot)
	at.AuctionFilters = filters
}
