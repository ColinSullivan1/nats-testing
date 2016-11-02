// Copyright 2015 Apcera Inc. All rights reserved.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"math/rand"

	"github.com/nats-io/go-nats-streaming"
	"github.com/nats-io/nats"
)

// Some sane defaults
const (
	DefaultConnectWait    = 4 * time.Minute
	DefaultConfigFileName = "config.json"
	UniqueSubject         = "UNIQUE"
)

// NatsServerConnPool manages a pool of NATS connections
type NatsServerConnPool struct {
	sync.Mutex
	currentConn int
	conns       []*nats.Conn
}

var trace bool
var verbose bool

func verbosef(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

// NewNatsServerConnPool creates a NATS connection pool
func NewNatsServerConnPool(cfg *Config) (*NatsServerConnPool, error) {
	var err error

	opts := nats.DefaultOptions
	opts.Servers = strings.Split(cfg.ServerURLs, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}

	opts.Secure = cfg.UseTLS
	opts.AsyncErrorCB = errorHandler
	opts.DisconnectedCB = disconnectedHandler
	opts.ReconnectedCB = reconnectedHandler
	opts.ClosedCB = closedHandler

	if cfg.NumConns < 0 {
		return nil, nil
	}

	ncp := &NatsServerConnPool{}
	ncp.conns = make([]*nats.Conn, cfg.NumConns)
	for i := 0; i < cfg.NumConns; i++ {
		opts.Name = fmt.Sprintf("conn-%d", i)
		ncp.conns[i], err = opts.Connect()
		if err != nil {
			return nil, err
		}
	}

	return ncp, nil
}

// GetNextNatsConn returns an active NATS connection
func (p *NatsServerConnPool) GetNextNatsConn() *nats.Conn {
	p.Lock()
	defer p.Unlock()

	if p.conns == nil {
		return nil
	}

	if p.currentConn == len(p.conns) {
		p.currentConn = 0
	}
	nc := p.conns[p.currentConn]
	p.currentConn++

	return nc
}

// ClientSubConfig represents a subscription for a client
type ClientSubConfig struct {
	Count   int    `json:"count"`
	Subject string `json:"subject"`
}

// ClientConfig represents a streaming client
type ClientConfig struct {
	Name           string            `json:"name"`
	Instances      int               `json:"instances"`
	PubAsync       bool              `json:"pub_async"`
	PubMsgSize     int               `json:"pub_msgsize"`
	PubRate        string            `json:"pub_delay"`
	PubMsgCount    int               `json:"pub_msgcount"`
	PublishSubject string            `json:"pub_subject"`
	Subscriptions  []ClientSubConfig `json:"subscriptions"`
}

// Config is the server configuration
type Config struct {
	NumConns      int            `json:"numconns"`
	MaxStartDelay int            `json:"client_start_delay_max"`
	ServerURLs    string         `json:"url"`
	UseTLS        bool           `json:"usetls"`
	Clients       []ClientConfig `json:"clients"`
}

// ClientSub is a client subscription
type ClientSub struct {
	subject  string
	sub      stan.Subscription
	ch       chan (bool)
	received int32
	max      int32
	isDone   bool
}

// GetReceivedCount returns the count of received messages
func (cs *ClientSub) GetReceivedCount() int32 {
	return atomic.LoadInt32(&cs.received)
}

// Client represents a NATS streaming client
type Client struct {
	sync.Mutex
	cman              *ClientManager
	clientID          string
	config            *ClientConfig
	hasUniqueSubjects bool
	nc                *nats.Conn
	sc                stan.Conn
	subs              []*ClientSub
	publishCount      int32
	publishDelay      time.Duration
	lastErr           error
	subCh             chan (bool)
	pubCh             chan (bool)
	pubAckCount       int
	ah                stan.AckHandler
	payload           []byte
	done              bool
}

// NewClient returns a new client.
// TODO:  NATS/Stan options per client, if necessary
func NewClient(config *ClientConfig, instance int, cman *ClientManager) *Client {
	c := &Client{}
	c.cman = cman
	c.config = config
	c.clientID = fmt.Sprintf("%s-%d", c.config.Name, instance)
	c.nc = cman.ncPool.GetNextNatsConn()

	if c.isPublisher() {
		c.publishDelay = parsePubRate(c.config.PubRate)
	}
	return c
}

var currentClientID int32

func (c *Client) connect() error {
	var err error
	c.sc, err = stan.Connect("test-cluster", c.clientID, stan.NatsConn(c.nc),
		stan.ConnectWait(DefaultConnectWait), stan.PubAckWait(2*time.Minute),
		stan.MaxPubAcksInflight(4096))
	return err
}

func (c *Client) close() {
	c.closeSubscriptions()
	if err := c.sc.Close(); err != nil {
		log.Printf("error closing stan connection: %v\n", err)
	}
}

func (c *Client) publishUniqueSubjects() bool {
	return c.config.PublishSubject == UniqueSubject
}

func nextUniqueSubject(currentCount int32) string {
	return fmt.Sprintf("%s.%d", UniqueSubject, currentCount)
}

var currentSubjectID int32

func nextGlobalUniqueSubject() string {
	return nextUniqueSubject(atomic.AddInt32(&currentSubjectID, 1))
}

func (c *Client) createClientSubscription(configSub *ClientSubConfig) error {
	csub := &ClientSub{}
	csub.ch = make(chan bool)
	csub.max = int32(configSub.Count)
	csub.subject = configSub.Subject

	// unique callback per sub
	mh := func(msg *stan.Msg) {
		val := atomic.AddInt32(&csub.received, 1)
		if trace {
			log.Printf("%s: Received message %d on %s.\n", c.clientID,
				val, msg.Subject)
		}
		if val == csub.max {
			verbosef("%s: Done receiving messages on subject %s.", c.clientID, msg.Subject)
			csub.ch <- true
		}
	}

	if csub.subject == UniqueSubject {
		csub.subject = nextGlobalUniqueSubject()
	}
	stanSub, err := c.sc.Subscribe(csub.subject, mh)
	if err != nil {
		return err
	}

	csub.sub = stanSub
	c.subs = append(c.subs, csub)

	verbosef("%s: Subscribed to %s.\n", c.clientID, csub.subject)

	return nil
}

func (c *Client) waitForSubscriptions() {
	for _, sub := range c.subs {
		<-sub.ch
	}
	c.cman.subDoneWg.Done()
	verbosef("%s: All messages received.", c.clientID)
}

func (c *Client) isSubscriber() bool {
	return len(c.config.Subscriptions) > 0
}

func (c *Client) isPublisher() bool {
	return c.config.PubMsgCount > 0
}

func (c *Client) createSubscriptions() error {
	for _, s := range c.config.Subscriptions {
		if err := c.createClientSubscription(&s); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) closeSubscriptions() {
	for _, s := range c.subs {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Printf("error closing subscription: %v", err)
		}
	}
}

func (c *Client) publishSubjectMsgs() {
	count := c.config.PubMsgCount
	subject := c.config.PublishSubject

	for i := 0; i < count; i++ {
		c.publishMessage(subject)
	}
}

func (c *Client) publishUniqueMessages() {
	count := c.config.PubMsgCount
	for i := 0; i < count; i++ {
		// use the current unique subject ID.
		for j := int32(1); j <= currentSubjectID; j++ {
			subject := nextUniqueSubject(j)
			c.publishMessage(subject)
		}
	}
}

func parsePubRate(durationStr string) time.Duration {
	if durationStr == "" || durationStr == "0" {
		return 0
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Fatalf("Expected duration to parse, e.g. '100ms'")
	}
	return duration
}

func (c *Client) delayPublish() {
	if c.publishDelay == 0 {
		return
	}
	time.Sleep(c.publishDelay)
}

func (c *Client) publishMessage(subject string) {
	var err error

	c.delayPublish()

	if c.config.PubAsync {
		_, err = c.sc.PublishAsync(subject, c.payload, c.ah)
	} else {
		err = c.sc.Publish(subject, c.payload)
	}
	if err != nil {
		log.Fatalf("Error publishing: %v.\n", err)
	}
	atomic.AddInt32(&c.publishCount, 1)

	if trace {
		log.Printf("%s: Sent message %d to %s.\n", c.clientID,
			atomic.LoadInt32(&c.publishCount), subject)
	}
}

// Publish publishes client messages
func (c *Client) Publish() {
	async := c.config.PubAsync

	verbosef("%s: Started publishing.\n", c.clientID)
	if async {
		c.pubCh = make(chan bool)
		c.ah = func(guid string, err error) {
			if err != nil {
				log.Fatalf("Error publishing: %v.\n", err)
			}
			c.pubAckCount++
			if c.pubAckCount == c.config.PubMsgCount {
				c.pubCh <- true
			}
		}
	}
	c.payload = c.cman.payloadBuffer[:c.config.PubMsgSize]

	if c.publishUniqueSubjects() {
		c.publishUniqueMessages()
	} else {
		c.publishSubjectMsgs()
	}

	// wait for async publishers
	if async {
		<-c.pubCh
	}

	verbosef("%s: Publishing complete.\n", c.clientID)

	c.cman.pubDoneWg.Done()
}

// GetPublishCount returns the current count of published messages
func (c *Client) GetPublishCount() int32 {
	return atomic.LoadInt32(&c.publishCount)
}

// Run connects a client to to the NATS streaming server, starts the subscribers, then
func (c *Client) Run() error {

	delayMax := c.cman.config.MaxStartDelay
	if delayMax > 0 {
		d := time.Duration(rand.Intn(delayMax*1000)) * time.Millisecond
		verbosef("%s:  Delaying start by %v\n", c.clientID, d)
		time.Sleep(d)
	}

	if err := c.connect(); err != nil {
		return err
	}

	verbosef("%s: Connected.", c.clientID)

	if c.isSubscriber() {
		if err := c.createSubscriptions(); err != nil {
			return err
		}

		// in case we are publishing to them.
		c.cman.subStartedWg.Done()
	}

	if c.isPublisher() {
		// wait for all other subscribing clients to start
		c.cman.subStartedWg.Wait()
		c.Publish()
	}

	if c.isSubscriber() {
		c.waitForSubscriptions()
	}
	c.close()

	c.Lock()
	c.done = true
	c.Unlock()

	return nil
}

func usage() {
	log.Fatal("Usage: scale-client-emulator -cfg <config file> [-v VERBOSE]")
}

// GenerateDefaultConfigFile generates a default config file with
// one publisher and one subscriber
func GenerateDefaultConfigFile() ([]byte, error) {
	cfg := Config{}
	cfg.MaxStartDelay = 0
	cfg.NumConns = 1
	cfg.ServerURLs = "nats://localhost:4222"
	cfg.UseTLS = false

	cfg.Clients = make([]ClientConfig, 2)

	cfg.Clients[0].Instances = 1
	cfg.Clients[0].Name = "pub"
	cfg.Clients[0].PubAsync = true
	cfg.Clients[0].PubMsgCount = 100000
	cfg.Clients[0].PubMsgSize = 128
	cfg.Clients[0].PubRate = "0"
	cfg.Clients[0].PublishSubject = "foo"

	cfg.Clients[1].Instances = 1
	cfg.Clients[1].Name = "sub"
	cfg.Clients[1].Subscriptions = make([]ClientSubConfig, 1)
	cfg.Clients[1].Subscriptions[0].Count = 100000
	cfg.Clients[1].Subscriptions[0].Subject = "foo"

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal json: %v\n", err)
	}

	err = ioutil.WriteFile(DefaultConfigFileName, raw, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not write default config file: %v\n", err)
	}

	log.Printf("Generated default configuration file %s.\n", DefaultConfigFileName)
	return raw, nil
}

