package main

import (
	"log"

	"github.com/keshav025/nexdrive/p2p"
)

func main() {

	tcpOpts := p2p.TCPTransportOps{
		ListnerAddr:   ":3000",
		HandshakeFunc: p2p.NOHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	// go func() {
	// 	for {
	// 		msg := <-tr.Consume()
	// 		fmt.Printf("message: %v\n", msg)

	// 	}
	// }()

	fileServerOpts := FileServerOpts{
		ListenAddr:        ":3000",
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tr,
	}

	s := NewFileServer(fileServerOpts)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
	select {}
}
