package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	gnatsd "github.com/nats-io/gnatsd/server"
)

// ServerStat holds measurements of various data
type ServerStat struct {
	StatTime time.Time      `json:"time"`
	StatIdx  int64          `json:"index"`
	Hostname string         `json:"hostname"`
	Varz     *gnatsd.Varz   `json:"varz,omitempty"`
	Connz    *gnatsd.Connz  `json:"connz,omitempty"`
	Subsz    *gnatsd.Subsz  `json:"subsz,omitempty"`
	Routesz  *gnatsd.Routez `json:"routez,omitempty"`
}

// Options are server options.
type Options struct {
	URL         string `json:"url"`
	GetVarz     bool   `json:"getvarz"`
	GetConnz    bool   `json:"getconnz"`
	GetSubsz    bool   `json:"getsubsz"`
	GetRoutez   bool   `json:"getroutez"`
	Interval    string `json:"interval"`
	OutputPath  string `json:"outfile"`
	PrettyPrint bool   `json:"prettyprint"`
	Verbose     bool   `json:"verbose"`
}

// GetDefaultOptions returns a default options set.
func GetDefaultOptions() *Options {
	return &Options{
		Verbose:     false,
		URL:         "http://localhost:8222",
		GetVarz:     true,
		GetConnz:    false,
		GetRoutez:   false,
		GetSubsz:    false,
		Interval:    "1s",
		OutputPath:  "results.json",
		PrettyPrint: false,
	}
}

var (
	outputFp *os.File
	opts     *Options
	stat     *ServerStat
)

func verbosef(format string, v ...interface{}) {
	if opts.Verbose {
		log.Printf(format, v...)
	}
}

func getStatz(monURL string, statz interface{}) error {

	HTTPClient := &http.Client{}

	verbosef("Polling data from url: %s\n", monURL)
	resp, err := HTTPClient.Get(monURL)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("could not get stats from server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %v", err)
	}
	err = json.Unmarshal(body, &statz)
	if err != nil {
		return fmt.Errorf("could not unmarshal json: %v", err)
	}
	return nil
}

func marshalStat(s *ServerStat) ([]byte, error) {
	var (
		raw []byte
		err error
	)

	if opts.PrettyPrint {
		raw, err = json.MarshalIndent(stat, "", "    ")
	} else {
		if raw, err = json.Marshal(stat); err != nil {
			return nil, err
		}
		raw = append(raw, '\n')
	}
	return raw, err
}

func writeStat(varz *gnatsd.Varz, connz *gnatsd.Connz,
	subsz *gnatsd.Subsz, routez *gnatsd.Routez) error {

	stat.StatTime = time.Now()
	stat.StatIdx++
	stat.Varz = varz
	stat.Connz = connz
	stat.Subsz = subsz
	stat.Routesz = routez

	raw, err := marshalStat(stat)
	if err != nil {
		log.Fatalf("Unable to marshall output data: %v\n", err)
	}

	_, err = outputFp.Write(raw)

	return err
}

// This function polls the urls
func pollAndSaveStats() error {
	var (
		varz   *gnatsd.Varz
		connz  *gnatsd.Connz
		routez *gnatsd.Routez
		subsz  *gnatsd.Subsz
	)

	if opts.GetVarz {
		varz = &gnatsd.Varz{}
		if err := getStatz(opts.URL+"/varz", varz); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	if opts.GetConnz {
		connz = &gnatsd.Connz{}
		if err := getStatz(opts.URL+"/connz", connz); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	if opts.GetSubsz {
		subsz = &gnatsd.Subsz{}
		if err := getStatz(opts.URL+"/subsz", subsz); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	if opts.GetRoutez {
		routez = &gnatsd.Routez{}
		if err := getStatz(opts.URL+"/routez", routez); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	return writeStat(varz, connz, subsz, routez)
}

func createFile(fname string) *os.File {
	fp, err := os.OpenFile(fname, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("couldn't create file %s: %v", fname, err)
	}
	return fp
}

func closeFiles() {
	if outputFp != nil {
		if err := outputFp.Close(); err != nil {
			log.Printf("couldn't close output file: %v", err)
		}
	}
}

func banner() {
	log.Printf("Monitor Url: %v\n", opts.URL)
	log.Printf("Poll ivl:    %v\n", opts.Interval)
	log.Printf("Get varz:    %v\n", opts.GetVarz)
	log.Printf("Get subsz:   %v\n", opts.GetSubsz)
	log.Printf("Get connz:   %v\n", opts.GetConnz)
	log.Printf("Get routez:  %v\n", opts.GetRoutez)
	log.Printf("Output File: %v\n", opts.OutputPath)
	log.Printf("PrettyPrint: %v\n", opts.PrettyPrint)
	log.Printf("============================\n")
}

// GenerateDefaultConfigFile generates a default config file
func generateDefaultConfigFile(fname string) error {
	o := GetDefaultOptions()
	o.GetVarz = true
	raw, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal json: %v", err)
	}

	err = ioutil.WriteFile(fname, raw, 0644)
	if err != nil {
		return fmt.Errorf("could not write config file: %v", err)
	}

	return nil
}

func loadConfig(fname string) (*Options, error) {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		return nil, err
	}

	raw, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	o := GetDefaultOptions()
	err = json.Unmarshal([]byte(raw), o)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %v", err)
	}

	// check options
	if !o.GetVarz && !o.GetConnz && !o.GetSubsz && !o.GetRoutez {
		log.Fatalf("must configure at least one of varz, connz, subsz, or routez")
	}

	return o, err
}

func usage() {
	log.Printf("%v [-V] [-create] <configfile>\n", os.Args[0])
	os.Exit(1)
}

func main() {
	var (
		verbose    bool
		createfile bool
		configFile string
		err        error
	)

	flag.BoolVar(&verbose, "V", false, "Verbose")
	flag.BoolVar(&createfile, "create", false, "Create the configuration file")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	configFile = args[0]
	if createfile {
		if err := generateDefaultConfigFile(configFile); err != nil {
			log.Fatalf("Couldn't generate configuration file: %v", err)
		}
		os.Exit(0)
	}

	if opts, err = loadConfig(configFile); err != nil {
		log.Fatalf("Unable to load config file: %v", err)
	}

	ivl, err := time.ParseDuration(opts.Interval)
	if err != nil {
		log.Fatalf("invalid interval: %v", err)
	}

	banner()

	outputFp = createFile(opts.OutputPath)
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("WARNING: couldn't get hostname.")
		hostname = "Unknown"
	}

	stat = &ServerStat{
		Hostname: hostname,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	quit := make(chan bool, 1)
	go func() {
		for {
			if err := pollAndSaveStats(); err != nil {
				log.Fatalf("%v", err)
			}
			select {
			case <-quit:
				closeFiles()
				wg.Done()
				return
			default:
				time.Sleep(ivl)
			}
		}
	}()

	// Setup the interrupt handler to gracefully exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		quit <- true
		wg.Wait()
		os.Exit(0)
	}()
	runtime.Goexit()
}
