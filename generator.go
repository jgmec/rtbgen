package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// Generator configuration
type GeneratorConfig struct {
	MinImpressions int
	MaxImpressions int
	MinBidFloor    float64
	MaxBidFloor    float64
	TestMode       bool
	BoundingBox    *BoundingBox // nil = use random city geo
	NearGeo        *Geo         // if set, generate geo within 1 km of this point
}

// Default configuration
var DefaultConfig = GeneratorConfig{
	MinImpressions: 1,
	MaxImpressions: 3,
	MinBidFloor:    0.5,
	MaxBidFloor:    10.0,
	TestMode:       false,
}

// tsFormat is the layout used for ext.ts in all generated bid requests.
const tsFormat = "2006-01-02T15:04:05.000-07:00"

// Helper functions
func randomID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(100000))
}

func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func randomInt(min, max int) int {
	if max <= min {
		return min
	}
	return min + rand.Intn(max-min+1)
}

func randomChoice(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[rand.Intn(len(choices))]
}

func randomChoiceInt(choices []int) int {
	if len(choices) == 0 {
		return 0
	}
	return choices[rand.Intn(len(choices))]
}

func randomBool() bool {
	return rand.Intn(2) == 1
}

func intPtr(i int) *int {
	return &i
}

// Banner generation
func generateBanner() *Banner {
	formats := []Format{
		{W: 300, H: 250}, // Medium Rectangle
		{W: 728, H: 90},  // Leaderboard
		{W: 320, H: 50},  // Mobile Banner
		{W: 160, H: 600}, // Wide Skyscraper
		{W: 970, H: 250}, // Billboard
		{W: 300, H: 600}, // Half Page
		{W: 320, H: 100}, // Large Mobile Banner
		{W: 468, H: 60},  // Full Banner
	}

	selectedFormat := formats[rand.Intn(len(formats))]

	banner := &Banner{
		W:      &selectedFormat.W,
		H:      &selectedFormat.H,
		Format: []Format{selectedFormat},
		Pos:    randomInt(0, 7), // Ad position
		API:    []int{3, 5},     // MRAID-1, MRAID-2
		MIMEs:  []string{"image/jpeg", "image/png", "image/gif"},
	}

	// Optionally add additional formats
	if randomBool() {
		additionalFormat := formats[rand.Intn(len(formats))]
		banner.Format = append(banner.Format, additionalFormat)
	}

	return banner
}

// Video generation
func generateVideo() *Video {
	widths := []int{640, 854, 1280, 1920}
	heights := []int{360, 480, 720, 1080}

	video := &Video{
		MIMEs:          []string{"video/mp4", "video/x-flv", "video/webm"},
		MinDuration:    5,
		MaxDuration:    randomInt(15, 60),
		Protocols:      []int{2, 3, 5, 6}, // VAST 2.0, 3.0, VAST 2.0 Wrapper, VAST 3.0 Wrapper
		W:              randomChoiceInt(widths),
		H:              randomChoiceInt(heights),
		StartDelay:     randomInt(-2, 5), // -2=generic mid-roll, -1=generic post-roll, 0=pre-roll, >0=mid-roll
		Placement:      randomInt(1, 5),  // 1=in-stream, 2=in-banner, 3=in-article, 4=in-feed, 5=interstitial
		Linearity:      1,                // Linear/In-Stream
		PlaybackMethod: []int{randomInt(1, 6)},
		API:            []int{1, 2, 3, 5}, // VPAID 1.0, VPAID 2.0, MRAID-1, MRAID-2
		MinBitrate:     300,
		MaxBitrate:     2000,
		BoxingAllowed:  randomInt(0, 1),
	}

	// Add skip options randomly
	if randomBool() {
		video.Skip = 1
		video.SkipMin = 5
		video.SkipAfter = 5
	}

	return video
}

// Audio generation
func generateAudio() *Audio {
	return &Audio{
		MIMEs:       []string{"audio/mp3", "audio/mp4", "audio/aac"},
		MinDuration: 5,
		MaxDuration: randomInt(15, 60),
		Protocols:   []int{2, 3, 5, 6},
		StartDelay:  randomInt(-2, 5),
		API:         []int{1, 2},
		MinBitrate:  64,
		MaxBitrate:  320,
		Feed:        randomInt(1, 3), // 1=music, 2=broadcast, 3=podcast
	}
}

