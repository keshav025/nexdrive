package p2p

// Peer represents remote node
type Peer interface {
	Close() error
}

// transport is for communation between nodes in the network
// can be tcp, udp, websocket etc.
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
