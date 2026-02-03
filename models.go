package main

// OpenRTB 2.5 Bid Request structures

type BidRequest struct {
	ID     string      `json:"id"`
	Imp    []Imp       `json:"imp"`
	Site   *Site       `json:"site,omitempty"`
	App    *App        `json:"app,omitempty"`
	Device *Device     `json:"device,omitempty"`
	User   *User       `json:"user,omitempty"`
	Test   int         `json:"test,omitempty"`
	AT     int         `json:"at"`
	TMax   int         `json:"tmax,omitempty"`
	WSeat  []string    `json:"wseat,omitempty"`
	BCat   []string    `json:"bcat,omitempty"`
	BAdv   []string    `json:"badv,omitempty"`
	Regs   *Regs       `json:"regs,omitempty"`
	Ext    interface{} `json:"ext,omitempty"`
}

type Imp struct {
	ID                string      `json:"id"`
	Banner            *Banner     `json:"banner,omitempty"`
	Video             *Video      `json:"video,omitempty"`
	Native            *Native     `json:"native,omitempty"`
	Audio             *Audio      `json:"audio,omitempty"`
	Pmp               *Pmp        `json:"pmp,omitempty"`
	DisplayManager    string      `json:"displaymanager,omitempty"`
	DisplayManagerVer string      `json:"displaymanagerver,omitempty"`
	Instl             int         `json:"instl,omitempty"`
	TagID             string      `json:"tagid,omitempty"`
	BidFloor          float64     `json:"bidfloor,omitempty"`
	BidFloorCur       string      `json:"bidfloorcur,omitempty"`
	Clickbrowser      int         `json:"clickbrowser,omitempty"`
	Secure            *int        `json:"secure,omitempty"`
	IFrameBuster      []string    `json:"iframebuster,omitempty"`
	Exp               int         `json:"exp,omitempty"`
	Ext               interface{} `json:"ext,omitempty"`
}

type Banner struct {
	W        *int        `json:"w,omitempty"`
	H        *int        `json:"h,omitempty"`
	WMax     int         `json:"wmax,omitempty"`
	HMax     int         `json:"hmax,omitempty"`
	WMin     int         `json:"wmin,omitempty"`
	HMin     int         `json:"hmin,omitempty"`
	Format   []Format    `json:"format,omitempty"`
	BType    []int       `json:"btype,omitempty"`
	BAttr    []int       `json:"battr,omitempty"`
	Pos      int         `json:"pos,omitempty"`
	MIMEs    []string    `json:"mimes,omitempty"`
	TopFrame int         `json:"topframe,omitempty"`
	ExpDir   []int       `json:"expdir,omitempty"`
	API      []int       `json:"api,omitempty"`
	ID       string      `json:"id,omitempty"`
	VCM      int         `json:"vcm,omitempty"`
	Ext      interface{} `json:"ext,omitempty"`
}

type Format struct {
	W      int         `json:"w"`
	H      int         `json:"h"`
	WRatio int         `json:"wratio,omitempty"`
	HRatio int         `json:"hratio,omitempty"`
	WMin   int         `json:"wmin,omitempty"`
	Ext    interface{} `json:"ext,omitempty"`
}

type Video struct {
	MIMEs          []string    `json:"mimes"`
	MinDuration    int         `json:"minduration,omitempty"`
	MaxDuration    int         `json:"maxduration,omitempty"`
	Protocols      []int       `json:"protocols,omitempty"`
	Protocol       int         `json:"protocol,omitempty"`
	W              int         `json:"w,omitempty"`
	H              int         `json:"h,omitempty"`
	StartDelay     int         `json:"startdelay,omitempty"`
	Placement      int         `json:"placement,omitempty"`
	Linearity      int         `json:"linearity,omitempty"`
	Skip           int         `json:"skip,omitempty"`
	SkipMin        int         `json:"skipmin,omitempty"`
	SkipAfter      int         `json:"skipafter,omitempty"`
	Sequence       int         `json:"sequence,omitempty"`
	BAttr          []int       `json:"battr,omitempty"`
	MaxExtended    int         `json:"maxextended,omitempty"`
	MinBitrate     int         `json:"minbitrate,omitempty"`
	MaxBitrate     int         `json:"maxbitrate,omitempty"`
	BoxingAllowed  int         `json:"boxingallowed,omitempty"`
	PlaybackMethod []int       `json:"playbackmethod,omitempty"`
	PlaybackEnd    int         `json:"playbackend,omitempty"`
	Delivery       []int       `json:"delivery,omitempty"`
	Pos            int         `json:"pos,omitempty"`
	CompanionAd    []Banner    `json:"companionad,omitempty"`
	API            []int       `json:"api,omitempty"`
	CompanionType  []int       `json:"companiontype,omitempty"`
	Ext            interface{} `json:"ext,omitempty"`
}

