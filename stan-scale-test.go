// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats"
	"github.com/nats-io/nats/bench"
)

// Some sane defaults
const (
	DefaultNumMsgs            = 100000
	DefaultNumPubs            = 1
	DefaultNumSubs            = 0
	DefaultNumConns           = -1
	DefaultAsync              = false
	DefaultMessageSize        = 128
	DefaultIgnoreOld          = false
	DefaultMaxPubAcksInflight = 1000
	DefaultClientID           = "benchmark"
)

func usage() {
	log.Fatalf("Usage: stan-scale-test [-s server (%s)] [--tls] [-id CLIENT_ID] [-np NUM_PUBLISHERS] [-ns NUM_SUBSCRIBERS] [-n NUM_MSGS] [-ms MESSAGE_SIZE] [-nc NUMCONNS] [-csv csvfile] [-mpa MAX_NUMBER_OF_PUBLISHED_ACKS_INFLIGHT] [-io] [-a] <subject>\n", nats.DefaultURL)
}

var conns []*nats.Conn

func buildConns(count int, opts *nats.Options) error {
	var err error
	conns = nil
	err = nil

	// make a conn pool to use
	if count < 0 {
		return nil
	}
	conns = make([]*nats.Conn, count)
	for i := 0; i < count; i++ {
		conns[i], err = opts.Connect()
	}
	return err
}

var currentConn int
var connLock sync.Mutex

func getNextNatsConn() *nats.Conn {
	connLock.Lock()

	if conns == nil {
		connLock.Unlock()
		return nil
	}
	if currentConn == len(conns) {
		currentConn = 0
	}
	nc := conns[currentConn]
	currentConn++

	connLock.Unlock()

	return nc
}

var currentSubjCount int = 0
var useUniqueSubjects bool = false

func getNextSubject(baseSubject string, max int) string {
	if !useUniqueSubjects {
		return baseSubject
	}
	rv := fmt.Sprintf("%s.%d", baseSubject, currentSubjCount)
	currentSubjCount++
	if currentSubjCount == max {
		currentSubjCount = 0
	}

	return rv
}

var benchmark *bench.Benchmark

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	var tls = flag.Bool("tls", false, "Use TLS Secure Connection")
	var numConns = flag.Int("nc", DefaultNumConns, "Number of connections to use (default is publishers+subscribers)")
	var numPubs = flag.Int("np", DefaultNumPubs, "Number of Concurrent Publishers")
	var numSubs = flag.Int("ns", DefaultNumSubs, "Number of Concurrent Subscribers")
	var numMsgs = flag.Int("n", DefaultNumMsgs, "Number of Messages to Publish")
	var async = flag.Bool("a", DefaultAsync, "Async Message Publishing")
	var messageSize = flag.Int("ms", DefaultMessageSize, "Message Size in bytes.")
	var ignoreOld = flag.Bool("io", DefaultIgnoreOld, "Subscribers Ignore Old Messages")
	var maxPubAcks = flag.Int("mpa", DefaultMaxPubAcksInflight, "Max number of published acks in flight")
	var clientID = flag.String("id", DefaultClientID, "Benchmark process base client ID.")
	var csvFile = flag.String("csv", "", "Save bench data to csv file")
	var uniqueSubjs = flag.Bool("us", false, "Use unique subjects")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		usage()
	}

	useUniqueSubjects = *uniqueSubjs
	// Setup the option block
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(*urls, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}
	opts.Secure = *tls

	if err := buildConns(*numConns, &opts); err != nil {
		log.Fatal("Unable to create connections: %v", err)
	}

	benchmark = bench.NewBenchmark("NATS Streaming", *numSubs, *numPubs)

	var startwg sync.WaitGroup
	var donewg sync.WaitGroup

	donewg.Add(*numPubs + *numSubs)

	// Run Subscribers first
	startwg.Add(*numSubs)
	for i := 0; i < *numSubs; i++ {
		subID := fmt.Sprintf("%s-sub-%d", *clientID, i)
		go runSubscriber(&startwg, &donewg, opts, *numMsgs, *messageSize, *ignoreOld, subID, getNextSubject(args[0], *numSubs))
	}
	startwg.Wait()

	// Now Publishers
	startwg.Add(*numPubs)
	pubCounts := bench.MsgsPerClient(*numMsgs, *numPubs)
	for i := 0; i < *numPubs; i++ {
		pubID := fmt.Sprintf("%s-pub-%d", *clientID, i)
		go runPublisher(&startwg, &donewg, opts, pubCounts[i], *messageSize, *async, pubID, *maxPubAcks, args[0], *numSubs)
	}

	log.Printf("Starting benchmark [msgs=%d, msgsize=%d, pubs=%d, subs=%d]\n", *numMsgs, *messageSize, *numPubs, *numSubs)

	startwg.Wait()
	donewg.Wait()

	benchmark.Close()
	fmt.Print(benchmark.Report())

	if len(*csvFile) > 0 {
		csv := benchmark.CSV()
		ioutil.WriteFile(*csvFile, []byte(csv), 0644)
		fmt.Printf("Saved metric data in csv file %s\n", *csvFile)
	}
}