// LoadConfiguration loads a server configuration.
func LoadConfiguration(filename string) (*Config, error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		if filename == DefaultConfigFileName {
			raw, err = GenerateDefaultConfigFile()
		}
		if err != nil {
			return nil, err
		}
	}

	serverConfigs, err := getConfig(string(raw))
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}

	return serverConfigs, err
}

func getConfig(jsonString string) (*Config, error) {
	var config = &Config{}

	err := json.Unmarshal([]byte(jsonString), config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %v\n", err)
	}

	return config, nil
}

// ClientManager tracks all clients
type ClientManager struct {
	sync.Mutex
	clientsMap    map[string]*Client
	ncPool        *NatsServerConnPool
	config        *Config
	pubCount      int
	subCount      int
	subStartedWg  sync.WaitGroup
	pubDoneWg     sync.WaitGroup
	subDoneWg     sync.WaitGroup
	payloadBuffer []byte
}

func printClient(c *Client) {
	verbosef("%s: Created.  async=%v,pubsubj=%s,pubcount=%d,msgsize=%d,sub=%s,subcount=%d",
		c.clientID, c.config.PubAsync, c.config.PublishSubject,
		c.config.PubMsgCount,
		c.config.PubMsgSize, "",
		len(c.config.Subscriptions))
}

