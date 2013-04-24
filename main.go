package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	// Command-line flags
	out   = flag.String("out", "127.0.0.1:8125", "UDP destination address")
	in    = flag.String("in", "127.0.0.1:8125", "TCP listening address")
	token = flag.String("token", "", "Expected \"X-Token\" header for all requests")
	cert  = flag.String("cert", "cert.pem", "Path to SSL certificate file")
	key   = flag.String("key", "key.pem", "Path to SSL key file")
	help  = flag.Bool("help", false, "Print usage info")

	// Stdout logging
	logger *log.Logger

	// Statsd connection
	writeLock sync.Mutex
	destConn  net.Conn

	// Stats
	requestsReceived int
	bytesReceived    int
)

// Print usage information and exit.
func usage() {
	fmt.Printf("Usage: %s [flags]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}

// Log stats to the logger and reset them.
func logStats() {
	logger.Printf("Received %d bytes, %d requests\n", bytesReceived, requestsReceived)
	requestsReceived = 0
	bytesReceived = 0
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Token") != *token {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty payload", http.StatusBadRequest)
		return
	}

	// Update stats
	bytesReceived += len(body)
	requestsReceived += 1

	writeLock.Lock()
	_, err = destConn.Write(body)
	writeLock.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	var err error

	// Load command-line options
	flag.Parse()
	flag.Usage = usage
	if *help {
		usage()
	}

	// Setup logger
	loggerPrefix := fmt.Sprintf("%s[%d] ", os.Args[0], os.Getpid())
	logger = log.New(os.Stdout, loggerPrefix, log.LstdFlags)

	// Set up UDP connection
	destConn, err = net.DialTimeout("udp", *out, time.Second)
	if err != nil {
		panic(err)
	}

	// Record stats every minute
	go func() {
		statsTicker := time.NewTicker(time.Minute)
		for {
			select {
			case <-statsTicker.C:
				logStats()
			}
		}
	}()

	// Listen for requests
	http.HandleFunc("/", handler)
	err = http.ListenAndServeTLS(*in, *cert, *key, nil)
	if err != nil {
		panic(err)
	}
}
