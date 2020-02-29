package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	bidder "github.com/mridul-sahu/greedy_auction/bidder/src"
	"github.com/sirupsen/logrus"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Host       string `short:"h" long:"host" description:"Bidder Host" default:"localhost"`
	Port       int    `short:"p" long:"port" description:"Bidder Port" default:"8001"`
	Delay      int    `long:"delay" description:"Maximum delay(ms) to repond" default:"10"`
	RegisterAt string `long:"register-at" description:"Registeration Endpoint" default:"http://localhost:8000/v1/register"`
}

func main() {
	var opts Options
	if _, err := flags.ParseArgs(&opts, os.Args); err != nil {
		logrus.WithError(err).Fatalln("Could not parse input flags")
	}
	bidderController, err := bidder.NewBidder(
		opts.RegisterAt,
		fmt.Sprintf("http://%s:%d/v1/bid", opts.Host, opts.Port),
		time.Duration(opts.Delay)*time.Millisecond,
		logrus.StandardLogger())
	if err != nil {
		logrus.WithError(err).Fatalln("NewBidder Failed")
	}
	r := mux.NewRouter()
	r.Handle("/v1/bid", bidderController.BidHandler()).Methods("POST")

	allowedMethods := handlers.AllowedMethods([]string{"POST"})
	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", opts.Host, opts.Port), handlers.CORS(allowedMethods)(r)); err != nil {
		logrus.WithError(err).Fatalln("ListenAndServe Failed")
	}
}
