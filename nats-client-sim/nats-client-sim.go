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

	"github.com/nats-io/nats"
)

// TODO:  Create interfaces and reuse code w/ stan-client-sim

// Some sane defaults
const (
	DefaultConnectWait    = 4 * time.Minute
	DefaultConfigFileName = "config.json"
	UniqueSubject         = "UNIQUE"
)

var trace bool
var verbose bool

func verbosef(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
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
	UserName       string            `json:"username"`
	Password       string            `json:"password"`
	PubMsgSize     int               `json:"pub_msgsize"`
	PubRate        string            `json:"pub_delay"`
	PubMsgCount    int               `json:"pub_msgcount"`
	PublishSubject string            `json:"pub_subject"`
	Subscriptions  []ClientSubConfig `json:"subscriptions"`
}

// Config is the server configuration
type Config struct {
	MaxStartDelay int            `json:"client_start_delay_max"`
	ServerURLs    string         `json:"url"`
	TLSClientCA   string         `json:"tlsca"`
	TLSClientCert string         `json:"tlscert"`
	TLSClientKey  string         `json:"tlskey"`
	UseTLS        bool           `json:"usetls"`
	Clients       []ClientConfig `json:"clients"`
}

// ClientSub is a client subscription
type ClientSub struct {
	subject  string
	sub      *nats.Subscription
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
	cm                *ClientManager
	config            *ClientConfig
	clientID          string
	hasUniqueSubjects bool
	nc                *nats.Conn
	subs              []*ClientSub
	publishCount      int32
	publishDelay      time.Duration
	subCh             chan (bool)
	payload           []byte
	done              bool
}

// NewClient returns a new client.
func NewClient(config *ClientConfig, instance int, cm *ClientManager) *Client {
	c := &Client{}
	c.cm = cm
	c.config = config
	c.clientID = fmt.Sprintf("%s-%d", c.config.Name, instance)

	if c.isPublisher() {
		c.publishDelay = parsePubRate(c.config.PubRate)
	}
	return c
}

func (c *Client) connect() error {
	var err error
	opts := nats.DefaultOptions
	opts.Servers = strings.Split(c.cm.config.ServerURLs, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}

	opts.Secure = c.cm.config.UseTLS
	opts.AsyncErrorCB = errorHandler
	opts.DisconnectedCB = disconnectedHandler
	opts.ReconnectedCB = reconnectedHandler
	opts.ClosedCB = closedHandler
	opts.User = c.config.UserName
	opts.Password = c.config.Password
	opts.Name = c.clientID
	c.nc, err = opts.Connect()
	return err
}