// Native generation
func generateNative() *Native {
	nativeRequest := map[string]any{
		"ver": "1.2",
		"assets": []map[string]any{
			{
				"id":       1,
				"required": 1,
				"title": map[string]any{
					"len": 80,
				},
			},
			{
				"id": 2,
				"img": map[string]any{
					"type": 3,
					"w":    300,
					"h":    250,
				},
			},
		},
	}

	jsonData, _ := json.Marshal(nativeRequest)

	// Convert to JSON string (simplified for example)
	return &Native{
		Request: string(jsonData),
		Ver:     "1.2",
		API:     []int{3, 5},
	}
}

// Impression generation
func generateImpression(impType string, config GeneratorConfig) Imp {
	imp := Imp{
		ID:          randomID(),
		TagID:       fmt.Sprintf("tag-%d", rand.Intn(10000)),
		BidFloor:    randomFloat(config.MinBidFloor, config.MaxBidFloor),
		BidFloorCur: "USD",
		Secure:      intPtr(1),
	}

	// Set display manager
	if randomBool() {
		imp.DisplayManager = randomChoice([]string{"GoogleAds", "MoPub", "AdMob", "Smaato"})
		imp.DisplayManagerVer = fmt.Sprintf("%d.%d.%d", randomInt(1, 5), randomInt(0, 9), randomInt(0, 9))
	}

	// Add impression type
	switch impType {
	case "banner":
		imp.Banner = generateBanner()
	case "video":
		imp.Video = generateVideo()
	case "audio":
		imp.Audio = generateAudio()
	case "native":
		imp.Native = generateNative()
	default:
		// Random type
		types := []string{"banner", "video"}
		selectedType := randomChoice(types)
		if selectedType == "banner" {
			imp.Banner = generateBanner()
		} else {
			imp.Video = generateVideo()
		}
	}

	return imp
}

// Site generation
func generateSite() *Site {
	domains := []string{
		"example.com", "news-site.com", "blog.net", "magazine.org", "portal.io",
		"tech-news.com", "sports-daily.com", "entertainment.net", "finance-hub.com",
	}
	categories := []string{
		"IAB1",  // Arts & Entertainment
		"IAB2",  // Automotive
		"IAB3",  // Business
		"IAB4",  // Careers
		"IAB12", // News
		"IAB15", // Science
		"IAB17", // Sports
		"IAB19", // Technology & Computing
		"IAB20", // Travel
	}

	domain := randomChoice(domains)

	site := &Site{
		ID:     randomID(),
		Name:   "Sample Site",
		Domain: domain,
		Cat:    []string{randomChoice(categories)},
		Page:   fmt.Sprintf("https://%s/page-%d", domain, rand.Intn(10000)),
		Publisher: &Publisher{
			ID:     randomID(),
			Name:   "Sample Publisher",
			Domain: domain,
		},
		PrivacyPolicy: 1,
		Mobile:        randomInt(0, 1),
	}

	// Add optional fields
	if randomBool() {
		site.Ref = fmt.Sprintf("https://%s/referrer", randomChoice(domains))
	}

	if randomBool() {
		site.Keywords = randomChoice([]string{
			"technology,gadgets,reviews",
			"sports,football,basketball",
			"news,politics,world",
			"entertainment,movies,music",
		})
	}

	return site
}

