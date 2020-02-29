package auctioneer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/mridul-sahu/greedy_auction/models"
	"github.com/sirupsen/logrus"
)

type Auctioneer struct {
	allowedResponseDelay time.Duration
	activeBidders        map[string]string
	inactiveBidders      map[string]string
	client               *http.Client

	logger            *logrus.Entry
	activeBiddersLock sync.RWMutex
}

func (ac *Auctioneer) RegisterHandler() http.Handler {
	registerHandlerLogger := ac.logger.WithField("operation", "register-handler")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		bidderConfig := models.BidderConfig{}
		if err := valueFromBody(r.Body, &bidderConfig); err != nil || bidderConfig.ID == "" || bidderConfig.Endpoint == "" {
			registerHandlerLogger.WithError(err).Errorln("Invalid Request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bLogger := registerHandlerLogger.WithField("bidder_id", bidderConfig.ID).WithField("bidder_endpoint", bidderConfig.Endpoint)
		bLogger.Debugln("Registering Bidder")

		ac.activeBiddersLock.RLock()
		registerHandlerLogger.Traceln("Got Active Bidders Read Lock")
		_, exsists := ac.activeBidders[bidderConfig.ID]
		ac.activeBiddersLock.RUnlock()
		registerHandlerLogger.Traceln("Active Bidders Read Unlock")

		ret := &models.RegisterBidderResponse{}

		if exsists {
			ret.Success = false
			ret.Error = "Already Registered"
			bLogger.Warningln("Bidder Not Added")
		} else {
			ac.activeBiddersLock.Lock()
			registerHandlerLogger.Traceln("Got Active Bidders Write Lock")
			ac.activeBidders[bidderConfig.ID] = bidderConfig.Endpoint
			ret.RegisteredAt = time.Now()
			ret.Success = true
			ac.activeBiddersLock.Unlock()
			bLogger.Infoln("Bidder Added")
		}

		resp, err := json.Marshal(ret)
		if err != nil {
			// Shouldn't happen, but just in case.
			registerHandlerLogger.WithError(err).Errorln("Error Marshalling RegisterBidderResponse to JSON")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		registerHandlerLogger.Debugln("RegisterBidderResponse sent")
	})
}

func (ac *Auctioneer) ListHandler() http.Handler {
	listHandlerLogger := ac.logger.WithField("operation", "list-handler")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listHandlerLogger.Debugln("Getting Bidder Configs")

		ac.activeBiddersLock.RLock()
		listHandlerLogger.Traceln("Got Active Bidders Read Lock")
		n := len(ac.activeBidders)
		bidderConfigs := make([]*models.BidderConfig, n)
		i := 0
		for bidderID, endpoint := range ac.activeBidders {
			bidderConfigs[i] = &models.BidderConfig{
				ID:       bidderID,
				Endpoint: endpoint,
			}
			i++
		}
		ac.activeBiddersLock.RUnlock()
		listHandlerLogger.Traceln("Active Bidders Read Unlock")

		resp, err := json.Marshal(&models.ListBiddersResponse{
			BidderConfigs: bidderConfigs,
		})
		if err != nil {
			// Shouldn't happen, but just in case.
			listHandlerLogger.WithError(err).Errorln("Error Marshalling ListBiddersResponse to JSON")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		listHandlerLogger.Debugf("Sent List(%d) of Bidder Configs\n", n)
	})
}

