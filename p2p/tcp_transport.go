package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// represents tcp connection with remote node

type TCPPeer struct {
	//connection with peer
	net.Conn

	// dial we dial and retrieve conn: outbound =true
	// dial we accept and retrieve conn: outbound =false
	outbound bool
	Wg       *sync.WaitGroup
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

type TCPTransportOps struct {
	ListnerAddr   string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOps

	listner net.Listener
	rpcch   chan RPC

	mu sync.RWMutex
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func NewTCPTransport(opts TCPTransportOps) *TCPTransport {
	return &TCPTransport{
		TCPTransportOps: opts,
		rpcch:           make(chan RPC),
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listner, err = net.Listen("tcp", t.ListnerAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	fmt.Printf("TCP transport is listening at port: %v\n", t.ListnerAddr)

	return nil
}

// return read only channel for reading incoming messages received from another peer
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	return t.listner.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil

}

func (t *TCPTransport) startAcceptLoop() {
	for {

		conn, err := t.listner.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP accept error: %s\n", err)
			return
		}
		//fmt.Printf("new incoming connection %+v\n", conn)
		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error

	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
	}()

	peer := NewTCPPeer(conn, outbound)

	if err := t.HandshakeFunc(peer); err != nil {
		conn.Close()
		fmt.Printf("TCP handshake error: %s\n", err)
		return
	}

	if t.OnPeer != nil {
		if err := t.OnPeer(peer); err != nil {

			return
		}
	}
	rpc := RPC{}
	// buf := make([]byte, 2000)
	for {

		// n, err := conn.Read(buf)
		// if err != nil {
		// 	fmt.Printf("TCP error: %s\n", err)
		// }

		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}
		rpc.From = conn.RemoteAddr().String()
		peer.Wg.Add(1)
		fmt.Println("waiting till stream is done")
		t.rpcch <- rpc
		peer.Wg.Wait()
		fmt.Println("stream done.. continuing normal loop")
		// fmt.Printf("message: %+v\n", rpc)
	}

}

// func Tset() {
// 	t := NewTCPTransport(":4344")
// }
