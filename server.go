package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/keshav025/nexdrive/p2p"
)

type FileServerOpts struct {
	EncKey            []byte
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

type MessageGeteFile struct {
	Key string
	// Size int64
	// Data []byte
}

func (s *FileServer) Remove(key string) error {
	return nil
}

func (s *FileServer) Get(key string) (int64, io.Reader, error) {
	if s.store.Has(key) {
		fmt.Printf("file (%s)  found locally\n", key)
		return s.store.Read(key)
	}

	msg := Message{
		Payload: MessageGeteFile{
			Key: key,
		},
	}
	fmt.Printf("file (%s) cannot be found locally with key, checking in network\n", key)
	if err := s.broadcast(&msg); err != nil {
		return 0, nil, err
	}
	//time.Sleep(time.Second * 5)
	for _, peer := range s.peers {
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		//copy file sent from peer to local disk for a given key
		n, err := s.store.WriteDecrypt(s.EncKey, key, io.LimitReader(peer, fileSize))
		//n, err := s.store.Write(key, io.LimitReader(peer, fileSize))
		if err != nil {
			return 0, nil, err
		}
		fmt.Printf("[%s] received (%d) from netwok peer: %s \n", s.Transport.Addr(), n, peer.RemoteAddr())
		peer.CloseStream()
		// filebuffer := new(bytes.Buffer)
		// n, err := io.Copy(filebuffer, peer)
		// if err != nil {
		// 	return nil, err
		// }
		// fmt.Println("received bytes fover the network: ", n)
		// fmt.Println("got file: ", filebuffer)

	}

	return s.store.Read(key)
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
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
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil
	}

	time.Sleep(time.Millisecond * 5)
	// payload := []byte("this large file")
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncKey, filebuffer, mw)
	// n, err := io.Copy(peer, filebuffer)
	if err != nil {
		return err
	}
	fmt.Println("written to the disk: ", n)
	// for _, peer := range s.peers {
	// 	peer.Send([]byte{p2p.IncomingStream})
	// 	n, err := copyEncrypt(s.EncKey, filebuffer, peer)
	// 	// n, err := io.Copy(peer, filebuffer)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	fmt.Println("written to the disk: ", n)
	// }

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
		fmt.Println("file server stopped to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():

			var msg Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				fmt.Printf("decoding error: %v\n", err)
				continue

			}
			fmt.Printf("recv: %+v\n", msg.Payload)

			if err := s.handleMessage(rpc.From, &msg); err != nil {

				fmt.Printf("handle message error: %+v\n", err)

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
	case MessageGeteFile:
		return s.handleMessageGetFile(from, &v)

	}
	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg *MessageGeteFile) error {
	fmt.Println("handling get file: ", from, msg)
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("file (%s) doesnot exist on disk", msg.Key)

	}
	fmt.Printf("got file: %s, serving over networn\n", msg)
	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok {
		fmt.Println("closing read closer")
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer map", from)
	}
	peer.Send([]byte{p2p.IncomingStream})

	err = binary.Write(peer, binary.LittleEndian, fileSize)
	if err != nil {
		return err
	}
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	fmt.Printf("written (%d) byted over the network to %v\n", n, from)

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
	peer.CloseStream()
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

	gob.Register(MessageGeteFile{})
}
