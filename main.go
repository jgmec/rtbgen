package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

// Global flag for pretty printing
var prettyPrint bool

func main() {
	// Command line flags
	requestType := flag.String("type", "random", "Request type: site, app, or random")
	impType := flag.String("imp", "banner", "Impression type: banner, video, audio, native")
	count := flag.Int("count", 1, "Number of requests to generate")
	testMode := flag.Bool("test", false, "Set test flag in requests")
	examples := flag.Bool("examples", false, "Show example requests and exit")
	pretty := flag.Bool("pretty", true, "Pretty print JSON output (default true)")
	compact := flag.Bool("compact", false, "Compact JSON output (no indentation)")

	flag.Parse()

	// Set pretty print mode (compact overrides pretty)
	if *compact {
		prettyPrint = false
	} else {
		prettyPrint = *pretty
	}

	// Show examples if requested
	if *examples {
		showExamples()
		return
	}

	// Set up configuration
	config := DefaultConfig
	config.TestMode = *testMode

	// Generate RTB requests based on flags
	for i := 0; i < *count; i++ {
		req := GenerateRandomBidRequestWithConfig(*requestType, *impType, config)
		printJSON(req)
	}

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
