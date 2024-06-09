package p2p

import "net"

// rpc holds any arbitrary data that is being sent over each transport between nodes in the network
type RPC struct {
	From    net.Addr
	Payload []byte
}
