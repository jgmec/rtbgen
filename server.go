package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

// Handler for generating random bid requests
func generateBidRequestHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	requestType := r.URL.Query().Get("type")
	if requestType == "" {
		requestType = "random"
	}

	impType := r.URL.Query().Get("imp")
	if impType == "" {
		impType = "banner"
	}

	// Generate bid request
	bidReq := GenerateRandomBidRequest(requestType, impType)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(bidReq); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error generating bid request", http.StatusInternalServerError)
		return
	}
}

// Handler for generating multiple bid requests
func generateBatchHandler(w http.ResponseWriter, r *http.Request) {
	// Parse count parameter
	countParam := r.URL.Query().Get("count")
	count := 10
	if countParam != "" {
		fmt.Sscanf(countParam, "%d", &count)
	}
	if count > 100 {
		count = 100 // Limit to 100 requests per batch
	}

	requestType := r.URL.Query().Get("type")
	if requestType == "" {
		requestType = "random"
	}

	impType := r.URL.Query().Get("imp")
	if impType == "" {
		impType = "banner"
	}

	// Generate multiple bid requests
	bidRequests := make([]*BidRequest, count)
	for i := 0; i < count; i++ {
		bidRequests[i] = GenerateRandomBidRequest(requestType, impType)
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(bidRequests); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error generating bid requests", http.StatusInternalServerError)
		return
	}
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "RTB 2.5 Generator",
	})
}

// API documentation handler
func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>RTB 2.5 Generator API</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        h2 { color: #666; margin-top: 30px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { background: #e3f2fd; padding: 10px; margin: 10px 0; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>OpenRTB 2.5 Bid Request Generator API</h1>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <h3>GET /generate</h3>
        <p>Generate a single random bid request</p>
        <p><strong>Query Parameters:</strong></p>
        <ul>
            <li><code>type</code> - Request type: "site", "app", or "random" (default: "random")</li>
            <li><code>imp</code> - Impression type: "banner" or "video" (default: "banner")</li>
        </ul>
        <p><strong>Example:</strong></p>
        <pre>curl "http://localhost:8080/generate?type=site&imp=banner"</pre>
    </div>
    
    <div class="endpoint">
        <h3>GET /batch</h3>
        <p>Generate multiple bid requests</p>
        <p><strong>Query Parameters:</strong></p>
        <ul>
            <li><code>count</code> - Number of requests to generate (1-100, default: 10)</li>
            <li><code>type</code> - Request type: "site", "app", or "random" (default: "random")</li>
            <li><code>imp</code> - Impression type: "banner" or "video" (default: "banner")</li>
        </ul>
        <p><strong>Example:</strong></p>
        <pre>curl "http://localhost:8080/batch?count=5&type=app&imp=video"</pre>
    </div>
    
    <div class="endpoint">
        <h3>GET /health</h3>
        <p>Health check endpoint</p>
        <p><strong>Example:</strong></p>
        <pre>curl "http://localhost:8080/health"</pre>
    </div>
    
    <h2>Examples</h2>
    
    <h3>Banner Ad for Website</h3>
    <pre>curl "http://localhost:8080/generate?type=site&imp=banner"</pre>
    
    <h3>Video Ad for Mobile App</h3>
    <pre>curl "http://localhost:8080/generate?type=app&imp=video"</pre>
    
    <h3>10 Random Bid Requests</h3>
    <pre>curl "http://localhost:8080/batch?count=10"</pre>
    
    <h2>Response Format</h2>
    <p>All responses are in JSON format following the OpenRTB 2.5 specification.</p>
</body>
</html>
`
	fmt.Fprint(w, html)
}

func mainServer() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	// Setup routes
	http.HandleFunc("/", docsHandler)
	http.HandleFunc("/generate", generateBidRequestHandler)
	http.HandleFunc("/batch", generateBatchHandler)
	http.HandleFunc("/health", healthHandler)

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting RTB 2.5 Generator Server on %s", addr)
	log.Printf("API documentation available at http://localhost:%s/", *port)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

// Uncomment the line below and comment out the main() function in main.go to run as server
// func main() { mainServer() }
