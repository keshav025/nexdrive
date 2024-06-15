package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/keshav025/nexdrive/p2p"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOps{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Message struct {
	// From    string
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
	Data []byte
}

type DataMessage struct {
	Key  string
	Data []byte
}

func (s *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	//1. store this file to disk
	//2. broadcast this file to all known peers in the network

	filebuffer := new(bytes.Buffer)
	tee := io.TeeReader(r, filebuffer)

	size, err := s.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	msgbuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgbuf).Encode(msg); err != nil {
		return err
	}
	for _, peer := range s.peers {
		if err := peer.Send(msgbuf.Bytes()); err != nil {
			return err
		}
	}

	time.Sleep(time.Second * 5)
	// payload := []byte("this large file")
	for _, peer := range s.peers {
		n, err := io.Copy(peer, filebuffer)
		if err != nil {
			return err
		}
		fmt.Println("written to the disk: ", n)
	}

	return nil
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)
	// if err := s.store.Write(key, tee); err != nil {
	// 	return err
	// }

	// _, err := io.Copy(buf, r)

	// if err != nil {
	// 	return err
	// }
	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }
	// fmt.Println(buf.Bytes())

	// return s.broadcast(&Message{
	// 	From:    "todo",
	// 	Payload: p,
	// })

}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p
	fmt.Printf("connected with remote %s\n", p.RemoteAddr())
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():

			var msg Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				continue
			}
			fmt.Printf("recv: %+v\n", msg.Payload)

			if err := s.handleMessage(rpc.From, &msg); err != nil {

				fmt.Printf("error: %+v\n", err)
				return
			}
			// peer, ok := s.peers[rpc.From]
			// if !ok {
			// 	panic("peer  not found")
			// }

			// b := make([]byte, 1000)
			// if _, err := peer.Read(b); err != nil {
			// 	panic(err)
			// }
			// fmt.Printf("recvd: %s\n", string(b))
			// peer.(*p2p.TCPPeer).Wg.Done()
			// mm := Message{Payload: m}
			// if err := s.handleMessage(&mm); err != nil {
			// 	continue
			// }
			// fmt.Println(string(p.Data))
		case <-s.quitch:
			return
		}
	}

}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) Start() error {

	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	s.loop()
	return nil
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, &v)

	}
	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg *MessageStoreFile) error {
	fmt.Printf("received message %+v\n", msg)
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}

	if _, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size)); err != nil {
		return nil
	}
	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			if err := s.Transport.Dial(addr); err != nil {
				fmt.Printf("dial err for addr: %s", addr)
			}
		}(addr)

	}
	return nil
}

// func (s *FileServer) Store(key string, r io.Reader) error {

// 	return s.store.Write(key, r)
// }

func init() {
	gob.Register(MessageStoreFile{})
}
