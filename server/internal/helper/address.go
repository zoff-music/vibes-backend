package helper

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP selects the rightmost valid proxy-appended address, as used by rate limiting.
func ClientIP(r *http.Request) string {
	forwardedAddresses := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(forwardedAddresses) - 1; i >= 0; i-- {
		parsed, err := netip.ParseAddr(strings.TrimSpace(forwardedAddresses[i]))
		if err == nil {
			address := parsed.Unmap().String()
			return address
		}
	}
	parsed, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-Ip")))
	if err == nil {
		address := parsed.Unmap().String()
		return address
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		parsed, err = netip.ParseAddr(host)
		if err == nil {
			address := parsed.Unmap().String()
			return address
		}
	}
	parsed, err = netip.ParseAddr(r.RemoteAddr)
	if err == nil {
		address := parsed.Unmap().String()
		return address
	}
	return "unknown-ip"
}
