package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	auctioneer "github.com/mridul-sahu/greedy_auction/auctioneer/src"
	"github.com/sirupsen/logrus"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Host     string `short:"h" long:"host" description:"Auctioneer Host" default:"localhost"`
	Port     int    `short:"p" long:"port" description:"Auctioneer Port" default:"8000"`
	MaxDelay int    `long:"max-delay" description:"Maximum delay(ms) to repond" default:"200"`
}

func main() {
	var opts Options
	if _, err := flags.ParseArgs(&opts, os.Args); err != nil {
		logrus.WithError(err).Fatalln("Could not parse input flags")
	}
	auctionController := auctioneer.NewAcutioneer(time.Duration(opts.MaxDelay)*time.Millisecond, logrus.StandardLogger())
	r := mux.NewRouter()
	r.Handle("/v1/bid", auctionController.BidHandler()).Methods("POST")
	r.Handle("/v1/bidders", auctionController.ListHandler()).Methods("GET")
	r.Handle("/v1/register", auctionController.RegisterHandler()).Methods("POST")

	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST"})
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", opts.Host, opts.Port), handlers.CORS(allowedMethods)(r)); err != nil {
		logrus.WithError(err).Fatalln("ListenAndServe Failed")
	}
}
