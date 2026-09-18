package ipgeo

import (
	"fmt"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

type MaxMindResolver struct {
	city *geoip2.Reader
	asn  *geoip2.Reader
}

func NewMaxMindResolver(cityPath, asnPath string) (*MaxMindResolver, error) {
	city, err := geoip2.Open(cityPath)
	if err != nil {
		return nil, fmt.Errorf("open GeoIP city database: %w", err)
	}
	resolver := &MaxMindResolver{city: city}
	if strings.TrimSpace(asnPath) != "" {
		resolver.asn, err = geoip2.Open(asnPath)
		if err != nil {
			_ = city.Close()
			return nil, fmt.Errorf("open GeoIP ASN database: %w", err)
		}
	}
	return resolver, nil
}

func (r *MaxMindResolver) Resolve(ip net.IP) (Location, error) {
	record, err := r.city.City(ip)
	if err != nil {
		return Location{}, fmt.Errorf("resolve GeoIP city: %w", err)
	}
	location := Location{
		Country: localizedName(record.Country.Names),
		City:    localizedName(record.City.Names),
	}
	if len(record.Subdivisions) > 0 {
		location.Region = localizedName(record.Subdivisions[0].Names)
	}
	location.Country = normalizeCountry(record.Country.IsoCode, location.Country)
	location.Region = normalizeChineseRegion(location.Region)

	if r.asn != nil {
		asn, asnErr := r.asn.ASN(ip)
		if asnErr != nil {
			return Location{}, fmt.Errorf("resolve GeoIP ASN: %w", asnErr)
		}
		location.ISP = strings.TrimSpace(asn.AutonomousSystemOrganization)
	}
	return location, nil
}

func (r *MaxMindResolver) Close() error {
	var firstErr error
	if r.city != nil {
		firstErr = r.city.Close()
	}
	if r.asn != nil {
		if err := r.asn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *MaxMindResolver) CacheVersion() string {
	return fmt.Sprintf("%d", r.city.Metadata().BuildEpoch)
}

func localizedName(names map[string]string) string {
	for _, language := range []string{"zh-CN", "zh", "en"} {
		if value := strings.TrimSpace(names[language]); value != "" {
			return value
		}
	}
	return ""
}

func normalizeCountry(code, name string) string {
	switch strings.ToUpper(code) {
	case "CN":
		return "中国"
	case "HK":
		return "中国香港"
	case "MO":
		return "中国澳门"
	case "TW":
		return "中国台湾"
	default:
		return strings.TrimSpace(name)
	}
}

func normalizeChineseRegion(region string) string {
	region = strings.TrimSpace(region)
	for _, suffix := range []string{"壮族自治区", "回族自治区", "维吾尔自治区", "自治区", "特别行政区", "省", "市"} {
		if strings.HasSuffix(region, suffix) {
			return strings.TrimSuffix(region, suffix)
		}
	}
	return region
}
