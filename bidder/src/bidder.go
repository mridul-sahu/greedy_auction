package bidder

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mridul-sahu/greedy_auction/models"
	"github.com/sirupsen/logrus"
)

type Bidder struct {
	responseDelay time.Duration
	id            string
	endpoint      string
	client        *http.Client

	logger *logrus.Entry
}

func (b *Bidder) BidHandler() http.Handler {
	bidHandlerLogger := b.logger.WithField("operation", "bid-handler")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionRequest := models.AuctionRequest{}

		if err := valueFromBody(r.Body, &auctionRequest); err != nil {
			bidHandlerLogger.WithError(err).Errorln("Invalid Request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bidHandlerLogger.Debugf("Delaying Bid by %d milliseconds\n", b.responseDelay.Milliseconds())
		time.Sleep(b.responseDelay)

		bid := rand.Float64() * 10000
		resp, err := json.Marshal(&models.AuctionResponse{
			BidderID: b.id,
			Price:    bid,
		})
		if err != nil {
			// Shouldn't happen, but just in case something goes wrong.
			bidHandlerLogger.WithError(err).Errorln("Error Marshalling AuctionResponse to JSON")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		bidHandlerLogger.WithField("bid", bid).Infoln("Bid Sent")
	})
}

func (b *Bidder) register(endpoint string) error {
	registrationLogger := b.logger.WithField("register_endpoint", endpoint)

	bidderConfigBytes, err := json.Marshal(&models.BidderConfig{
		ID:       b.id,
		Endpoint: b.endpoint,
	})
	if err != nil {
		// Shouldn't happen, but just in case something goes wrong.
		registrationLogger.WithError(err).Errorln("Error Marshalling BidderConfig")
		return err
	}

	delay := 1

	// Trying exponential backoff. Maybe control this from args.
	for {
		registrationLogger.Infoln("Trying to register")

		request, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bidderConfigBytes))
		if err != nil {
			// Shouldn't happen normally, something is wrong, don't want to try again.
			registrationLogger.WithError(err).Errorln("Error Forming Request")
			return err
		}

		registrationLogger.Debugln("Sending Request")
		resp, err := b.client.Do(request)
		if err != nil {
			if delay >= 64 {
				break
			}
			registrationLogger.WithError(err).Errorln("Error Registering Bidder")
			registrationLogger.Debugf("Sleeping for %d seconds\n", delay)
			time.Sleep(time.Duration(delay) * time.Second)

			delay *= 2
			continue
		}

		// Got a response.
		defer resp.Body.Close()

		var registerResponse *models.RegisterBidderResponse
		if err := valueFromBody(resp.Body, &registerResponse); err != nil {
			registrationLogger.WithError(err).Errorln("Invalid Response")
			return err
		}

		if !registerResponse.Success {
			// Not allowed to register. Maybe try again. TODO:- decide to try again.
			err := errors.New(registerResponse.Error)
			registrationLogger.WithError(err).Errorln("Error Registering Bidder")
			return err
		}
		// Maybe store this if needed.
		registrationLogger.Infof("Bidder Registered at time: %v\n", registerResponse.RegisteredAt)
		return nil
	}

	err = errors.New("Maximum Reties Reached")
	registrationLogger.WithError(err).Errorln("Error Registering Bidder")
	return err
}

// This tries to registers and return a bidder, returns error if registeration fails.
func NewBidder(registerationEndpoint string, bidEndpoint string, responseDelay time.Duration, logger *logrus.Logger) (*Bidder, error) {
	bidderID := uuid.New().String()
	ret := &Bidder{
		responseDelay: responseDelay,
		id:            bidderID,
		endpoint:      bidEndpoint,
		client: &http.Client{
			Timeout: time.Second * 5,
		},

		logger: logger.WithField("entity", "bidder").WithField("bidder_id", bidderID),
	}
	return ret, ret.register(registerationEndpoint)
}

// Utility Func for json body to type. Not responsible to close body.
func valueFromBody(body io.ReadCloser, value interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, value)
}
