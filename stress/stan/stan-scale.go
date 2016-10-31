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

	"math/rand"

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
	DefaultClientID           = "scale"
	DefaultConnectWait        = 120 * time.Second
)

func usage() {
	log.Fatalf("Usage: stan-scale [-s server (%s)] [--tls] [-id CLIENT_ID] [-np NUM_PUBLISHERS] [-ns NUM_SUBSCRIBERS] [-n NUM_MSGS] [-ms MESSAGE_SIZE] [-nc NUMCONNS] [-csv csvfile] [-mpa MAX_NUMBER_OF_PUBLISHED_ACKS_INFLIGHT] [-io] [-a] <subject>\n", nats.DefaultURL)
}

func disconnectedHandler(nc *nats.Conn) {
	if nc.LastError() != nil {
		log.Fatalf("connection %q has been disconnected: %v",
			nc.Opts.Name, nc.LastError())
	}
}

func reconnectedHandler(nc *nats.Conn) {
	log.Fatalf("connection %q reconnected to NATS Server at %q",
		nc.Opts.Name, nc.ConnectedUrl())
}

func closedHandler(nc *nats.Conn) {
	log.Fatalf("connection %q has been closed", nc.Opts.Name)
}

func errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	log.Fatalf("asynchronous error on connection %s, subject %s: %s",
		nc.Opts.Name, sub.Subject, err)
}

type subTrack struct {
	sync.Mutex
	subsMap map[string]int
}

func newSubTrack() *subTrack {
	newst := &subTrack{}
	newst.subsMap = make(map[string]int)
	return newst
}

// global subscription tracker
var st *subTrack

func (s *subTrack) initSubscriber(subject string) {
	s.Lock()
	defer s.Unlock()
	// for future recycling
	if _, ok := s.subsMap[subject]; ok {
		s.subsMap[subject]++
	} else {
		s.subsMap[subject] = 1
	}
}

func (s *subTrack) completeSubscriber(subject string) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.subsMap[subject]; ok {
		s.subsMap[subject]--
		if s.subsMap[subject] == 0 {
			delete(s.subsMap, subject)
		}
	}
}

func (s *subTrack) printUnfinishedCount() bool {
	s.Lock()
	defer s.Unlock()

	log.Printf("Remaining Subscribers (%d)\n", len(s.subsMap))

	return len(s.subsMap) == 0
}

func (s *subTrack) printUnfinishedDetail(max int) {

	i := 0
	for subj, remaining := range s.subsMap {
		log.Printf("    %s;%d", subj, remaining)
		i++
		if i == max {
			break
		}
	}
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
		opts.Name = "conn-" + string(i)
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

var currentSubjCount int
var useUniqueSubjects bool

func resetSubjects() {
	currentSubjCount = 0
}

// TODO:  is recycling subjects necessary?
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
var verbose bool

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
	var vb = flag.Bool("v", false, "Verbose")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		usage()
	}

	useUniqueSubjects = *uniqueSubjs
	verbose = *vb

	st = newSubTrack()

	// Setup the option block
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(*urls, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}

	opts.Secure = *tls
	opts.AsyncErrorCB = errorHandler
	opts.DisconnectedCB = disconnectedHandler
	opts.ReconnectedCB = reconnectedHandler
	opts.ClosedCB = closedHandler

	if err := buildConns(*numConns, &opts); err != nil {
		log.Fatalf("Unable to create connections: %v", err)
	}

	benchmark = bench.NewBenchmark("NATS Streaming", *numSubs, *numPubs)

	var startwg sync.WaitGroup
	var pubwg sync.WaitGroup
	var subwg sync.WaitGroup

	subwg.Add(*numSubs)
	pubwg.Add(*numPubs)

	// Run Subscribers first
	startwg.Add(*numSubs)
	for i := 0; i < *numSubs; i++ {
		subID := fmt.Sprintf("%s-sub-%d", *clientID, i)
		go runSubscriber(&startwg, &subwg, opts, *numMsgs, *messageSize, *ignoreOld, subID, getNextSubject(args[0], *numSubs))
	}
	startwg.Wait()

	log.Printf("Starting scaling test [msgs=%d, msgsize=%d, pubs=%d, subs=%d]\n", *numMsgs, *messageSize, *numPubs, *numSubs)
	// Now Publishers
	startwg.Add(*numPubs)
	pubCounts := bench.MsgsPerClient(*numMsgs, *numPubs)
	for i := 0; i < *numPubs; i++ {
		pubID := fmt.Sprintf("%s-pub-%d", *clientID, i)
		go runPublisher(&startwg, &pubwg, opts, pubCounts[i], *messageSize, *async, pubID, *maxPubAcks, args[0], *numSubs)
	}

	startwg.Wait()
	pubwg.Wait()

	log.Println("Done publishing.")

	isFinished := st.printUnfinishedCount()
	// 10 mins is a long time to wait, but for stress tests see if they recover.
	for i := 0; i < 600 && !isFinished; i++ {
		st.printUnfinishedDetail(5)
		time.Sleep(1 * time.Second)
		isFinished = st.printUnfinishedCount()
	}

	subwg.Wait()
	benchmark.Close()
	fmt.Print(benchmark.Report())

	if len(*csvFile) > 0 {
		csv := benchmark.CSV()
		ioutil.WriteFile(*csvFile, []byte(csv), 0644)
		fmt.Printf("Saved metric data in csv file %s\n", *csvFile)
	}
}