// App generation
func generateApp() *App {
	type appTemplate struct {
		name      string
		bundle    string
		domain    string
		cat       string
		keywords  string
		publisher string
	}
	templates := []appTemplate{
		// Games
		{"Pixel Dungeon Quest", "com.pixelstudio.dungeonquest", "pixelstudio.com", "IAB9-30", "gaming,rpg,dungeon,pixel", "Pixel Studio Games"},
		{"Merge Island Escape", "com.tropiclab.mergeisland", "tropiclab.io", "IAB9-30", "gaming,casual,merge,puzzle", "Tropic Lab"},
		{"Shadow Legends Strike", "com.darkforge.shadowstrike", "darkforge.gg", "IAB9-30", "gaming,action,rpg,fantasy", "Dark Forge Entertainment"},
		{"Idle Farm Empire", "com.sunleaf.idlefarm", "sunleaf.games", "IAB9-30", "gaming,idle,farm,simulation", "Sunleaf Games"},
		{"Block Blast Mania", "com.puzzlejoy.blockblast", "puzzlejoy.com", "IAB9-30", "gaming,puzzle,blocks,casual", "Puzzle Joy Ltd"},
		{"Castle Defense Wars", "com.ironkeep.castledefense", "ironkeep.io", "IAB9-30", "gaming,strategy,defense,castle", "Iron Keep Studios"},
		{"Word Storm Champions", "com.lexicraft.wordstorm", "lexicraft.com", "IAB9-30", "gaming,word,puzzle,educational", "LexiCraft"},
		{"Turbo Race GT", "com.nitroworks.turborace", "nitroworks.io", "IAB9-30", "gaming,racing,cars,multiplayer", "Nitro Works"},
		// News & Media
		{"FlashNews Daily", "com.medianow.flashnews", "medianow.com", "IAB12", "news,breaking,daily,headlines", "Media Now Inc"},
		{"PocketTribune", "com.tribemedia.pockettribune", "tribemedia.net", "IAB12", "news,politics,world,live", "Tribe Media Group"},
		{"SportsBuzz Live", "com.sportsbuzz.live", "sportsbuzz.tv", "IAB17", "sports,news,live,scores", "SportsBuzz Media"},
		{"Finance Pulse", "com.wealthtrack.financepulse", "wealthtrack.com", "IAB13", "finance,stocks,markets,investing", "WealthTrack"},
		// Social & Messaging
		{"SnapCircle", "com.circleapp.snapcircle", "snapcircle.app", "IAB14", "social,messaging,friends,sharing", "Circle App Co"},
		{"BondChat", "com.bondlabs.bondchat", "bondlabs.io", "IAB14", "social,chat,video,messaging", "Bond Labs"},
		{"CrowdVibe", "com.vibetech.crowdvibe", "crowdvibe.app", "IAB14", "social,community,events,local", "VibeTech"},
		// Shopping & Deals
		{"DealHunter Pro", "com.savvybyte.dealhunter", "dealhunter.io", "IAB22", "shopping,deals,coupons,savings", "Savvy Byte Apps"},
		{"StyleCart", "com.fashionloop.stylecart", "stylecart.com", "IAB22", "shopping,fashion,clothing,style", "FashionLoop"},
		{"GroceryDash", "com.freshroute.grocerydash", "grocerydash.app", "IAB22", "shopping,grocery,delivery,food", "Fresh Route Inc"},
		// Health & Fitness
		{"FitSprint Coach", "com.motionlab.fitsprint", "motionlab.fit", "IAB7", "fitness,workout,coach,health", "Motion Lab"},
		{"ZenMind Meditation", "com.quietpath.zenmind", "quietpath.app", "IAB7", "health,meditation,mindfulness,sleep", "Quiet Path"},
		{"CalTrack Nutrition", "com.nutricore.caltrack", "nutricore.io", "IAB7", "health,nutrition,calories,diet", "NutriCore"},
		// Music & Audio
		{"BeatWave Music", "com.waveform.beatwave", "beatwave.fm", "IAB1-6", "music,streaming,beats,playlist", "Waveform Audio"},
		{"PodcastVault", "com.audioworks.podcastvault", "podcastvault.fm", "IAB1-6", "podcasts,audio,radio,talk", "AudioWorks"},
		{"TuneBox Radio", "com.radiosync.tunebox", "tunebox.app", "IAB1-6", "music,radio,stations,live", "RadioSync"},
		// Video & Entertainment
		{"StreamFlick", "com.flicknet.streamflick", "streamflick.tv", "IAB1", "video,streaming,movies,tv", "FlickNet"},
		{"ClipReel Short Video", "com.reelco.clipreel", "clipreel.app", "IAB1", "video,shorts,clips,viral", "Reel Co"},
		{"LiveStage Events", "com.stagenet.livestage", "livestage.tv", "IAB1", "video,live,events,concerts", "StageNet"},
		// Travel
		{"WanderMap Travel", "com.roamtech.wandermap", "wandermap.app", "IAB20", "travel,maps,explore,trips", "Roam Tech"},
		{"HotelDirect Booking", "com.stayvault.hoteldirect", "hoteldirect.com", "IAB20", "travel,hotels,booking,vacation", "StayVault"},
		{"FlightAlert Pro", "com.skywatch.flightalert", "flightalert.io", "IAB20", "travel,flights,deals,alerts", "SkyWatch"},
		// Utilities
		{"VaultKeeper Password", "com.securebit.vaultkeeper", "securebit.io", "IAB19", "utilities,security,password,privacy", "SecureBit"},
		{"CleanBoost Pro", "com.cleantech.cleanboost", "cleanboost.app", "IAB19", "utilities,cleaner,booster,storage", "CleanTech Apps"},
		{"ScanDoc Scanner", "com.docusnap.scandoc", "docusnap.io", "IAB19", "utilities,scanner,pdf,documents", "DocuSnap"},
	}

	tmpl := templates[rand.Intn(len(templates))]
	storeType := randomChoice([]string{"play", "itunes"})

	var storeURL string
	if storeType == "play" {
		storeURL = fmt.Sprintf("https://play.google.com/store/apps/details?id=%s", tmpl.bundle)
	} else {
		storeURL = fmt.Sprintf("https://apps.apple.com/app/id%d", rand.Intn(1000000000))
	}

	app := &App{
		ID:       randomID(),
		Name:     tmpl.name,
		Bundle:   tmpl.bundle,
		Domain:   tmpl.domain,
		Cat:      []string{tmpl.cat},
		StoreURL: storeURL,
		Ver:      fmt.Sprintf("%d.%d.%d", randomInt(1, 8), randomInt(0, 15), randomInt(0, 9)),
		Publisher: &Publisher{
			ID:   randomID(),
			Name: tmpl.publisher,
		},
		PrivacyPolicy: 1,
		Paid:          randomInt(0, 1),
	}

	if randomBool() {
		app.Keywords = tmpl.keywords
	}

	return app
}