func (ac *Auctioneer) BidHandler() http.Handler {
	bidHandlerLogger := ac.logger.WithField("operation", "bid-handler")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		auctionRequest := models.AuctionRequest{}
		if err := valueFromBody(r.Body, &auctionRequest); err != nil {
			bidHandlerLogger.WithError(err).Errorln("Invalid Request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionLogger := bidHandlerLogger.WithField("auction_id", auctionRequest.AuctionID)
		auctionLogger.Debugln("Getting Bids")
		bidderID, bid := ac.GetMaxBid(r.Context(), &auctionRequest, ac.allowedResponseDelay)
		if bid == -1 || bidderID == "" {
			auctionLogger.WithField("event", "no-response").Infoln("No Bids Found")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp, err := json.Marshal(&models.AuctionResponse{
			BidderID: bidderID,
			Price:    bid,
		})
		if err != nil {
			// Shouldn't happen, but just in case.
			auctionLogger.WithError(err).Errorln("Error Marshalling AuctionResponse to JSON")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		auctionLogger.WithField("bidder_id", bidderID).WithField("bid", bid).Traceln("Best Bid")
		auctionLogger.Infoln("Sent Best Bid")
	})
}

// Promises to return at timeout with the best possible bid found
func (ac *Auctioneer) GetMaxBid(ctxt context.Context, auctionRequest *models.AuctionRequest, timeout time.Duration) (string, float64) {
	bidLogger := ac.logger.WithField("operation", "get-max-bid").WithField("auction_id", auctionRequest.AuctionID)

	bidderID := ""
	maxBid := -1.0
	reciecvedBidsCount := 0

	timeoutChan := time.After(timeout)
	bidsContext, cancel := context.WithCancel(ctxt)
	bidsChan := ac.GetBids(bidsContext, auctionRequest)
	for {
		select {
		case <-timeoutChan:
			cancel()
			bidLogger.Debugln("Timeout")
			return bidderID, maxBid
		case resp := <-bidsChan:
			reciecvedBidsCount += 1
			if resp.Price > maxBid {
				bidderID = resp.BidderID
				maxBid = resp.Price
				bidLogger.WithField("event", "best-bid").Tracef("Recieved Bid ID: %s, Price: %f\n", resp.BidderID, resp.Price)
			}
			bidLogger.WithField("bid_count", reciecvedBidsCount).Tracef("Recieved Bid ID: %s, Price: %f\n", resp.BidderID, resp.Price)
		}
	}
}

func (ac *Auctioneer) GetBids(ctxt context.Context, auctionRequest *models.AuctionRequest) <-chan *models.AuctionResponse {
	getBidsLogger := ac.logger.WithField("operation", "get-bids").WithField("auction_id", auctionRequest.AuctionID)

	// Maybe some other buffer size based on requirements
	bidsChan := make(chan *models.AuctionResponse, 1000)

	// Generate bids and push them to the channel
	go func(bidsChan chan<- *models.AuctionResponse) {
		ac.activeBiddersLock.RLock()
		getBidsLogger.Traceln("Got Active Bidders Read Lock")
		n := len(ac.activeBidders)

		if n == 0 {
			getBidsLogger.WithField("event", "no-bidders").Debugln("No bidders Found")
			ac.activeBiddersLock.RUnlock()
			getBidsLogger.Traceln("Active Bidders Read Unlock")
			return
		}

		bidders := make(map[string]string, n)
		for bidderID, endpoint := range ac.activeBidders {
			bidders[bidderID] = endpoint
		}
		ac.activeBiddersLock.RUnlock()
		getBidsLogger.Traceln("Active Bidders Read Unlock")

		auctionRequestBytes, err := json.Marshal(auctionRequest)
		if err != nil {
			// Shouldn't happen, but just in case
			getBidsLogger.WithError(err).Errorln("Error Marshalling AuctionRequest")
			return
		}

		getBidsLogger.Debugln("Sending Bid Requests")
		for _, endpoint := range bidders {
			// Request for bids from bidders.
			// May want to limit number of routines allowed as each is sending a http request.
			go func(bidderEndpoint string) {
				bidderLogger := getBidsLogger.WithField("bidder_endpoint", bidderEndpoint)

				request, err := http.NewRequestWithContext(ctxt, "POST", bidderEndpoint, bytes.NewBuffer(auctionRequestBytes))
				if err != nil {
					// Shouldn't happen normally, something is wrong.
					bidderLogger.WithError(err).Errorln("Error Forming Request")
					return
				}

				bidderLogger.Traceln("Sending Request")
				resp, err := ac.client.Do(request)
				if err != nil {
					bidderLogger.WithError(err).Errorln("Error Requesting Bid")
					return
				}
				defer resp.Body.Close()

				var auctionResonse *models.AuctionResponse
				if err := valueFromBody(resp.Body, &auctionResonse); err != nil {
					bidderLogger.WithError(err).Errorln("Invalid Response")
					return
				}
				// If the context is still valid, try sending bid to be considered
				select {
				case bidsChan <- auctionResonse:
					bidderLogger.Traceln("Sent Response")
				case <-ctxt.Done():
					bidderLogger.Traceln("Context Cancelled")
				}

			}(endpoint)
		}
	}(bidsChan)

	return bidsChan
}

func NewAcutioneer(allowedResponseDelay time.Duration, logger *logrus.Logger) *Auctioneer {
	return &Auctioneer{
		allowedResponseDelay: allowedResponseDelay,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
		activeBidders:   make(map[string]string),
		inactiveBidders: make(map[string]string),
		logger:          logger.WithField("entity", "auctioneer"),
	}
}

// Utility Func for json body to type. Not responsible to close body.
func valueFromBody(body io.ReadCloser, value interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, value)
}
