package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"sync"

	"github.com/nats-io/gnatsd/test"
	"github.com/nats-io/nats"
)

func usage() {
	log.Fatalf("Usage: nats-connect [-s server] [-c count] [-u username]\n")
}

var csvOutput *bool
var doReconnect *bool

func runTest(url, user string, count int64) {
	var wg sync.WaitGroup

	srv, _ := test.RunServerWithConfig("./gnatsd.conf")

	opts := nats.DefaultOptions
	opts.Timeout = time.Second * 600
	opts.Url = url
	opts.User = user
	opts.Password = "password"

	opts.ReconnectedCB = func(c *nats.Conn) {
		if *doReconnect {
			wg.Done()
		}
	}

	var connStartTime = time.Now()

	connList := make([]*nats.Conn, 0, count)
	wg.Add(int(count))

	// create connections simultaneously to make the test complete faster.
	for i := int64(0); i < count; i++ {
		go func() {
			// randomize connect times to prevent connection read errors
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(2000)))
			nc, err := opts.Connect()
			if err != nil {
				log.Fatalf("Can't connect: %v\n", err)
			}
			connList = append(connList, nc)
			wg.Done()
		}()
	}
	defer func() {
		for _, nc := range connList {
			nc.Close()
		}
	}()

	// wait for all connections to connect
	wg.Wait()
	totalConnectTime := time.Now().Sub(connStartTime)

	var reconnectWaitTime time.Duration

	// Bounce the server
	if *doReconnect {
		wg.Add(int(count))

		srv.Shutdown()
		disconnectTime := time.Now()
		srv, _ = test.RunServerWithConfig("./gnatsd.conf")

		// wait for all connections to reconnect
		wg.Wait()
		reconnectWaitTime = time.Now().Sub(disconnectTime)
	}
	defer srv.Shutdown()

	if *csvOutput {
		log.Printf("%s,%f", user, totalConnectTime.Seconds())
	} else {
		if *doReconnect {
			log.Printf("user=%s, connect time=%v, reconnect time=%v",
				user, totalConnectTime, reconnectWaitTime)
		} else {
			log.Printf("user=%s, connect time=%v",
				user, totalConnectTime)
		}
	}
}

func main() {
	var url = flag.String("s", "nats://localhost:4442", "The nats server URLs (separated by comma)")
	var count = flag.Int("c", 50, "# of connections")
	csvOutput = flag.Bool("csv", false, "csv output")
	doReconnect = flag.Bool("rc", false, "do reconnect")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	fmt.Printf("Connections: %d\n", *count)

	runTest(*url, "cost_0", int64(*count))
	runTest(*url, "cost_4", int64(*count))
	runTest(*url, "cost_8", int64(*count))
	runTest(*url, "cost_9", int64(*count))
	runTest(*url, "cost_10", int64(*count))
	runTest(*url, "cost_11", int64(*count))
	runTest(*url, "cost_12", int64(*count))
	runTest(*url, "cost_13", int64(*count))
	runTest(*url, "cost_14", int64(*count))
	runTest(*url, "cost_15", int64(*count))
	runTest(*url, "cost_16", int64(*count))
	runTest(*url, "cost_17", int64(*count))
	runTest(*url, "cost_18", int64(*count))
	runTest(*url, "cost_19", int64(*count))
	runTest(*url, "cost_20", int64(*count))

	log.Println("Done.")
}
