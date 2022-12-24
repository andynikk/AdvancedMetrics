package networks

import (
	"net"

	"github.com/andynikk/advancedmetrics/internal/constants"
)

func AddressAllowed(IPs []string) bool {
	_, ipv4Net, _ := net.ParseCIDR("192.168.1.0/24")

	for _, sIP := range IPs {
		ip := net.ParseIP(sIP)

		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return false
		}

		if ipv4Net.Contains(ip) {
			return true
		}
	}

	return false
}

func IPStr(IPs []net.IP) string {

	strIP := ""
	for _, ip := range IPs {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			continue
		}

		strIP += ip.String() + constants.SepIPAddress
	}

	return strIP
}
