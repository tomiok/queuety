package manager

import "net"

func Connect(protocol, addr string) {
	net.Dial(protocol, addr)
}
