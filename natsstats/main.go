package main

import (
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
)

var (
	verbose   bool
	varzFp    *os.File
	routezFp  *os.File
	connzFp   *os.File
	subszFp   *os.File
	url       string
	getVarz   bool
	getConnz  bool
	getSubsz  bool
	getRoutez bool
	outputDir string
	ivl       time.Duration
)

func verbosef(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

func getStatz(monURL string, fp *os.File) error {

	// make life easy for the caller
	if fp == nil {
		verbosef("skipping %v\n", monURL)
		return nil
	}

	HTTPClient := &http.Client{}

	verbosef("Getting data from base url: %s\n", monURL)
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
	if _, err = fp.Write(body); err != nil {
		return err
	}
	if _, err = fp.WriteString("\n"); err != nil {
		return err
	}
	return err
}

func pollAndSaveStats() error {
	if err := getStatz(url+"/varz", varzFp); err != nil {
		return fmt.Errorf("couldn't process varz: %v", err)
	}
	if err := getStatz(url+"/connz", connzFp); err != nil {
		return fmt.Errorf("couldn't process varz: %v", err)
	}
	if err := getStatz(url+"/subsz", subszFp); err != nil {
		return fmt.Errorf("%v", err)
	}
	if err := getStatz(url+"/routez", routezFp); err != nil {
		return fmt.Errorf("%v", err)
	}
	return nil
}

func createFile(fname string) *os.File {
	fp, err := os.OpenFile(outputDir+"/"+fname, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("couldn't open file %s: %v", fname, err)
	}
	return fp
}

func createFiles() {
	if getVarz {
		varzFp = createFile("varz.json")
	}
	if getConnz {
		connzFp = createFile("connz.json")
	}
	if getSubsz {
		subszFp = createFile("subsz.json")
	}
	if getRoutez {
		routezFp = createFile("routez.json")
	}
}

func closeFile(fp *os.File) {
	if fp != nil {
		fp.Close()
	}
}
func closeFiles() {
	closeFile(varzFp)
	closeFile(connzFp)
	closeFile(subszFp)
	closeFile(routezFp)
}

func banner() {
	log.Printf("Monitor Url: %v\n", url)
	log.Printf("Get varz:    %v\n", getVarz)
	log.Printf("Get subsz:   %v\n", getSubsz)
	log.Printf("Get connz:   %v\n", getConnz)
	log.Printf("Get routez:  %v\n", getRoutez)
	log.Printf("Poll ivl:    %v\n", ivl)
	log.Printf("Output Path: %v\n", outputDir)
	log.Printf("============================\n")
}

func main() {
	flag.StringVar(&url, "url", "http://localhost:8222", "Monitor url")
	flag.BoolVar(&verbose, "V", false, "Verbose")
	flag.BoolVar(&getVarz, "varz", false, "Get varz")
	flag.BoolVar(&getConnz, "connz", false, "Get connz")
	flag.BoolVar(&getSubsz, "subsz", false, "Get subsz")
	flag.BoolVar(&getRoutez, "routez", false, "Get routez")
	flag.StringVar(&outputDir, "outdir", ".", "Output Directory")

	var ivlStr = flag.String("ivl", "1s", "Poll interval.  Default is 1s")

	log.SetFlags(0)
	flag.Parse()

	if !getVarz && !getConnz && !getSubsz && !getRoutez {
		log.Fatalf("must select at least one of -varz, -connz, -subsz, or -routez")
	}
	ivl, err := time.ParseDuration(*ivlStr)
	if err != nil {
		log.Fatalf("couldn't parse duration: %v", err)
	}

	createFiles()
	banner()

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
