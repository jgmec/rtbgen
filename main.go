package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// Global flag for pretty printing
var prettyPrint bool

func main() {
	// Shared flags
	serverMode := flag.Bool("server", false, "Run as HTTP server")
	port := flag.String("port", "8080", "HTTP server port (server mode only)")
	tasksFile := flag.String("tasks-file", "tasks.json", "Path to task persistence file (server mode only)")
	outDir := flag.String("out-dir", "output", "Directory for generated JSONL files (server mode only)")
	schedulerInterval := flag.Duration("scheduler-interval", 5*time.Minute, "How often the scheduler generates requests for active tasks (server mode only)")
	mmdbPath := flag.String("mmdb", "", "Path to MaxMind GeoIP2 City MMDB file (optional)")
	nominatimURL := flag.String("nominatim-url", "https://nominatim.openstreetmap.org", "Base URL for Nominatim reverse geocoding (optional)")
	tlsCert := flag.String("tls-cert", "", "Path to TLS certificate file (enables HTTPS)")
	tlsKey := flag.String("tls-key", "", "Path to TLS private key file (enables HTTPS)")

	// CLI generation flags
	requestType := flag.String("type", "random", "Request type: site, app, or random")
	impType := flag.String("imp", "banner", "Impression type: banner, video, audio, native")
	count := flag.Int("count", 1, "Number of requests to generate")
	testMode := flag.Bool("test", false, "Set test flag in requests")
	examples := flag.Bool("examples", false, "Show example requests and exit")
	pretty := flag.Bool("pretty", true, "Pretty print JSON output (default true)")
	compact := flag.Bool("compact", false, "Compact JSON output (no indentation)")
	bbox := flag.String("bbox", "", "Bounding box filter: maxlat,maxlon,minlat,minlon")

	flag.Parse()

	if *serverMode {
		runServer(*port, *tasksFile, *outDir, *schedulerInterval, *mmdbPath, *nominatimURL, *tlsCert, *tlsKey)
		return
	}

	// CLI mode
	if *compact {
		prettyPrint = false
	} else {
		prettyPrint = *pretty
	}

	if *examples {
		showExamples()
		return
	}

	config := DefaultConfig
	config.TestMode = *testMode

	if *bbox != "" {
		bb, err := parseBoundingBox(*bbox)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid -bbox: %v\n", err)
			os.Exit(1)
		}
		config.BoundingBox = bb
	}

	for i := 0; i < *count; i++ {
		now := time.Now()
		req := GenerateRandomBidRequestWithConfig(*requestType, *impType, config)
		ts := randomTimestamp(now.Add(-*schedulerInterval), now)
		req.Ext = map[string]any{"timestamp": ts.UnixMilli()}
		printJSON(req)
	}
}

func runServer(port, tasksFile, outDir string, schedulerInterval time.Duration, mmdbPath, nominatimURL, tlsCert, tlsKey string) {
	store, err := NewTaskStore(tasksFile)
	if err != nil {
		log.Fatalf("load task store: %v", err)
	}

	var mmdb *geoip2.Reader
	if mmdbPath != "" {
		mmdb, err = geoip2.Open(mmdbPath)
		if err != nil {
			log.Fatalf("open mmdb: %v", err)
		}
		defer mmdb.Close()
		log.Printf("MaxMind MMDB loaded: %s", mmdbPath)
	}

	var geocoder *ReverseGeocoder
	if nominatimURL != "" {
		geocoder = NewReverseGeocoder(nominatimURL)
		log.Printf("reverse geocoding enabled: %s", nominatimURL)
	}

	scheduler := NewScheduler(store, outDir, schedulerInterval, mmdb, geocoder)
	go scheduler.Start()

	srv := NewServer(store, mmdb)
	addr := ":" + port
	handler := srv.Handler()
	if tlsCert != "" && tlsKey != "" {
		log.Printf("HTTPS server listening on %s", addr)
		if err := http.ListenAndServeTLS(addr, tlsCert, tlsKey, handler); err != nil {
			log.Fatalf("server: %v", err)
		}
	} else {
		log.Printf("HTTP server listening on %s", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Fatalf("server: %v", err)
		}
	}
}

// parseBoundingBox parses "maxlat,maxlon,minlat,minlon".
func parseBoundingBox(s string) (*BoundingBox, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("expected maxlat,maxlon,minlat,minlon")
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q: %w", p, err)
		}
		vals[i] = v
	}
	return &BoundingBox{MaxLat: vals[0], MaxLon: vals[1], MinLat: vals[2], MinLon: vals[3]}, nil
}

func showExamples() {
	fmt.Println("=== OpenRTB 2.5 Bid Request Generator Examples ===")

	fmt.Println("Example 1: Banner Display Ad - Site")
	fmt.Println("Command: ./rtb-generator -type=site -imp=banner")
	fmt.Println("---")
	siteBannerReq := GenerateRandomBidRequest("site", "banner")
	printJSON(siteBannerReq)

	fmt.Println("\n\nExample 2: Video Ad - App")
	fmt.Println("Command: ./rtb-generator -type=app -imp=video")
	fmt.Println("---")
	appVideoReq := GenerateRandomBidRequest("app", "video")
	printJSON(appVideoReq)

	fmt.Println("\n\nExample 3: Multiple Random Requests")
	fmt.Println("Command: ./rtb-generator -count=3")
	fmt.Println("---")
	batch := GenerateBatch(3, "random", "banner")
	printJSON(batch)

	fmt.Println("\n\nExample 4: Compact Output")
	fmt.Println("Command: ./rtb-generator -compact")
	fmt.Println("---")
	// Temporarily set compact mode
	oldPretty := prettyPrint
	prettyPrint = false
	compactReq := GenerateRandomBidRequest("site", "banner")
	printJSON(compactReq)
	prettyPrint = oldPretty
}

func printJSON(v interface{}) {
	var data []byte
	var err error

	if prettyPrint {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