// Device generation
func generateDevice(config GeneratorConfig) *Device {
	oses := []string{"iOS", "Android", "Windows", "MacOS"}
	makes := []string{"Apple", "Samsung", "Google", "Huawei", "Xiaomi", "OnePlus", "LG", "Motorola"}
	carriers := []string{"Verizon", "AT&T", "T-Mobile", "Sprint", "Vodafone", "Orange", "O2"}

	os := randomChoice(oses)
	var make string
	if os == "iOS" {
		make = "Apple"
	} else {
		make = randomChoice(makes)
	}

	widths := []int{320, 375, 414, 768, 1024, 1920}
	heights := []int{568, 667, 812, 1024, 1366, 1080}

	geo := generateGeo()
	if config.NearGeo != nil {
		geo = generateGeoNear(config.NearGeo.Lat, config.NearGeo.Lon, 1.0)
	} else if config.BoundingBox != nil {
		geo = generateGeoInBBox(config.BoundingBox)
	}

	device := &Device{
		UA:             generateUA(os, make),
		Geo:            geo,
		DNT:            randomInt(0, 1),
		Lmt:            randomInt(0, 1),
		IP:             fmt.Sprintf("%d.%d.%d.%d", randomInt(1, 255), randomInt(1, 255), randomInt(1, 255), randomInt(1, 255)),
		DeviceType:     randomInt(1, 7), // 1=Mobile/Tablet, 2=PC, 3=Connected TV, 4=Phone, 5=Tablet, 6=Connected Device, 7=Set Top Box
		Make:           make,
		Model:          fmt.Sprintf("Model-%d", rand.Intn(20)),
		OS:             os,
		OSV:            fmt.Sprintf("%d.%d", randomInt(8, 15), randomInt(0, 5)),
		Language:       randomChoice([]string{"en", "es", "fr", "de", "zh", "ja", "pt", "ru"}),
		Carrier:        randomChoice(carriers),
		ConnectionType: randomInt(0, 6), // 0=unknown, 1=Ethernet, 2=WIFI, 3=Cellular-Unknown, 4=2G, 5=3G, 6=4G
		IFA:            uuid.New().String(),
		JS:             1,
		W:              randomChoiceInt(widths),
		H:              randomChoiceInt(heights),
	}

	// Add optional IPv6
	if randomBool() {
		device.IPv6 = fmt.Sprintf("2001:0db8:85a3:0000:0000:8a2e:%04x:%04x", rand.Intn(65536), rand.Intn(65536))
	}

	// Add optional hardware version
	if randomBool() {
		device.HWV = fmt.Sprintf("HW-%d.%d", randomInt(1, 10), randomInt(0, 9))
	}

	// Add PPI and pixel ratio for mobile devices
	if device.DeviceType == 4 || device.DeviceType == 5 {
		ppis := []int{264, 326, 401, 458}
		device.PPI = randomChoiceInt(ppis)
		device.PxRatio = randomFloat(1.0, 3.0)
	}

	return device
}

