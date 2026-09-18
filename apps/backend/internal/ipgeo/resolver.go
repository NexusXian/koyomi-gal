package ipgeo

import "net"

type Resolver interface {
	Resolve(ip net.IP) (Location, error)
	Close() error
}