type Audio struct {
	MIMEs         []string    `json:"mimes"`
	MinDuration   int         `json:"minduration,omitempty"`
	MaxDuration   int         `json:"maxduration,omitempty"`
	Protocols     []int       `json:"protocols,omitempty"`
	StartDelay    int         `json:"startdelay,omitempty"`
	Sequence      int         `json:"sequence,omitempty"`
	BAttr         []int       `json:"battr,omitempty"`
	MaxExtended   int         `json:"maxextended,omitempty"`
	MinBitrate    int         `json:"minbitrate,omitempty"`
	MaxBitrate    int         `json:"maxbitrate,omitempty"`
	Delivery      []int       `json:"delivery,omitempty"`
	CompanionAd   []Banner    `json:"companionad,omitempty"`
	API           []int       `json:"api,omitempty"`
	CompanionType []int       `json:"companiontype,omitempty"`
	MaxSeq        int         `json:"maxseq,omitempty"`
	Feed          int         `json:"feed,omitempty"`
	Stitched      int         `json:"stitched,omitempty"`
	NVol          int         `json:"nvol,omitempty"`
	Ext           interface{} `json:"ext,omitempty"`
}

type Native struct {
	Request string      `json:"request"`
	Ver     string      `json:"ver,omitempty"`
	API     []int       `json:"api,omitempty"`
	BAttr   []int       `json:"battr,omitempty"`
	Ext     interface{} `json:"ext,omitempty"`
}

type Pmp struct {
	PrivateAuction int         `json:"private_auction,omitempty"`
	Deals          []Deal      `json:"deals,omitempty"`
	Ext            interface{} `json:"ext,omitempty"`
}

type Deal struct {
	ID          string      `json:"id"`
	BidFloor    float64     `json:"bidfloor,omitempty"`
	BidFloorCur string      `json:"bidfloorcur,omitempty"`
	AT          int         `json:"at,omitempty"`
	WSeat       []string    `json:"wseat,omitempty"`
	WADomain    []string    `json:"wadomain,omitempty"`
	Ext         interface{} `json:"ext,omitempty"`
}

