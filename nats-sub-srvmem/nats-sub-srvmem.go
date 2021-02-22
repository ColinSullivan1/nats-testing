package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	gnatsd "github.com/nats-io/nats-server/server"
	"github.com/nats-io/nats.go"
)

// Quick and dirty app to read self reported memory from a server
// based on subscriptions.

var trace bool
var verbose bool
var singleSubj bool

func verbosef(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

func connect(urls string) (*nats.Conn, error) {
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(urls, ",")

	opts.AsyncErrorCB = errorHandler
	opts.DisconnectedCB = disconnectedHandler
	opts.ReconnectedCB = reconnectedHandler
	opts.ClosedCB = closedHandler

	return opts.Connect()
}

func createSubscription(nc *nats.Conn, subject string) (*nats.Subscription, error) {

	// unique callback per sub, add some code
	mh := func(msg *nats.Msg) {
		log.Printf("Received message on %s.\n", msg.Subject)
	}

	verbosef("Subscribing to to %s.\n", subject)
	s, err := nc.Subscribe(subject, mh)
	if err != nil {
		return nil, err
	}

	verbosef("Subscribed to %s.\n", subject)
	return s, err
}

func usage() {
	log.Fatal("Usage: scale-client-emulator -cfg <config file> [-v VERBOSE]")
}

func disconnectedHandler(nc *nats.Conn) {
	if nc.LastError() != nil {
		log.Printf("connection disconnected: %v\n",
			nc.LastError())
	}
}

func reconnectedHandler(nc *nats.Conn) {
	log.Printf("reconnected to NATS Server at %q\n",
		nc.ConnectedUrl())
}

func closedHandler(nc *nats.Conn) {
	verbosef("connection has been closed\n")
}

func errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	log.Fatalf("asynchronous error on connection %s, subject %s: %s\n",
		nc.Opts.Name, sub.Subject, err)
}

func run(url, monURL, subject string, subcount int, isVerbose, isTraceVerbose bool) {
	verbose = isVerbose
	if isTraceVerbose {
		verbose = true
		trace = true
	}
	nc, err := connect(url)
	if err != nil {
		log.Fatalf("Couldn't connect:  %v\n", err)
	}

	var subj string
	for i := 1; i <= subcount; i++ {
		if singleSubj {
			subj = subject
		} else {
			subj = fmt.Sprintf("%s.%d", subject, i)
		}
		_, err = createSubscription(nc, subj)
		if err != nil {
			log.Fatalf("Couldn't subscribe:  %v\n", err)
		}
		if i%500000 == 0 {
			if err := printServerRSS(monURL, i); err != nil {
				log.Fatalf("Couldn't get server memory: %v", err)
			}
		}
	}

	if err := nc.Flush(); err != nil {
		log.Fatalf("Couldn't flush:  %v\n", err)
	}
}

var firstPrint bool

func printServerRSS(monURL string, subcount int) error {

	varz := &gnatsd.Varz{}
	HTTPClient := &http.Client{}

	verbosef("Getting data from monitor url: %s\n", monURL)
	resp, err := HTTPClient.Get(monURL)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("could not get stats from server: %v\n", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %v\n", err)
	}

	err = json.Unmarshal(body, &varz)
	if err != nil {
		return fmt.Errorf("could not unmarshal json: %v\n", err)
	}

	if !firstPrint {
		firstPrint = true
		log.Printf("subcount, mem(rss)")
	}
	log.Printf("%d, %d\n", subcount, varz.Mem)

	return nil
}

func main() {
	var url = flag.String("url", "nats://localhost:4222", "nats server url")
	var murl = flag.String("murl", "http://localhost:6060", "Monitor url")
	var su = flag.String("subject", "foo", "base subject to use")
	var sc = flag.Int("subcount", 30000000, "number of subscribers")
	var vb = flag.Bool("V", false, "Verbose")
	var tb = flag.Bool("DV", false, "Verbose/Trace")
	var ss = flag.Bool("oneSubj", false, "Use a single subject.")

	log.SetFlags(0)
	flag.Parse()

	singleSubj = *ss
	mu := *murl + "/varz"

	run(*url, mu, *su, *sc, *vb, *tb)
}
