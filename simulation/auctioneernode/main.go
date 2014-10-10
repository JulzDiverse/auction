package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_client"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry/yagnats"
)

var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var httpLookup = flag.String("httpLookup", "", "http lookup table")
var timeout = flag.Duration("timeout", time.Second, "timeout for nats responses")
var httpAddr = flag.String("httpAddr", "", "http address to listen on")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *natsAddrs == "" && *httpLookup == "" {
		panic("need nats addr or http lookup table")
	}

	if *natsAddrs != "" && *httpLookup != "" {
		panic("choose one: nats communication or http communication")
	}

	if *httpAddr == "" {
		panic("need http addr")
	}

	var repClient auctiontypes.RepPoolClient

	if *natsAddrs != "" {
		natsMembers := []string{}
		for _, addr := range strings.Split(*natsAddrs, ",") {
			uri := url.URL{
				Scheme: "nats",
				Host:   addr,
			}
			natsMembers = append(natsMembers, uri.String())
		}
		client, err := yagnats.Connect(natsMembers)
		if err != nil {
			log.Fatalln("no nats:", err)
		}

		repClient, err = auction_nats_client.New(client, *timeout, cf_lager.New("auctioneer-nats"))
		if err != nil {
			log.Fatalln("no rep client:", err)
		}
	}

	if *httpLookup != "" {
		lookupTable := map[string]string{}
		err := json.Unmarshal([]byte(*httpLookup), &lookupTable)
		if err != nil {
			log.Fatalln("couldn't parse lookup table:", err)
		}

		addressLookup := auction_http_client.AddressLookupFromMap(lookupTable)
		repClient = auction_http_client.New(&http.Client{
			Timeout: *timeout,
		}, cf_lager.New("auctioneer-http"), addressLookup)
	}

	http.HandleFunc("/start-auction", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequest auctiontypes.StartAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStartAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	http.HandleFunc("/stop-auction", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequest auctiontypes.StopAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStopAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe(*httpAddr, nil))
}
