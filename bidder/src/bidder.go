package bidder

import "net/http"

type Bidder struct {
}

func (b *Bidder) BidHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	})
}
