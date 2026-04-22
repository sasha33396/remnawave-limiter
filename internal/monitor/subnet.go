package monitor

import "net"

func subnetPrefix(ipStr string, v4 int) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ipStr
	}
	if ip4 := ip.To4(); ip4 != nil {
		mask := net.CIDRMask(v4, 32)
		network := ip4.Mask(mask)
		return fmtCIDR(network, v4)
	}
	return ipStr
}

func fmtCIDR(network net.IP, prefix int) string {
	n := &net.IPNet{IP: network, Mask: net.CIDRMask(prefix, len(network)*8)}
	return n.String()
}
