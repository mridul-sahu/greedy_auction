package auctioneer

import (
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

		registerHandlerLogger.WithField("bidder_id", bidderConfig.ID).WithField("bidder_endpoint", bidderConfig.Endpoint).Debugln("Registering Bidder")

		ac.activeBiddersLock.RLock()
		registerHandlerLogger.Traceln("Got Active Bidders Read Lock")
		_, exsists := ac.activeBidders[bidderConfig.ID]
		ac.activeBiddersLock.RUnlock()
		registerHandlerLogger.Traceln("Active Bidders Read Unlock")

		ret := &models.RegisterBidderResponse{}

		if exsists {
			ret.Success = false
			ret.Error = "Already Registered"
			registerHandlerLogger.WithField("bidder_id", bidderConfig.ID).WithField("bidder_endpoint", bidderConfig.Endpoint).Warningln("Bidder Not Added")
		} else {
			ac.activeBiddersLock.Lock()
			registerHandlerLogger.Traceln("Got Active Bidders Write Lock")
			ac.activeBidders[bidderConfig.ID] = bidderConfig.Endpoint
			ret.RegisteredAt = time.Now()
			ret.Success = true
			ac.activeBiddersLock.Unlock()
			registerHandlerLogger.WithField("bidder_id", bidderConfig.ID).WithField("bidder_endpoint", bidderConfig.Endpoint).Infoln("Bidder Added")
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

func (act *Auctioneer) ListHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})
}

func (act *Auctioneer) BidHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})
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
