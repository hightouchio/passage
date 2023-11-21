package tunnel

import (
	"net"
	"strconv"
)

// newEphemeralTCPListener returns a TCP listener bound to a random, unused port
func newEphemeralTCPListener() (*net.TCPListener, error) {
	return net.ListenTCP("tcp", &net.TCPAddr{Port: 0})
}

func portFromNetAddr(addr net.Addr) int {
	_, portStr, _ := net.SplitHostPort(addr.String())
	port, _ := strconv.Atoi(portStr)
	return port
}
