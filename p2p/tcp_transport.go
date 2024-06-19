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
	wg       *sync.WaitGroup
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	return err
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
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
		wg:       &sync.WaitGroup{},
	}
}

func NewTCPTransport(opts TCPTransportOps) *TCPTransport {
	return &TCPTransport{
		TCPTransportOps: opts,
		rpcch:           make(chan RPC, 1024),
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

func (t *TCPTransport) Addr() string {
	return t.ListnerAddr
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

	// buf := make([]byte, 2000)
	for {
		rpc := RPC{}
		// n, err := conn.Read(buf)
		// if err != nil {
		// 	fmt.Printf("TCP error: %s\n", err)
		// }

		err = t.Decoder.Decode(conn, &rpc)
		if err != nil {
			return
		}
		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr().String())
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed resuming read loop ...\n", conn.RemoteAddr().String())
			continue
		}

		t.rpcch <- rpc

		// fmt.Printf("message: %+v\n", rpc)
	}

}

// func Tset() {
// 	t := NewTCPTransport(":4344")
// }
