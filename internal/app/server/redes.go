package server

import (
	"net"
	"strings"
)

func remoteIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}

func ipAllowed(ip net.IP, allowed []*net.IPNet) bool {
	for _, network := range allowed {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func redesExclusivamenteLocales(redes []*net.IPNet) bool {
	if len(redes) == 0 {
		return false
	}
	for _, red := range redes {
		if red == nil {
			return false
		}
		unos, bits := red.Mask.Size()
		ip := red.IP
		if ipv4 := ip.To4(); ipv4 != nil {
			if bits != net.IPv4len*8 || unos < 8 || ipv4[0] != 127 {
				return false
			}
			continue
		}
		if bits != net.IPv6len*8 || unos != net.IPv6len*8 || !ip.Equal(net.IPv6loopback) {
			return false
		}
	}
	return true
}

func direccionEscuchaLoopback(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil || strings.TrimSpace(host) == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(strings.TrimSpace(host), "[]"))
	return ip != nil && ip.IsLoopback()
}

func direccionEscuchaLocalPresentacion(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil || strings.TrimSpace(host) == "" {
		return false
	}
	ip := net.ParseIP(strings.Trim(strings.TrimSpace(host), "[]"))
	return ip != nil && !ip.IsUnspecified() &&
		(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func redesExclusivamenteLocalesPresentacion(redes []*net.IPNet) bool {
	if len(redes) == 0 {
		return false
	}
	permitidas := []*net.IPNet{
		debeParsearCIDR("127.0.0.0/8"),
		debeParsearCIDR("10.0.0.0/8"),
		debeParsearCIDR("172.16.0.0/12"),
		debeParsearCIDR("192.168.0.0/16"),
		debeParsearCIDR("169.254.0.0/16"),
		debeParsearCIDR("::1/128"),
		debeParsearCIDR("fc00::/7"),
		debeParsearCIDR("fe80::/10"),
	}
	for _, red := range redes {
		if red == nil || !redContenidaEnAlguna(red, permitidas) {
			return false
		}
	}
	return true
}

func redContenidaEnAlguna(red *net.IPNet, permitidas []*net.IPNet) bool {
	unos, bits := red.Mask.Size()
	if unos < 0 {
		return false
	}
	for _, permitida := range permitidas {
		unosPermitidos, bitsPermitidos := permitida.Mask.Size()
		if bits == bitsPermitidos && unos >= unosPermitidos && permitida.Contains(red.IP) {
			return true
		}
	}
	return false
}

func debeParsearCIDR(valor string) *net.IPNet {
	_, red, err := net.ParseCIDR(valor)
	if err != nil {
		panic(err)
	}
	return red
}
