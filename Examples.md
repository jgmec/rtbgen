# Example RTB 2.5 Bid Request Output

This file shows examples of what the RTB generator produces.

## Example 1: Banner Ad for Website

```json
{
  "id": "1706990567123456789-42857",
  "imp": [
    {
      "id": "1706990567123456790-67234",
      "banner": {
        "w": 728,
        "h": 90,
        "format": [
          {
            "w": 728,
            "h": 90
          }
        ],
        "pos": 3,
        "api": [3, 5]
      },
      "tagid": "tag-456",
      "bidfloor": 2.75,
      "bidfloorcur": "USD",
      "secure": 1
    }
  ],
  "site": {
    "id": "1706990567123456791-12345",
    "name": "Sample Site",
    "domain": "news-site.com",
    "cat": ["IAB12"],
    "page": "https://news-site.com/page-789",
    "publisher": {
      "id": "1706990567123456792-98765",
      "name": "Sample Publisher",
      "domain": "news-site.com"
    },
    "privacypolicy": 1,
    "mobile": 0
  },
  "device": {
    "ua": "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36",
    "geo": {
      "lat": 40.7128,
      "lon": -74.0060,
      "country": "USA",
      "region": "NY",
      "city": "New York",
      "zip": "10001",
      "type": 2
    },
    "dnt": 0,
    "lmt": 0,
    "ip": "192.168.15.42",
    "devicetype": 4,
    "make": "Samsung",
    "model": "Model-8",
    "os": "Android",
    "osv": "11.0",
    "language": "en",
    "carrier": "T-Mobile",
    "connectiontype": 2,
    "ifa": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "js": 1,
    "w": 1920,
    "h": 1080
  },
  "user": {
    "id": "1706990567123456793-54321",
    "buyeruid": "1706990567123456794-11111",
    "yob": 1987,
    "gender": "M",
    "keywords": "sports,technology"
  },
  "regs": {
    "coppa": 0
  },
  "at": 1,
  "tmax": 250
}
```

## Example 2: Video Ad for Mobile App

```json
{
  "id": "1706990567234567890-85219",
  "imp": [
    {
      "id": "1706990567234567891-42186",
      "video": {
        "mimes": ["video/mp4", "video/x-flv"],
        "minduration": 5,
        "maxduration": 30,
        "protocols": [2, 3, 5, 6],
        "w": 640,
        "h": 480,
        "startdelay": 0,
        "placement": 1,
        "linearity": 1,
        "playbackmethod": [1, 3],
        "api": [1, 2]
      },
      "tagid": "tag-789",
      "bidfloor": 4.25,
      "bidfloorcur": "USD",
      "secure": 1
    }
  ],
  "app": {
    "id": "1706990567234567892-33333",
    "name": "Sample App",
    "bundle": "com.game.fun",
    "domain": "example.com",
    "cat": ["IAB9"],
    "storeurl": "https://play.google.com/store/apps/details?id=com.game.fun",
    "ver": "1.0.0",
    "publisher": {
      "id": "1706990567234567893-44444",
      "name": "App Publisher"
    },
    "privacypolicy": 1
  },
  "device": {
    "ua": "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36",
    "geo": {
      "lat": 34.0522,
      "lon": -118.2437,
      "country": "USA",
      "region": "CA",
      "city": "Los Angeles",
      "zip": "90012",
      "type": 1
    },
    "dnt": 1,
    "lmt": 0,
    "ip": "10.0.0.142",
    "devicetype": 1,
    "make": "Apple",
    "model": "Model-12",
    "os": "iOS",
    "osv": "14.4",
    "language": "en",
    "carrier": "Verizon",
    "connectiontype": 3,
    "ifa": "f9e8d7c6-b5a4-3210-9876-543210fedcba",
    "js": 1,
    "w": 375,
    "h": 812
  },
  "user": {
    "id": "1706990567234567894-77777",
    "buyeruid": "1706990567234567895-88888",
    "yob": 1992,
    "gender": "F",
    "keywords": "gaming,entertainment"
  },
  "regs": {
    "coppa": 0
  },
  "at": 2,
  "tmax": 300
}
```

## Example 3: Multiple Impressions

The generator can also create requests with multiple impressions:

```json
{
  "id": "1706990567345678901-19283",
  "imp": [
    {
      "id": "1706990567345678902-11111",
      "banner": {
        "w": 300,
        "h": 250,
        "format": [
          {
            "w": 300,
            "h": 250
          }
        ],
        "pos": 1,
        "api": [3, 5]
      },
      "tagid": "tag-123",
      "bidfloor": 1.50,
      "bidfloorcur": "USD",
      "secure": 1
    },
    {
      "id": "1706990567345678903-22222",
      "banner": {
        "w": 728,
        "h": 90,
        "format": [
          {
            "w": 728,
            "h": 90
          }
        ],
        "pos": 0,
        "api": [3, 5]
      },
      "tagid": "tag-124",
      "bidfloor": 2.00,
      "bidfloorcur": "USD",
      "secure": 1
    }
  ],
  "site": {
    "id": "1706990567345678904-33333",
    "name": "Sample Site",
    "domain": "blog.net",
    "cat": ["IAB1"],
    "page": "https://blog.net/page-456",
    "publisher": {
      "id": "1706990567345678905-44444",
      "name": "Sample Publisher",
      "domain": "blog.net"
    },
    "privacypolicy": 1,
    "mobile": 1
  },
  "device": {
    "ua": "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36",
    "geo": {
      "lat": 41.8781,
      "lon": -87.6298,
      "country": "USA",
      "region": "IL",
      "city": "Chicago",
      "zip": "60601",
      "type": 2
    },
    "dnt": 0,
    "lmt": 1,
    "ip": "172.16.254.1",
    "devicetype": 5,
    "make": "Google",
    "model": "Model-5",
    "os": "Android",
    "osv": "12.0",
    "language": "es",
    "carrier": "AT&T",
    "connectiontype": 6,
    "ifa": "12345678-90ab-cdef-1234-567890abcdef",
    "js": 1,
    "w": 414,
    "h": 812
  },
  "user": {
    "id": "1706990567345678906-55555",
    "buyeruid": "1706990567345678907-66666",
    "yob": 1985,
    "gender": "M",
    "keywords": "travel,food"
  },
  "regs": {
    "coppa": 0
  },
  "at": 1,
  "tmax": 200
}
```

## Key Features Demonstrated

1. **Request IDs**: Unique timestamp-based IDs for tracking
2. **Impression Types**: Banner and video ads with proper specifications
3. **Site vs App**: Different context types for web and mobile
4. **Device Information**: Realistic device specs, OS, carrier, etc.
5. **Geolocation**: Country, region, city with lat/lon coordinates
6. **User Data**: Demographics including age, gender, interests
7. **Privacy Flags**: DNT (Do Not Track), LMT (Limit Ad Tracking), COPPA
8. **Auction Type**: First price (1) or second price (2) auctions
9. **Bid Floor**: Minimum bid prices in USD
10. **Publisher Info**: Publisher identification and categorization