// NewClientManager creates a client manager
func NewClientManager(ncPool *NatsServerConnPool, cfg *Config) *ClientManager {
	var maxMsgSize int

	cc := &ClientManager{}
	cc.ncPool = ncPool
	cc.config = cfg
	cc.clientsMap = make(map[string]*Client)

	for i := 0; i < len(cfg.Clients); i++ {
		for j := 0; j < cfg.Clients[i].Instances; j++ {
			cli := NewClient(&cfg.Clients[i], j, cc)
			if cli.isPublisher() {
				cc.pubCount++
				if cfg.Clients[i].PubMsgSize > maxMsgSize {
					maxMsgSize = cfg.Clients[i].PubMsgSize
				}
			}

			if cli.isSubscriber() {
				cc.subCount++
			}

			cc.clientsMap[cli.clientID] = cli
			printClient(cli)
		}
	}

	cc.payloadBuffer = make([]byte, maxMsgSize)

	log.Printf("Created %d clients:  %d publishing and %d subscribing.\n",
		len(cc.clientsMap), cc.pubCount, cc.subCount)

	return cc
}

// RunClients runs all the configured clients
func (cc *ClientManager) RunClients() {
	cc.subStartedWg.Add(cc.subCount)
	cc.pubDoneWg.Add(cc.pubCount)
	cc.subDoneWg.Add(cc.subCount)

	for _, c := range cc.clientsMap {
		go c.Run()
	}
}

