package bidder

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"

	"github.com/mridul-sahu/greedy_auction/models"
)

type Bidder struct {
	responseDelay time.Duration
	id            string
	endpoint      string
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

// Utility Func for json body to type. Not responsible to close body.
func valueFromBody(body io.ReadCloser, value interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, value)
}
