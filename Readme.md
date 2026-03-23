# OpenRTB 2.5 Bid Request Generator

A Go application for generating random OpenRTB 2.5 compliant bid requests. This tool is useful for testing ad exchanges, Demand-Side Platforms (DSPs), and Supply-Side Platforms (SSPs).

## Project Structure

```
rtb-generator/
├── main.go          # Main entry point with CLI
├── models.go        # OpenRTB 2.5 data structures
├── generator.go     # Random data generation logic
├── main_test.go     # Unit tests
├── go.mod           # Go module file
├── README.md        # This file
└── EXAMPLES.md      # Sample outputs
```

## Features

- **Complete OpenRTB 2.5 Support**: All major objects and fields
- **Multiple Impression Types**:
  - Banner ads (various sizes)
  - Video ads (in-stream, out-stream)
  - Audio ads
  - Native ads
- **Flexible Configuration**: Customizable bid floors, impression counts, etc.
- **Random Data Generation**:
  - Site and App contexts
  - Device information (mobile, tablet, desktop, CTV)
  - User demographics and segments
  - Geolocation data (major cities worldwide)
  - Publisher information
  - Privacy regulations (COPPA, DNT, LMT)
- **CLI and REST API**: Use as command-line tool or HTTP server
- **Batch Generation**: Create multiple requests at once
- **Flexible Output**: Pretty-printed or compact JSON

## Installation

```bash
# Clone or download the code
cd rtb-generator

# Build the application
go build -o rtb-generator

# Or run directly
go run .
```

## Usage

### Command Line Interface

#### Basic Usage

```bash
# Generate a single random banner request
./rtb-generator

# Generate a site banner request
./rtb-generator -type=site -imp=banner

# Generate an app video request
./rtb-generator -type=app -imp=video

# Generate multiple requests
./rtb-generator -count=5

# Generate with test flag enabled
./rtb-generator -test

# Generate compact JSON (no indentation)
./rtb-generator -compact

# Show example outputs
./rtb-generator -examples
```

#### Command Line Flags

- `-type` - Request type: `site`, `app`, or `random` (default: `random`)
- `-imp` - Impression type: `banner`, `video`, `audio`, `native` (default: `banner`)
- `-count` - Number of requests to generate (default: `1`)
- `-test` - Set test flag in requests (default: `false`)
- `-pretty` - Pretty print JSON output with indentation (default: `true`)
- `-compact` - Compact JSON output, no indentation (overrides `-pretty`)
- `-examples` - Show example requests and exit

#### Examples

```bash
# Compact output for piping
./rtb-generator -compact | jq .

# Generate 10 video requests
./rtb-generator -imp=video -count=10

# Generate test mode requests
./rtb-generator -test -type=app

# Audio ad for site
./rtb-generator -type=site -imp=audio
```

### Programmatic Usage

Import and use in your own Go code:

```go
package main

import (
    "encoding/json"
    "fmt"
    "math/rand"
    "time"
)

func main() {
    rand.Seed(time.Now().UnixNano())
    
    // Generate a banner ad request for a website
    bidReq := GenerateRandomBidRequest("site", "banner")
    
    // Generate with custom configuration
    config := GeneratorConfig{
        MinImpressions: 2,
        MaxImpressions: 5,
        MinBidFloor:    1.0,
        MaxBidFloor:    15.0,
        TestMode:       true,
    }
    customReq := GenerateRandomBidRequestWithConfig("app", "video", config)
    
    // Generate batch
    batch := GenerateBatch(10, "random", "banner")
    
    // Convert to JSON
    jsonData, _ := json.MarshalIndent(bidReq, "", "  ")
    fmt.Println(string(jsonData))
}
```

## Output Example

```json
{
  "id": "1706990000000000000-12345",
  "imp": [
    {
      "id": "1706990000000000001-67890",
      "banner": {
        "w": 300,
        "h": 250,
        "format": [
          {
            "w": 300,
            "h": 250
          }
        ],
        "pos": 3,
        "api": [3, 5],
        "mimes": ["image/jpeg", "image/png", "image/gif"]
      },
      "tagid": "tag-1234",
      "bidfloor": 2.5,
      "bidfloorcur": "USD",
      "secure": 1
    }
  ],
  "site": {
    "id": "site-123",
    "name": "Sample Site",
    "domain": "example.com",
    "cat": ["IAB1"],
    "page": "https://example.com/page-456",
    "publisher": {
      "id": "pub-789",
      "name": "Sample Publisher",
      "domain": "example.com"
    },
    "privacypolicy": 1,
    "mobile": 0
  },
  "device": {
    "ua": "Mozilla/5.0 (Samsung; Android 12) AppleWebKit/537.36",
    "geo": {
      "lat": 34.0522,
      "lon": -118.2437,
      "country": "USA",
      "region": "CA",
      "city": "Los Angeles",
      "zip": "90001",
      "type": 2
    },
    "dnt": 0,
    "lmt": 0,
    "ip": "192.168.1.1",
    "devicetype": 4,
    "make": "Samsung",
    "model": "Model-5",
    "os": "Android",
    "osv": "12.0",
    "language": "en",
    "carrier": "Verizon",
    "connectiontype": 2,
    "ifa": "12345678-1234-1234-1234-123456789012",
    "js": 1,
    "w": 414,
    "h": 812
  },
  "user": {
    "id": "user-456",
    "buyeruid": "buyer-789",
    "yob": 1985,
    "gender": "M",
    "keywords": "sports,technology,gaming"
  },
  "regs": {
    "coppa": 0
  },
  "at": 1,
  "tmax": 250
}
```

## OpenRTB 2.5 Compliance

This generator creates bid requests following the OpenRTB 2.5 specification with:

- Required fields (id, imp, at)
- Common optional fields (site/app, device, user, regs)
- Standard IAB categories
- Device types and connection types
- Banner formats and positions
- Video parameters and protocols
- Audio stream types
- Native ad specifications
- Privacy indicators (DNT, LMT, COPPA)
- Private marketplace (PMP) support
- User data segments

## Customization

You can easily customize the generator by modifying:

- **Banner sizes**: Edit the `formats` array in `generateBanner()` (generator.go)
- **Device types**: Modify the device generation logic in `generateDevice()` (generator.go)
- **Geographic regions**: Update the cities array in `generateGeo()` (generator.go)
- **Categories**: Change IAB category codes in site/app generation (generator.go)
- **Bid floor ranges**: Adjust min/max values in `GeneratorConfig` (generator.go)
- **Data models**: Add or modify OpenRTB fields in `models.go`

## Use Cases

- **Testing DSP bid logic**: Generate diverse bid requests for testing
- **Load testing**: Create high volumes of realistic bid requests
- **Integration testing**: Validate OpenRTB endpoint implementations
- **Development**: Mock data for SSP/Exchange development
- **Quality assurance**: Test ad rendering for various formats
- **Training**: Learn OpenRTB spec with real examples

## Testing

Run the test suite:

```bash
# Run all tests
go test -v

# Run benchmarks
go test -bench=. -benchmem

# Run specific test
go test -run TestGenerateRandomBidRequest
```

## License

MIT License - Feel free to use and modify for your needs.

## Contributing

Contributions are welcome! Feel free to submit issues or pull requests.