// WaitForCompletion waits until all clients have been completed.
func (cc *ClientManager) WaitForCompletion() {
	cc.subStartedWg.Wait()
	log.Printf("All subscribing clients ready.")
	cc.pubDoneWg.Wait()
	log.Printf("All publishers have completed.")
	cc.subDoneWg.Wait()
	log.Printf("All subscribers have completed.")
}

// Prints the status of the current test
func (cc *ClientManager) PrintActiveClientStatus(ivl int) {
	for {
		time.Sleep(time.Duration(ivl) * time.Second)
		cc.PrintReport(true)
	}
}

// PrintReport runs a report of current active clients
func (cc *ClientManager) PrintReport(activeOnly bool) {
	var line string
	var count int

	cc.Lock()
	defer cc.Unlock()

	if activeOnly {
		log.Printf("*** Active Clients ***")
	} else {
		log.Printf("*** All Clients ***")
	}

	for _, c := range cc.clientsMap {
		c.Lock()
		done := c.done
		c.Unlock()

		if done && activeOnly {
			continue
		}
		line = fmt.Sprintf("%v: Client %s,", time.Now().Format("2006-01-02 15:04:05"), c.clientID)
		if c.isPublisher() {
			line += fmt.Sprintf(" pub: %s=(%d/%d)", c.config.PublishSubject,
				c.GetPublishCount(), c.config.PubMsgCount)
		}

		if c.isSubscriber() {
			line += " subs:"
			for _, csub := range c.subs {
				line += fmt.Sprintf(" %s=(%d/%d)", csub.subject,
					csub.GetReceivedCount(), csub.max)
			}
		}
		log.Printf("%s\n", line)
		count++
	}

	if activeOnly {
		log.Printf("%d active clients.\n", count)
	}
	log.Printf("\n")
}

func disconnectedHandler(nc *nats.Conn) {
	if nc.LastError() != nil {
		log.Fatalf("connection %q has been disconnected: %v\n",
			nc.Opts.Name, nc.LastError())
	}
}

func reconnectedHandler(nc *nats.Conn) {
	log.Fatalf("connection %q reconnected to NATS Server at %q\n",
		nc.Opts.Name, nc.ConnectedUrl())
}

func closedHandler(nc *nats.Conn) {
	log.Fatalf("connection %q has been closed\n", nc.Opts.Name)
}

func errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	log.Fatalf("asynchronous error on connection %s, subject %s: %s\n",
		nc.Opts.Name, sub.Subject, err)
}

func run(configFile string, vbs bool, tbs bool, prIvl int) {
	verbose = vbs
	if tbs {
		verbose = true
		trace = true
	}

	cfg, err := LoadConfiguration(configFile)
	if err != nil {
		log.Fatalf("error loading configuration file:  %v\n", err)
	}

	connPool, err := NewNatsServerConnPool(cfg)
	if err != nil {
		log.Fatalf("connection error:  %v\n", err)
	}

	cman := NewClientManager(connPool, cfg)
	cman.RunClients()
	if prIvl > 0 {
		go cman.PrintActiveClientStatus(prIvl)
	}
	cman.WaitForCompletion()
	cman.PrintReport(false)
	log.Println("Exiting.")
}

func main() {
	var configFile = flag.String("config", DefaultConfigFileName, "configuration file to use.  Default is generated.")
	var vb = flag.Bool("V", false, "Verbose")
	var tb = flag.Bool("DV", false, "Verbose/Trace")
	var pr = flag.Int("report", 0, "Print an active client report every X seconds")

	log.SetFlags(0)
	flag.Parse()

	run(*configFile, *vb, *tb, *pr)
}