func publishMsgs(snc stan.Conn, msg []byte, async bool, numMsgs int, subj string) {
	var published int = 0

	if async {
		ch := make(chan bool)
		acb := func(lguid string, err error) {
			published++
			if published >= numMsgs {
				ch <- true
			}
		}
		for i := 0; i < numMsgs; i++ {
			_, err := snc.PublishAsync(subj, msg, acb)
			if err != nil {
				log.Fatal(err)
			}
		}
		<-ch
	} else {
		for i := 0; i < numMsgs; i++ {
			err := snc.Publish(subj, msg)
			if err != nil {
				log.Fatal(err)
			}
			published++
		}
	}

}

func runPublisher(startwg, donewg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int, async bool, pubID string, maxPubAcksInflight int, subj string, numSubs int) {

	var snc stan.Conn
	var err error

	nc := getNextNatsConn()
	if nc == nil {
		snc, err = stan.Connect("test-cluster", pubID, stan.MaxPubAcksInflight(maxPubAcksInflight))
	} else {
		snc, err = stan.Connect("test-cluster", pubID,
			stan.MaxPubAcksInflight(maxPubAcksInflight), stan.NatsConn(nc))
	}
	if err != nil {
		log.Fatalf("Publisher %s can't connect: %v\n", pubID, err)
	}

	startwg.Done()

	var msg []byte
	if msgSize > 0 {
		msg = make([]byte, msgSize)
	}

	start := time.Now()

	if useUniqueSubjects {
		for i := 0; i < numSubs; i++ {
			publishMsgs(snc, msg, async, numMsgs, fmt.Sprintf("%s.%d", subj, i))
		}
	} else {
		publishMsgs(snc, msg, async, numMsgs, subj)
	}

	benchmark.AddPubSample(bench.NewSample(numMsgs, msgSize, start, time.Now(), snc.NatsConn()))
	snc.Close()
	donewg.Done()
}

func runSubscriber(startwg, donewg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int, ignoreOld bool, subID, subj string) {
	var snc stan.Conn
	var err error

	nc := getNextNatsConn()
	if nc == nil {
		snc, err = stan.Connect("test-cluster", subID)
	} else {
		snc, err = stan.Connect("test-cluster", subID, stan.NatsConn(nc))
	}
	if err != nil {
		log.Fatalf("Subscriber %s can't connect: %v\n", subID, err)
	}

	ch := make(chan bool)
	start := time.Now()

	received := 0
	mcb := func(msg *stan.Msg) {
		received++
		if received >= numMsgs {
			ch <- true
		}
	}

	if ignoreOld {
		snc.Subscribe(subj, mcb)
	} else {
		snc.Subscribe(subj, mcb, stan.DeliverAllAvailable())
	}
	startwg.Done()

	<-ch
	benchmark.AddSubSample(bench.NewSample(numMsgs, msgSize, start, time.Now(), snc.NatsConn()))
	snc.Close()
	donewg.Done()
}
