# Greedy Auction

### Auction system which can auction bids and select the winning bid such that it always responds before a specific time interval.

## Entities

### Bidder
```
Takes a bid request and responds with a bid with some delay. Responsible for registering itself to an auctioneer.
```

### Auctioneer
```
Responsible for carring out the bid rounds. Takes an auction-request, carries out the bid rounds by querying the bidders for bids and chooses highest bidder within some limited time frame and responds with it.
```

## Communication
```
All communications happen via request/response structure, might come back to this later.
```

## To Look Into
```
1. In case of high number of bidders, might want to bring up more auctioneers in a tree like structure with bidders being leaf nodes and auctioneers being the intermediaries.

2. Make this README better. 
```