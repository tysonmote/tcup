package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	// Command-line flags
	in          = flag.String("in", "127.0.0.1:8125", "TCP listening address")
	out         = flag.String("out", "127.0.0.1:8125", "UDP destination address")
	token       = flag.String("token", "", "Expected \"X-Token\" header for all requests")
	cert        = flag.String("cert", "cert.pem", "Path to SSL certificate file")
	key         = flag.String("key", "key.pem", "Path to SSL key file")
	logInterval = flag.Int("log", 0, "Stats logging interval. A value of 0 will cause no stats to be logged")
	help        = flag.Bool("help", false, "Print usage info")

	// Stdout logging
	logger *log.Logger

	// Statsd connection
	writeLock sync.Mutex
	outConn   net.Conn

	// Stats
	requestsReceived uint64
	bytesReceived    uint64
	totalRequestTime float64
)

// Print usage information and exit.
func usage() {
	fmt.Printf("Usage: tcup [flags]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

// Log stats to the logger and reset them.
func logStats() {
	if requestsReceived > 0 {
		avgRequestMs := strconv.FormatFloat(totalRequestTime/float64(requestsReceived), 'f', 3, 64)
		logger.Printf("%d requests, %d bytes received (avg. %sms)\n", requestsReceived, bytesReceived, avgRequestMs)
	}
	requestsReceived = 0
	bytesReceived = 0
	totalRequestTime = 0.0
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Header.Get("X-Token") != *token {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	writeLock.Lock()
	defer writeLock.Unlock()

	bytes, err := io.Copy(io.Writer(outConn), io.Reader(r.Body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update stats
	bytesReceived += uint64(bytes)
	requestsReceived += 1
	totalRequestTime += float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond)
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
	loggerPrefix := fmt.Sprintf("tcup[%d] ", os.Getpid())
	logger = log.New(os.Stdout, loggerPrefix, log.LstdFlags)

	// Set up UDP connection
	outConn, err = net.DialTimeout("udp", *out, time.Second)
	if err != nil {
		panic(err)
	}

	// Log stats to the logger, if needed
	go func() {
		if *logInterval > 0 {
			statsTicker := time.NewTicker(time.Duration(*logInterval) * time.Second)
			for {
				select {
				case <-statsTicker.C:
					logStats()
				}
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
