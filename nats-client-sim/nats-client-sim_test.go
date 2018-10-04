package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nats-io/gnatsd/logger"
	"github.com/nats-io/gnatsd/server"
)

const (
	// ClientPort is the default port for clients to connect
	ClientPort = 4222

	// MonitorPort is the default monitor port
	MonitorPort = 8222
)

// RunServer runs the NATS server in a go routine
func RunServer() *server.Server {
	return RunServerWithPorts(ClientPort, MonitorPort)
}

// RunServerWithPorts runs the NATS server with a monitor port in a go routine
func RunServerWithPorts(cport, mport int) *server.Server {
	var enableLogging bool

	// To enable debug/trace output in the NATS server,
	// flip the enableLogging flag.
	// enableLogging = true

	opts := &server.Options{
		Host:     "localhost",
		Port:     cport,
		HTTPHost: "127.0.0.1",
		HTTPPort: mport,
		NoLog:    !enableLogging,
		NoSigs:   true,
	}

	s := server.New(opts)
	if s == nil {
		panic("No NATS Server object returned.")
	}

	if enableLogging {
		l := logger.NewStdLogger(true, true, true, false, true)
		s.SetLogger(l, true, true)
	}

	// Run server in Go routine.
	go s.Start()

	end := time.Now().Add(10 * time.Second)
	for time.Now().Before(end) {
		netAddr := s.Addr()
		if netAddr == nil {
			continue
		}
		addr := s.Addr().String()
		if addr == "" {
			time.Sleep(10 * time.Millisecond)
			// Retry. We might take a little while to open a connection.
			continue
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			// Retry after 50ms
			time.Sleep(50 * time.Millisecond)
			continue
		}
		_ = conn.Close() // nolint

		// Wait a bit to give a chance to the server to remove this
		// "client" from its state, which may otherwise interfere with
		// some tests.
		time.Sleep(25 * time.Millisecond)

		return s
	}
	panic("Unable to start NATS Server in Go Routine")
}

func getSummary(t *testing.T, fname string) *JSONOutput {
	var ar JSONOutput

	fp, err := os.Open(fname)
	if err != nil {
		t.Fatalf("Unable to open result file %s:  %v\n", fname, err)
	}
	raw, _ := ioutil.ReadAll(fp)

	if err := json.Unmarshal(raw, &ar); err != nil {
		t.Fatalf("Unable to unmarshal result file %s:  %v\n", fname, err)
	}

	return &ar
}

func checkBasicResults(t *testing.T, sr *SummaryRecord, numClients, numPubs, numSubs, minMsgsSent, minMsgsRecv, errors, avgConnAttempts int) {
	if sr.NumClients != numClients {
		t.Fatalf("unexpected number of clients: %d vs %d", sr.NumClients, numClients)
	}
	if sr.NumPublishers != numPubs {
		t.Fatalf("unexpected number of pubs: %d vs %d", sr.NumPublishers, numPubs)
	}
	if sr.NumSubscribers != numSubs {
		t.Fatalf("unexpected number of subs: %d vs %d", sr.NumSubscribers, numSubs)
	}
	if sr.TotalMessagesRecv < minMsgsRecv {
		t.Fatalf("unexpected number of msgs recv: %d vs %d", sr.TotalMessagesRecv, minMsgsRecv)
	}
	if sr.TotalMessagesSent < minMsgsSent {
		t.Fatalf("unexpected number of msgs sent: %d vs %d", sr.TotalMessagesSent, minMsgsSent)
	}
	if sr.TotalAsErrors != 0 {
		t.Fatalf("unexpected number of async errors: %d", sr.TotalAsErrors)
	}
	if sr.TotalErrors != errors {
		t.Fatalf("unexpected number of errors: %d", sr.TotalErrors)
	}
	if sr.TotalErrors == 0 {
		if sr.TotalDisconnects != sr.NumClients {
			t.Fatalf("unexpected number of disconnects: %d vs %d", sr.TotalDisconnects, sr.NumClients)
		}
		if sr.TotalConnects != sr.NumClients {
			t.Fatalf("unexpected number of connects: %d vs %d", sr.TotalConnects, sr.NumClients)
		}
	}
	if sr.AvgConnAttempts != avgConnAttempts {
		t.Fatalf("unexpected number of average connects: %d vs %d", sr.AvgConnAttempts, avgConnAttempts)
	}
}

func Test_run_simple(t *testing.T) {
	s := RunServer()
	defer s.Shutdown()
	run("configs/simple.json", false, false, false)
	results := getSummary(t, "./simple_results.json")
	checkBasicResults(t, results.Summary, 1, 1, 1, 3000, 3000, 0, 1)
	os.Remove("./simple_results.json")
}

func Test_run_streams(t *testing.T) {
	s := RunServer()
	defer s.Shutdown()
	run("configs/4streams.json", false, false, false)
	results := getSummary(t, "./4streams_results.json")
	checkBasicResults(t, results.Summary, 8, 4, 4, 4000, 4000, 0, 1)
	os.Remove("./4streams_results.json")
}

func Test_run_fanout_1to100(t *testing.T) {
	s := RunServer()
	defer s.Shutdown()
	run("configs/fanout_1to100.json", false, false, false)
	results := getSummary(t, "./fanout_1to100_results.json")
	checkBasicResults(t, results.Summary, 101, 1, 100, 1000, 100000, 0, 1)
	os.Remove("fanout_1to100_results.json")
}

func Test_run_NoServer(t *testing.T) {
	run("configs/simple.json", false, false, false)
	results := getSummary(t, "./simple_results.json")
	checkBasicResults(t, results.Summary, 1, 1, 1, 0, 0, 1, 2)
	os.Remove("./simple_results.json")
}
