package p2p

import "net"

// Peer represents remote node
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// transport is for communation between nodes in the network
// can be tcp, udp, websocket etc.
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