func publishMsgs(snc stan.Conn, msg []byte, async bool, numMsgs int, subj string) {
	var published int

	if async {
		ch := make(chan bool)
		acb := func(lguid string, err error) {
			if err != nil {
				log.Fatalf("Publish error: %v\n", err)
			}
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

// publishUniqueMsgs distributes messages evenly across subjects
func publishMsgsOnUniqueSubjects(snc stan.Conn, msg []byte, async bool, numMsgs int, subj string, numSubs int) {
	var published int

	ch := make(chan bool)
	acb := func(lguid string, err error) {
		if err != nil {
			log.Fatalf("Publish error: %v\n", err)
		}
		published++
		if published >= numMsgs {
			ch <- true
		}
	}

	for i := 0; i < numMsgs; i++ {
		resetSubjects()
		for j := 0; j < numSubs; j++ {
			sub := getNextSubject(subj, numSubs)
			if async {
				_, err := snc.PublishAsync(sub, msg, acb)
				if err != nil {
					log.Fatalf("Publish error: %v\n", err)
				}
			} else {
				err := snc.Publish(sub, msg)
				if err != nil {
					log.Fatalf("Publish error: %v\n", err)
				}
				published++
			}
		}
	}

	// wait for publishers
	if async {
		<-ch
	}
}

func runPublisher(startwg, donewg *sync.WaitGroup, opts nats.Options, numMsgs int, msgSize int, async bool, pubID string, maxPubAcksInflight int, subj string, numSubs int) {

	var snc stan.Conn
	var err error

	nc := getNextNatsConn()
	if nc == nil {
		snc, err = stan.Connect("test-cluster", pubID, stan.MaxPubAcksInflight(maxPubAcksInflight), stan.ConnectWait(DefaultConnectWait))
	} else {
		snc, err = stan.Connect("test-cluster", pubID,
			stan.MaxPubAcksInflight(maxPubAcksInflight), stan.NatsConn(nc), stan.ConnectWait(DefaultConnectWait))
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
		publishMsgsOnUniqueSubjects(snc, msg, async, numMsgs, subj, numSubs)
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
		snc, err = stan.Connect("test-cluster", subID, stan.ConnectWait(DefaultConnectWait))
	} else {
		snc, err = stan.Connect("test-cluster", subID, stan.NatsConn(nc), stan.ConnectWait(DefaultConnectWait))
	}
	if err != nil {
		log.Fatalf("Subscriber %s can't connect: %v\n", subID, err)
	}

	time.Sleep(time.Duration(rand.Intn(10000)) * time.Millisecond)

	ch := make(chan bool)
	start := time.Now()

	st.initSubscriber(subj)
	received := 0
	mcb := func(msg *stan.Msg) {
		received++
		if received >= numMsgs {
			/*f verbose {
				log.Printf("Done receiving on %s.\n", msg.Subject)
			}*/
			st.completeSubscriber(subj)
			ch <- true
		}
	}

	if ignoreOld {
		snc.Subscribe(subj, mcb, stan.AckWait(time.Second*120))
	} else {
		snc.Subscribe(subj, mcb, stan.DeliverAllAvailable(), stan.AckWait(time.Second*120))
	}
	startwg.Done()

	<-ch
	benchmark.AddSubSample(bench.NewSample(numMsgs, msgSize, start, time.Now(), snc.NatsConn()))
	snc.Close()
	donewg.Done()
}
