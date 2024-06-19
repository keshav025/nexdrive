package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// rpc holds any arbitrary data that is being sent over each transport between nodes in the network
type RPC struct {
	//From    net.Addr
	From    string
	Payload []byte
	Stream  bool
}
