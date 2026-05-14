package main

import (
	"fmt"
	"math/rand"
	"net"

	maxminddb "github.com/oschwald/maxminddb-golang"
)

// IPIndex holds a country-keyed map of IPv4 CIDR ranges built from a MaxMind
// MMDB file. It is used to generate device IP addresses that are realistic for
// a given geo country.
type IPIndex struct {
	ranges map[string][]*net.IPNet // ISO 3166-1 alpha-2 → IPv4 networks
}

// BuildIPIndex opens the MMDB at mmdbPath, iterates every IPv4 network record,
// and indexes the CIDR ranges by ISO country code. The MMDB file is closed
// before returning; the caller owns the returned *IPIndex.
func BuildIPIndex(mmdbPath string) (*IPIndex, error) {
	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var record struct {
		Country struct {
			IsoCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}

	idx := &IPIndex{ranges: make(map[string][]*net.IPNet)}
	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		network, err := networks.Network(&record)
		if err != nil || record.Country.IsoCode == "" {
			continue
		}
		if network.IP.To4() == nil { // IPv4 only
			continue
		}
		idx.ranges[record.Country.IsoCode] = append(
			idx.ranges[record.Country.IsoCode], network)
	}
	if err := networks.Err(); err != nil {
		return nil, err
	}
	return idx, nil
}

// CountryCount returns the number of countries in the index.
func (idx *IPIndex) CountryCount() int {
	if idx == nil {
		return 0
	}
	return len(idx.ranges)
}

// RandomIP returns a random IPv4 address from a CIDR range belonging to the
// given ISO country code. Falls back to a fully random IPv4 when the index is
// nil, the country is empty, or no ranges are found for that country.
func (idx *IPIndex) RandomIP(country string) string {
	if idx != nil && country != "" {
		if ranges := idx.ranges[country]; len(ranges) > 0 {
			if ip := randomIPInCIDR(ranges[rand.Intn(len(ranges))]); ip != "" {
				return ip
			}
		}
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		randomInt(1, 255), randomInt(1, 255), randomInt(1, 255), randomInt(1, 255))
}

func randomIPInCIDR(network *net.IPNet) string {
	ip4 := network.IP.To4()
	if ip4 == nil {
		return ""
	}
	ones, bits := network.Mask.Size()
	hostBits := bits - ones
	base := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])
	if hostBits <= 1 {
		return fmt.Sprintf("%d.%d.%d.%d", ip4[0], ip4[1], ip4[2], ip4[3])
	}
	hostCount := uint32(1) << hostBits
	offset := uint32(rand.Intn(int(hostCount-2))) + 1 // skip network address and broadcast
	addr := base + offset
	return fmt.Sprintf("%d.%d.%d.%d", addr>>24, (addr>>16)&0xff, (addr>>8)&0xff, addr&0xff)
}
