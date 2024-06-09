package p2p

type Handshaker interface {
	Handshake() error
}

type HandshakeFunc func(Peer) error

type DefaultHandshaker struct{}

func NOHandshakeFunc(Peer) error { return nil }
