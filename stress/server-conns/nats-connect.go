package main

import (
	"flag"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"sync"

	"github.com/nats-io/gnatsd/test"
	"github.com/nats-io/nats"
)

func usage() {
	log.Fatalf("Usage: nats-connect [-s server] [-c count] [-u username]\n")
}

func runTest(url, user string, count int64) {
	var wg sync.WaitGroup

	log.Printf("\nTest %s, %d connections.", user, count)
	srv, _ := test.RunServerWithConfig("./gnatsd.conf")

	var disconnectTime time.Time
	var reconnectTime time.Time

	opts := nats.DefaultOptions
	opts.Timeout = time.Second * 600
	opts.Url = url
	opts.User = user
	opts.Password = "password"

	disconnectCount := int64(0)

	opts.DisconnectedCB = func(c *nats.Conn) {
		if atomic.AddInt64(&disconnectCount, 1) == 1 {
			disconnectTime = time.Now()
		}
	}

	reconnectCount := int64(0)

	opts.ReconnectedCB = func(c *nats.Conn) {
		if atomic.AddInt64(&reconnectCount, 1) == count {
			reconnectTime = time.Now()
		}
		wg.Done()
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
	log.Printf("Total Connect time:   %v", time.Now().Sub(connStartTime))

	// Bounce the server
	wg.Add(int(count))
	srv.Shutdown()
	srv, _ = test.RunServerWithConfig("./gnatsd.conf")
	defer srv.Shutdown()

	// wait for all connections to reconnect
	wg.Wait()

	log.Printf("Total Reconnect time: %v", reconnectTime.Sub(disconnectTime))
}

func main() {
	var url = flag.String("s", "nats://localhost:4442", "The nats server URLs (separated by comma)")
	var count = flag.Int("c", 50, "# of connections")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	runTest(*url, "cost_0", int64(*count))
	runTest(*url, "cost_4", int64(*count))
	runTest(*url, "cost_8", int64(*count))
	runTest(*url, "cost_11", int64(*count))
	runTest(*url, "cost_16", int64(*count))

	log.Println("Done.")
}
