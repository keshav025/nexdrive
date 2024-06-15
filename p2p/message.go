package p2p

// rpc holds any arbitrary data that is being sent over each transport between nodes in the network
type RPC struct {
	//From    net.Addr
	From    string
	Payload []byte
}
