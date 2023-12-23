package util

import (
	"math"
	"net"

	"golang.org/x/sys/unix"
)

// RoundToDecimal round iFloat to iDecimalPlaces decimal points
func RoundToDecimal(iFloat float64, iDecimalPlaces int) float64 {
	var multiplier float64 = 10
	for i := 1; i < iDecimalPlaces; i++ {
		multiplier *= 10
	}

	return math.Round(iFloat*multiplier) / multiplier
}

// UDPAddr -> RawSockaddr conversion
func NetAddrToSockaddr(addr *net.UDPAddr) unix.Sockaddr {
	if ip := addr.IP.To4(); ip != nil {
		return &unix.SockaddrInet4{
			Port: addr.Port,
			Addr: [4]byte(ip),
		}
	}

	if ip := addr.IP.To16(); ip != nil {
		return &unix.SockaddrInet6{
			Addr: [16]byte(ip),
			Port: addr.Port,
		}
	}

	return nil
}

// RawSockaddr -> UDPAddr conversion
func SockaddrToUDPAddr(sockAddr unix.Sockaddr) *net.UDPAddr {
	switch typedSocketAddr := sockAddr.(type) {

	case *unix.SockaddrInet4:
		return &net.UDPAddr{
			IP:   typedSocketAddr.Addr[:],
			Port: typedSocketAddr.Port,
			Zone: "",
		}

	case *unix.SockaddrInet6:
		if !IsValidIPv6(string(typedSocketAddr.Addr[:])) {
			return nil
		}

		return &net.UDPAddr{
			IP:   typedSocketAddr.Addr[:],
			Port: typedSocketAddr.Port,
			Zone: getZone(typedSocketAddr.ZoneId),
		}
	default:
		return nil
	}
}

func IsValidIPv6(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To4() != nil {
		return false
	}
	return true
}

func getZone(zoneID uint32) string {
	var zone = ""
	ifi, err := net.InterfaceByIndex(int(zoneID))
	if err == nil && zoneID != 0 {
		zone = ifi.Name
	}
	return zone
}
