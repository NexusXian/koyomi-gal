package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// ErrBlockedURL reports a request that failed the SSRF guard.
var ErrBlockedURL = errors.New("url blocked by security policy")

var blockedNetworks = []*net.IPNet{
	// IPv4 private and link-local ranges.
	mustCIDR("127.0.0.0/8"),
	mustCIDR("10.0.0.0/8"),
	mustCIDR("172.16.0.0/12"),
	mustCIDR("192.168.0.0/16"),
	mustCIDR("169.254.0.0/16"),
	mustCIDR("0.0.0.0/8"),
	mustCIDR("100.64.0.0/10"),
	// IPv6 loopback, link-local and unique-local ranges.
	mustCIDR("::1/128"),
	mustCIDR("::/128"),
	mustCIDR("fe80::/10"),
	mustCIDR("fc00::/7"),
}

func mustCIDR(value string) *net.IPNet {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		panic(err)
	}
	return network
}

// validateFetchURL enforces scheme and hostname shape; caller then resolves.
func validateFetchURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: scheme %q is not http/https", ErrBlockedURL, parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("%w: empty host", ErrBlockedURL)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%w: userinfo is not allowed", ErrBlockedURL)
	}
	return parsed, nil
}

// resolveAndCheck resolves a hostname and rejects it when any address falls
// into a private, loopback or link-local network.
func resolveAndCheck(ctx context.Context, host string) error {
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("resolve %s: no addresses", host)
	}
	for _, address := range addresses {
		if blockedIP(address.IP) {
			return fmt.Errorf("%w: %s resolves to %s", ErrBlockedURL, host, address.IP)
		}
	}
	return nil
}

func blockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	for _, network := range blockedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// guardedDialer is an http.RoundTripper whose DialContext re-validates every
// resolved IP right before connecting, so DNS rebinding after validation does
// not bypass the SSRF guard.
type guardedDialer struct {
	base net.Dialer
}

func (d *guardedDialer) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if ip := net.ParseIP(host); ip != nil {
		if blockedIP(ip) {
			return nil, fmt.Errorf("%w: %s", ErrBlockedURL, ip)
		}
		return d.base.DialContext(ctx, network, address)
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var dialErr error
	for _, resolved := range ips {
		if blockedIP(resolved.IP) {
			continue
		}
		conn, err := d.base.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), mustPort(address)))
		if err == nil {
			return conn, nil
		}
		dialErr = err
	}
	if dialErr != nil {
		return nil, dialErr
	}
	return nil, fmt.Errorf("%w: %s has no public address", ErrBlockedURL, host)
}

func mustPort(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return ""
	}
	return port
}

// newGuardedHTTPClient builds a client that only reaches public http/https
// hosts. CheckRedirect validates each hop before it is followed.
func newGuardedHTTPClient(timeoutSeconds int) *http.Client {
	dialer := &guardedDialer{}
	return &http.Client{
		Timeout: 0, // overall deadline comes from the request context
		Transport: &http.Transport{
			DialContext: dialer.dialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			if _, err := validateFetchURL(req.URL.String()); err != nil {
				return err
			}
			return resolveAndCheck(req.Context(), req.URL.Hostname())
		},
	}
}
