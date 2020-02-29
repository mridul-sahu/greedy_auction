package bidder

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mridul-sahu/greedy_auction/models"
)

type Bidder struct {
	responseDelay time.Duration
	id            string
	endpoint      string
	client        *http.Client
}

func (b *Bidder) BidHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionRequest := models.AuctionRequest{}

		if err := valueFromBody(r.Body, &auctionRequest); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		time.Sleep(b.responseDelay)

		resp, err := json.Marshal(&models.AuctionResponse{
			BidderID: b.id,
			Price:    rand.Float64() * 10000,
		})
		if err != nil {
			// Shouldn't happen, but just in case something goes wrong.
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	})
}

func (b *Bidder) register(endpoint string) error {
	return nil
}

// This tries to registers and return a bidder, returns error if registeration fails.
func NewBidder(registerationEndpoint string, bidEndpoint string, responseDelay time.Duration) (*Bidder, error) {
	ret := &Bidder{
		responseDelay: responseDelay,
		id:            uuid.New().String(),
		endpoint:      bidEndpoint,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
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
