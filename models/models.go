package models

import "time"

type AuctionRequest struct {
	AuctionID string `json:"auction_id"`
}

type AuctionResponse struct {
	BidderID string  `json:"bidder_id"`
	Price    float64 `json:"price"`
}

// Can be used as Register bidder request.
type BidderConfig struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

type ListBiddersResponse struct {
	BidderConfigs []*BidderConfig `json:"bidder_configs"`
}

// Might be useful to have this info
type RegisterBidderResponse struct {
	RegisteredAt time.Time `json:"registered_at"`
	Success      bool      `json:"success"`
	Error        string    `json:"error"`
}