func (c *Client) close() {
	c.closeSubscriptions()
	c.nc.Close()
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

func (c *Client) createClientSubscription(configSub *ClientSubConfig) {
	csub := &ClientSub{}
	csub.ch = make(chan bool)
	csub.max = int32(configSub.Count)
	csub.subject = configSub.Subject

	// unique callback per sub
	mh := func(msg *nats.Msg) {
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
	natsSub, err := c.nc.Subscribe(csub.subject, mh)
	if err != nil {
		log.Fatalf("Error creating subscription for %s: %v", csub.subject, err)
	}
	c.nc.Flush()

	csub.sub = natsSub
	c.subs = append(c.subs, csub)

	verbosef("%s: Subscribed to %s.\n", c.clientID, csub.subject)
}

func (c *Client) waitForSubscriptions() {
	for _, sub := range c.subs {
		<-sub.ch
	}
	c.cm.subDoneWg.Done()
	verbosef("%s: All messages received.", c.clientID)
}

func (c *Client) isSubscriber() bool {
	return len(c.config.Subscriptions) > 0
}

func (c *Client) isPublisher() bool {
	return c.config.PubMsgCount > 0
}

func (c *Client) createSubscriptions() {
	for _, s := range c.config.Subscriptions {
		c.createClientSubscription(&s)
	}
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

	atomic.AddInt32(&c.publishCount, 1)
	err = c.nc.Publish(subject, c.payload)
	if err != nil {
		log.Fatalf("%s: Error publishing: %v.\n", c.clientID, err)
	}
	if trace {
		log.Printf("%s: Success sending msg # %d to %s.\n", c.clientID,
			atomic.LoadInt32(&c.publishCount), subject)
	}
}

// Publish publishes client messages
func (c *Client) Publish() {
	verbosef("%s: Started publishing.\n", c.clientID)

	c.payload = c.cm.payloadBuffer[:c.config.PubMsgSize]

	if c.publishUniqueSubjects() {
		c.publishUniqueMessages()
	} else {
		c.publishSubjectMsgs()
	}
	if err := c.nc.Flush(); err != nil {
		log.Fatalf("%s: error flushing: %v", c.clientID, err)
	}

	verbosef("%s: Publishing complete.\n", c.clientID)

	c.cm.pubDoneWg.Done()
}

// GetPublishCount returns the current count of published messages
func (c *Client) GetPublishCount() int32 {
	return atomic.LoadInt32(&c.publishCount)
}

// Run connects a client to to the NATS streaming server, starts the subscribers, then
func (c *Client) Run() error {

	delayMax := c.cm.config.MaxStartDelay
	if delayMax > 0 {
		d := time.Duration(rand.Intn(delayMax*1000)) * time.Millisecond
		verbosef("%s:  Delaying start by %v\n", c.clientID, d)
		time.Sleep(d)
	}

	if err := c.connect(); err != nil {
		log.Fatalf("%s:  unable to connect: %v", c.clientID, err)
	}

	verbosef("%s: Connected.", c.clientID)

	if c.isSubscriber() {
		c.createSubscriptions()
		c.cm.subStartedWg.Done()
	}

	if c.isPublisher() {
		// wait for all other subscribing clients to start
		c.cm.subStartedWg.Wait()
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
	cfg.ServerURLs = "nats://localhost:4222"
	cfg.UseTLS = false

	cfg.Clients = make([]ClientConfig, 2)

	cfg.Clients[0].Instances = 1
	cfg.Clients[0].Name = "pub"
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
	config        *Config
	pubCount      int
	subCount      int
	subStartedWg  sync.WaitGroup
	pubDoneWg     sync.WaitGroup
	subDoneWg     sync.WaitGroup
	payloadBuffer []byte
	perfStartTime time.Time
	printInterval int
	longReport    bool
}

func printClient(c *Client) {
	verbosef("%s: Created.  pubsubj=%s,pubcount=%d,msgsize=%d,sub=%s,subcount=%d",
		c.clientID, c.config.PublishSubject,
		c.config.PubMsgCount,
		c.config.PubMsgSize, "",
		len(c.config.Subscriptions))
}

// NewClientManager creates a client manager
func NewClientManager(cfg *Config, prIvl int, longReport bool) *ClientManager {
	var maxMsgSize int

	opts := nats.DefaultOptions
	opts.Servers = strings.Split(cfg.ServerURLs, ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}

	opts.Secure = true
	opts.AsyncErrorCB = errorHandler
	opts.DisconnectedCB = disconnectedHandler
	opts.ReconnectedCB = reconnectedHandler
	opts.ClosedCB = closedHandler
	if cfg.UseTLS {
		verbosef("Using TLS.\n")
		if err := nats.Secure()(&opts); err != nil {
			log.Fatalf("error enabliing tls: %v\n", err)
		}
	}
	if cfg.TLSClientCA != "" {
		verbosef("Using client CA %s\n", cfg.TLSClientCA)
		if err := nats.RootCAs(cfg.TLSClientCert)(&opts); err != nil {
			log.Fatalf("client CA error: %v\n", err)
		}
	}
	if cfg.TLSClientCert != "" {
		verbosef("Using client cert: %s\n", cfg.TLSClientCert)
		verbosef("Using client key: %s\n", cfg.TLSClientKey)
		if err := nats.ClientCert(cfg.TLSClientCert, cfg.TLSClientKey)(&opts); err != nil {
			log.Fatalf("client cert error: %v\n", err)
		}
		opts.TLSConfig.InsecureSkipVerify = true
	}

	cm := &ClientManager{}
	cm.config = cfg
	cm.printInterval = prIvl
	cm.longReport = longReport
	cm.clientsMap = make(map[string]*Client)

	for i := 0; i < len(cfg.Clients); i++ {
		for j := 0; j < cfg.Clients[i].Instances; j++ {
			cli := NewClient(&cfg.Clients[i], j, cm)
			if cli.isPublisher() {
				cm.pubCount++
				if cfg.Clients[i].PubMsgSize > maxMsgSize {
					maxMsgSize = cfg.Clients[i].PubMsgSize
				}
			}

			if cli.isSubscriber() {
				cm.subCount++
			}

			cm.clientsMap[cli.clientID] = cli
			printClient(cli)
		}
	}

	cm.payloadBuffer = make([]byte, maxMsgSize)

	log.Printf("Created %d clients:  %d publishing and %d subscribing.\n",
		len(cm.clientsMap), cm.pubCount, cm.subCount)

	return cm
}

// RunClients runs all the configured clients
func (cm *ClientManager) RunClients() {
	cm.subStartedWg.Add(cm.subCount)
	cm.pubDoneWg.Add(cm.pubCount)
	cm.subDoneWg.Add(cm.subCount)

	for _, c := range cm.clientsMap {
		go c.Run()
	}
	log.Printf("Started all clients.")
}

// WaitForCompletion waits until all clients have been completed.
func (cm *ClientManager) WaitForCompletion() {
	log.Printf("Waiting for clients to connect and subscribe.")
	cm.subStartedWg.Wait()
	log.Printf("All subscribing clients ready.")

	// subscribers are ready and publishing will commence,
	// so start measuring throughput
	cm.perfStartTime = time.Now()

	cm.StartActiveClientReporting()

	log.Printf("Waiting for publishers to complete.")
	cm.pubDoneWg.Wait()
	log.Printf("All publishers have completed.  Waiting for subscribers.")
	cm.subDoneWg.Wait()
	log.Printf("All subscribers have completed.")
}

// StartActiveClientReporting the status of the current test
func (cm *ClientManager) StartActiveClientReporting() {
	if cm.printInterval > 0 {
		go func() {
			for {
				time.Sleep(time.Duration(cm.printInterval) * time.Second)
				cm.PrintReport(true)
			}
		}()
	}
}

func (cm *ClientManager) printAggregateMsgRate(msgsSent, msgsRecv int) {
	d := time.Now().Sub(cm.perfStartTime)
	msRate := float64(msgsSent) / d.Seconds()
	mrRate := float64(msgsRecv) / d.Seconds()

	log.Printf("Sent aggregate %d msgs at %d msgs/sec.\n", msgsSent, int(msRate))
	log.Printf("Received aggregate %d msgs at %d msgs/sec.\n", msgsRecv, int(mrRate))
}

// PrintReport runs a report of current active clients
func (cm *ClientManager) PrintReport(activeOnly bool) {
	var line string
	var count int
	var tsent int32
	var trecv int32

	cm.Lock()
	defer cm.Unlock()

	if activeOnly {
		log.Printf("*** Active Clients ***")
	} else {
		log.Printf("*** All Clients ***")
	}

	for _, c := range cm.clientsMap {
		c.Lock()
		done := c.done
		c.Unlock()

		line = fmt.Sprintf("%v: Client %s,", time.Now().Format("2016-04-08 15:04:05"), c.clientID)
		if c.isPublisher() {
			tsent += c.GetPublishCount()
			line += fmt.Sprintf(" pub: %s=(%d/%d)", c.config.PublishSubject,
				c.GetPublishCount(), c.config.PubMsgCount)
		}

		if c.isSubscriber() {
			line += " subs:"
			for _, csub := range c.subs {
				trecv += csub.GetReceivedCount()
				line += fmt.Sprintf(" %s=(%d/%d)", csub.subject,
					csub.GetReceivedCount(), csub.max)
			}
		}

		if cm.longReport {
			if !activeOnly || (activeOnly && !done) {
				log.Printf("%s\n", line)
			}
		}
		count++
	}

	log.Printf("%d of %d clients listed.", count, len(cm.clientsMap))
	cm.printAggregateMsgRate(int(tsent), int(trecv))
}

func disconnectedHandler(nc *nats.Conn) {
	if nc.LastError() != nil {
		log.Printf("connection %q has been disconnected: %v\n",
			nc.Opts.Name, nc.LastError())
	}
}

func reconnectedHandler(nc *nats.Conn) {
	log.Printf("connection %q reconnected to NATS Server at %q\n",
		nc.Opts.Name, nc.ConnectedUrl())
}

func closedHandler(nc *nats.Conn) {
	verbosef("connection %q has been closed\n", nc.Opts.Name)
}

func errorHandler(nc *nats.Conn, sub *nats.Subscription, err error) {
	log.Fatalf("asynchronous error on connection %s, subject %s: %s\n",
		nc.Opts.Name, sub.Subject, err)
}

func run(configFile string, isVerbose, isTraceVerbose, longReport bool, prIvl int) {
	verbose = isVerbose
	if isTraceVerbose {
		verbose = true
		trace = true
	}

	cfg, err := LoadConfiguration(configFile)
	if err != nil {
		log.Fatalf("error loading configuration file:  %v\n", err)
	}

	cman := NewClientManager(cfg, prIvl, longReport)
	cman.RunClients()
	cman.WaitForCompletion()
	cman.PrintReport(false)
	log.Println("Test completed.")
}

func main() {
	var configFile = flag.String("config", DefaultConfigFileName, "configuration file to use.  Default is generated.")
	var vb = flag.Bool("V", false, "Verbose")
	var tb = flag.Bool("DV", false, "Verbose/Trace")
	var pr = flag.Int("report", 0, "Print an active client report every X seconds")
	var lf = flag.Bool("lr", false, "Print a long form of report data.")

	log.SetFlags(0)
	flag.Parse()

	run(*configFile, *vb, *tb, *lf, *pr)
}
