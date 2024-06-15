package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/keshav025/nexdrive/p2p"
)

func makeServer(listnerAddr string, nodes ...string) *FileServer {
	tcpOpts := p2p.TCPTransportOps{
		ListnerAddr:   listnerAddr,
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
		ListenAddr:        listnerAddr,
		StorageRoot:       listnerAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tr,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(fileServerOpts)
	tr.OnPeer = s.OnPeer

	return s
}

func main() {
	// Create a channel to receive signals.
	c := make(chan os.Signal)

	// Register the SIGINT signal to be sent to the channel.
	signal.Notify(c, os.Interrupt)
	s := makeServer(":3000")
	go func() {
		if err := s.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	s1 := makeServer(":4000", ":3000")
	go func() {
		if err := s1.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(5 * time.Second)

	data := bytes.NewReader([]byte("my big data file here!"))
	s1.StoreData("myprovatedata", data)
	select {
	case <-c:
		fmt.Println("closing the servers")
		s.Stop()
		s1.Stop()
	}
}