func generateUA(os, make string) string {
	switch os {
	case "iOS":
		return fmt.Sprintf("Mozilla/5.0 (iPhone; CPU iPhone OS %d_%d like Mac OS X) AppleWebKit/605.1.15", randomInt(14, 17), randomInt(0, 5))
	case "Android":
		return fmt.Sprintf("Mozilla/5.0 (Linux; Android %d; %s) AppleWebKit/537.36", randomInt(9, 14), make)
	case "Windows":
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	case "MacOS":
		return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_%d_%d) AppleWebKit/537.36", randomInt(13, 15), randomInt(0, 7))
	default:
		return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36", os)
	}
}

// Geo generation
func generateGeo() *Geo {
	cities := []struct {
		city    string
		region  string
		country string
		lat     float64
		lon     float64
		zip     string
	}{
		{"Los Angeles", "CA", "USA", 34.0522, -118.2437, "90001"},
		{"New York", "NY", "USA", 40.7128, -74.0060, "10001"},
		{"Chicago", "IL", "USA", 41.8781, -87.6298, "60601"},
		{"Houston", "TX", "USA", 29.7604, -95.3698, "77001"},
		{"Phoenix", "AZ", "USA", 33.4484, -112.0740, "85001"},
		{"London", "ENG", "GBR", 51.5074, -0.1278, "SW1A"},
		{"Paris", "IDF", "FRA", 48.8566, 2.3522, "75001"},
		{"Berlin", "BE", "DEU", 52.5200, 13.4050, "10115"},
		{"Tokyo", "13", "JPN", 35.6762, 139.6503, "100-0001"},
		{"Sydney", "NSW", "AUS", -33.8688, 151.2093, "2000"},
	}

	city := cities[rand.Intn(len(cities))]

	return &Geo{
		Lat:     city.lat + randomFloat(-0.1, 0.1), // Add some variance
		Lon:     city.lon + randomFloat(-0.1, 0.1),
		Country: city.country,
		Region:  city.region,
		City:    city.city,
		Zip:     city.zip,
		Type:    randomInt(1, 3), // 1=GPS/Location Services, 2=IP Address, 3=User provided
	}
}

// User generation
func generateUser() *User {
	user := &User{
		ID:       randomID(),
		BuyerUID: randomID(),
		Yob:      randomInt(1950, 2005),
		Gender:   randomChoice([]string{"M", "F", "O"}),
	}

	if randomBool() {
		user.Keywords = randomChoice([]string{
			"sports,technology,gaming",
			"travel,food,lifestyle",
			"fashion,beauty,shopping",
			"business,finance,investing",
			"health,fitness,wellness",
		})
	}

	// Optionally add user data segments
	if randomBool() {
		user.Data = []Data{
			{
				ID:   randomID(),
				Name: "DataProvider1",
				Segment: []Segment{
					{
						ID:    fmt.Sprintf("seg-%d", rand.Intn(1000)),
						Name:  randomChoice([]string{"Sports Enthusiast", "Tech Savvy", "Luxury Shopper", "Frequent Traveler"}),
						Value: fmt.Sprintf("%.2f", randomFloat(0.5, 1.0)),
					},
				},
			},
		}
	}

	return user
}

// Regulations generation
func generateRegs() *Regs {
	return &Regs{
		Coppa: randomInt(0, 1), // Children's Online Privacy Protection Act
	}
}

// Main bid request generator
func GenerateRandomBidRequest(requestType string, impType string) *BidRequest {
	return GenerateRandomBidRequestWithConfig(requestType, impType, DefaultConfig)
}

