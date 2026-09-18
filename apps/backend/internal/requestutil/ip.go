package requestutil

import (
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
)

func ClientIP(c *gin.Context) string {
	address, err := netip.ParseAddr(strings.TrimSpace(c.ClientIP()))
	if err != nil {
		return ""
	}
	address = address.WithZone("")
	return address.Unmap().String()
}
