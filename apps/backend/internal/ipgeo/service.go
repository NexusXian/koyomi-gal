package ipgeo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalidIP = errors.New("invalid IP address")

type Service struct {
	enabled      bool
	displayLevel string
	resolver     Resolver
	redis        *redis.Client
	cacheTTL     time.Duration
	cacheVersion string
}

func NewService(enabled bool, displayLevel string, resolver Resolver, redisClient *redis.Client, cacheTTL time.Duration) *Service {
	service := &Service{
		enabled: enabled, displayLevel: displayLevel, resolver: resolver,
		redis: redisClient, cacheTTL: cacheTTL, cacheVersion: "unknown",
	}
	if versioned, ok := resolver.(interface{ CacheVersion() string }); ok {
		service.cacheVersion = versioned.CacheVersion()
	}
	return service
}

func (s *Service) Resolve(ctx context.Context, value string) (string, Location, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return "", Location{}, ErrInvalidIP
	}
	address = address.WithZone("").Unmap()
	normalized := address.String()
	if !s.enabled || s.resolver == nil || !isPublicAddress(address) {
		return normalized, Location{}, nil
	}

	key := cacheKey(normalized, s.cacheVersion)
	if s.redis != nil {
		if encoded, cacheErr := s.redis.Get(ctx, key).Bytes(); cacheErr == nil {
			var location Location
			if json.Unmarshal(encoded, &location) == nil {
				location.DisplayRegion = displayRegion(location, s.displayLevel)
				return normalized, location, nil
			}
		}
	}

	location, err := s.resolver.Resolve(net.IP(address.AsSlice()))
	if err != nil {
		return normalized, Location{}, err
	}
	location.DisplayRegion = displayRegion(location, s.displayLevel)
	if s.redis != nil {
		if encoded, encodeErr := json.Marshal(location); encodeErr == nil {
			_ = s.redis.Set(ctx, key, encoded, s.cacheTTL).Err()
		}
	}
	return normalized, location, nil
}

func displayRegion(location Location, level string) string {
	switch level {
	case "country":
		return location.Country
	case "city":
		parts := make([]string, 0, 2)
		if location.Region != "" && location.Region != location.Country {
			parts = append(parts, location.Region)
		}
		if location.City != "" && location.City != location.Region {
			parts = append(parts, location.City)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return location.Country
	default:
		if location.Country == "中国" && location.Region != "" {
			return location.Region
		}
		return location.Country
	}
}

func isPublicAddress(address netip.Addr) bool {
	return address.IsGlobalUnicast() && !address.IsPrivate() && !address.IsLoopback() && !address.IsLinkLocalUnicast()
}

func cacheKey(ip, version string) string {
	digest := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("koyomi:ipgeo:v1:%s:%s", version, hex.EncodeToString(digest[:]))
}