type Site struct {
	ID            string      `json:"id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Domain        string      `json:"domain,omitempty"`
	Cat           []string    `json:"cat,omitempty"`
	SectionCat    []string    `json:"sectioncat,omitempty"`
	PageCat       []string    `json:"pagecat,omitempty"`
	Page          string      `json:"page,omitempty"`
	Ref           string      `json:"ref,omitempty"`
	Search        string      `json:"search,omitempty"`
	Mobile        int         `json:"mobile,omitempty"`
	PrivacyPolicy int         `json:"privacypolicy,omitempty"`
	Publisher     *Publisher  `json:"publisher,omitempty"`
	Content       *Content    `json:"content,omitempty"`
	Keywords      string      `json:"keywords,omitempty"`
	Ext           interface{} `json:"ext,omitempty"`
}

type App struct {
	ID            string      `json:"id,omitempty"`
	Name          string      `json:"name,omitempty"`
	Bundle        string      `json:"bundle,omitempty"`
	Domain        string      `json:"domain,omitempty"`
	StoreURL      string      `json:"storeurl,omitempty"`
	Cat           []string    `json:"cat,omitempty"`
	SectionCat    []string    `json:"sectioncat,omitempty"`
	PageCat       []string    `json:"pagecat,omitempty"`
	Ver           string      `json:"ver,omitempty"`
	PrivacyPolicy int         `json:"privacypolicy,omitempty"`
	Paid          int         `json:"paid,omitempty"`
	Publisher     *Publisher  `json:"publisher,omitempty"`
	Content       *Content    `json:"content,omitempty"`
	Keywords      string      `json:"keywords,omitempty"`
	Ext           interface{} `json:"ext,omitempty"`
}

type Publisher struct {
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Cat    []string    `json:"cat,omitempty"`
	Domain string      `json:"domain,omitempty"`
	Ext    interface{} `json:"ext,omitempty"`
}

type Content struct {
	ID                 string      `json:"id,omitempty"`
	Episode            int         `json:"episode,omitempty"`
	Title              string      `json:"title,omitempty"`
	Series             string      `json:"series,omitempty"`
	Season             string      `json:"season,omitempty"`
	Artist             string      `json:"artist,omitempty"`
	Genre              string      `json:"genre,omitempty"`
	Album              string      `json:"album,omitempty"`
	ISRC               string      `json:"isrc,omitempty"`
	Producer           *Producer   `json:"producer,omitempty"`
	URL                string      `json:"url,omitempty"`
	Cat                []string    `json:"cat,omitempty"`
	ProdQ              int         `json:"prodq,omitempty"`
	VideoQuality       int         `json:"videoquality,omitempty"`
	Context            int         `json:"context,omitempty"`
	ContentRating      string      `json:"contentrating,omitempty"`
	UserRating         string      `json:"userrating,omitempty"`
	QAGMediaRating     int         `json:"qagmediarating,omitempty"`
	Keywords           string      `json:"keywords,omitempty"`
	LiveStream         int         `json:"livestream,omitempty"`
	SourceRelationship int         `json:"sourcerelationship,omitempty"`
	Len                int         `json:"len,omitempty"`
	Language           string      `json:"language,omitempty"`
	Embeddable         int         `json:"embeddable,omitempty"`
	Data               []Data      `json:"data,omitempty"`
	Ext                interface{} `json:"ext,omitempty"`
}

type Producer struct {
	ID     string      `json:"id,omitempty"`
	Name   string      `json:"name,omitempty"`
	Cat    []string    `json:"cat,omitempty"`
	Domain string      `json:"domain,omitempty"`
	Ext    interface{} `json:"ext,omitempty"`
}

type Device struct {
	UA             string      `json:"ua,omitempty"`
	Geo            *Geo        `json:"geo,omitempty"`
	DNT            int         `json:"dnt,omitempty"`
	Lmt            int         `json:"lmt,omitempty"`
	IP             string      `json:"ip,omitempty"`
	IPv6           string      `json:"ipv6,omitempty"`
	DeviceType     int         `json:"devicetype,omitempty"`
	Make           string      `json:"make,omitempty"`
	Model          string      `json:"model,omitempty"`
	OS             string      `json:"os,omitempty"`
	OSV            string      `json:"osv,omitempty"`
	HWV            string      `json:"hwv,omitempty"`
	H              int         `json:"h,omitempty"`
	W              int         `json:"w,omitempty"`
	PPI            int         `json:"ppi,omitempty"`
	PxRatio        float64     `json:"pxratio,omitempty"`
	JS             int         `json:"js,omitempty"`
	GeoFetch       int         `json:"geofetch,omitempty"`
	FlashVer       string      `json:"flashver,omitempty"`
	Language       string      `json:"language,omitempty"`
	Carrier        string      `json:"carrier,omitempty"`
	MCCMNC         string      `json:"mccmnc,omitempty"`
	ConnectionType int         `json:"connectiontype,omitempty"`
	IFA            string      `json:"ifa,omitempty"`
	DIDSHA1        string      `json:"didsha1,omitempty"`
	DIDMD5         string      `json:"didmd5,omitempty"`
	DPIDSHA1       string      `json:"dpidsha1,omitempty"`
	DPIDMD5        string      `json:"dpidmd5,omitempty"`
	MACSHA1        string      `json:"macsha1,omitempty"`
	MACMD5         string      `json:"macmd5,omitempty"`
	Ext            interface{} `json:"ext,omitempty"`
}

type Geo struct {
	Lat           float64     `json:"lat,omitempty"`
	Lon           float64     `json:"lon,omitempty"`
	Type          int         `json:"type,omitempty"`
	Accuracy      int         `json:"accuracy,omitempty"`
	LastFix       int         `json:"lastfix,omitempty"`
	IPService     int         `json:"ipservice,omitempty"`
	Country       string      `json:"country,omitempty"`
	Region        string      `json:"region,omitempty"`
	RegionFIPS104 string      `json:"regionfips104,omitempty"`
	Metro         string      `json:"metro,omitempty"`
	City          string      `json:"city,omitempty"`
	Zip           string      `json:"zip,omitempty"`
	UTCOffset     int         `json:"utcoffset,omitempty"`
	Ext           interface{} `json:"ext,omitempty"`
}

type User struct {
	ID         string      `json:"id,omitempty"`
	BuyerUID   string      `json:"buyeruid,omitempty"`
	Yob        int         `json:"yob,omitempty"`
	Gender     string      `json:"gender,omitempty"`
	Keywords   string      `json:"keywords,omitempty"`
	CustomData string      `json:"customdata,omitempty"`
	Geo        *Geo        `json:"geo,omitempty"`
	Data       []Data      `json:"data,omitempty"`
	Ext        interface{} `json:"ext,omitempty"`
}

type Data struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name,omitempty"`
	Segment []Segment   `json:"segment,omitempty"`
	Ext     interface{} `json:"ext,omitempty"`
}

type Segment struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Value string      `json:"value,omitempty"`
	Ext   interface{} `json:"ext,omitempty"`
}

type Regs struct {
	Coppa int         `json:"coppa,omitempty"`
	Ext   interface{} `json:"ext,omitempty"`
}