func GenerateRandomBidRequestWithConfig(requestType string, impType string, config GeneratorConfig) *BidRequest {
	req := &BidRequest{
		ID:     randomID(),
		AT:     randomInt(1, 2), // 1=First Price Auction, 2=Second Price Auction
		TMax:   randomInt(100, 500),
		Device: generateDevice(config),
		User:   generateUser(),
		Regs:   generateRegs(),
		Test:   0,
	}

	if config.TestMode {
		req.Test = 1
	}

	// Add impressions
	numImpressions := randomInt(config.MinImpressions, config.MaxImpressions)
	req.Imp = make([]Imp, numImpressions)
	for i := 0; i < numImpressions; i++ {
		req.Imp[i] = generateImpression(impType, config)
	}

	// Add either site or app
	switch requestType {
	case "site":
		req.Site = generateSite()
	case "app":
		req.App = generateApp()
	default:
		// Random selection
		if randomBool() {
			req.Site = generateSite()
		} else {
			req.App = generateApp()
		}
	}

	// Optionally add blocked categories or advertisers
	if randomBool() {
		req.BCat = []string{"IAB25", "IAB26"} // Adult content, Illegal content
	}

	if randomBool() {
		req.BAdv = []string{"blocked-advertiser.com"}
	}

	return req
}

// Batch generator
func GenerateBatch(count int, requestType string, impType string) []*BidRequest {
	requests := make([]*BidRequest, count)
	for i := 0; i < count; i++ {
		requests[i] = GenerateRandomBidRequest(requestType, impType)
	}
	return requests
}

// randomTimestamp returns a random time in [start, end).
func randomTimestamp(start, end time.Time) time.Time {
	delta := end.Sub(start)
	if delta <= 0 {
		return start
	}
	return start.Add(time.Duration(rand.Int63n(int64(delta))))
}

// generateGeoNear generates a Geo point uniformly within radiusKm of (lat, lon).
func generateGeoNear(lat, lon, radiusKm float64) *Geo {
	const kmPerDegree = 111.0
	angle := rand.Float64() * 2 * math.Pi
	// sqrt gives uniform distribution within the circle
	distance := math.Sqrt(rand.Float64()) * radiusKm
	deltaLat := distance * math.Cos(angle) / kmPerDegree
	deltaLon := distance * math.Sin(angle) / (kmPerDegree * math.Cos(lat*math.Pi/180))
	return &Geo{
		Lat:  lat + deltaLat,
		Lon:  lon + deltaLon,
		Type: 1, // GPS
	}
}

// generateGeoInBBox generates a Geo point uniformly within the bounding box.
func generateGeoInBBox(bbox *BoundingBox) *Geo {
	return &Geo{
		Lat:  randomFloat(bbox.MinLat, bbox.MaxLat),
		Lon:  randomFloat(bbox.MinLon, bbox.MaxLon),
		Type: 1, // GPS/Location Services
	}
}

// generateRequestForTask creates a BidRequest tailored to a task's criteria,
// with a random timestamp in [windowStart, windowEnd).
func generateRequestForTask(task *Task, ts time.Time, baseGeo *Geo, deviceIFA string) *BidRequest {
	config := DefaultConfig
	switch task.CriteriaType {
	case CriteriaBBox:
		if task.Geometry != nil {
			if bb, err := task.Geometry.bbox(); err == nil {
				config.BoundingBox = bb
			}
		}
	case CriteriaIFA, CriteriaIP:
		config.NearGeo = baseGeo
	}

	req := GenerateRandomBidRequestWithConfig("random", "banner", config)

	switch task.CriteriaType {
	case CriteriaIP:
		req.Device.IP = task.IPAddress
		if deviceIFA != "" {
			req.Device.IFA = deviceIFA
		}
	case CriteriaBBox:
		if deviceIFA != "" {
			req.Device.IFA = deviceIFA
		}
	case CriteriaIFA:
		req.Device.IFA = task.IFA
	}

	req.Ext = map[string]any{
		"task_id":        task.CorrelationID,
		"correlation_id": task.CorrelationID,
		"ts":             ts.UTC().Format(tsFormat),
	}

	return req
}
