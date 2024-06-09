package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listnerAddr := ":4000"
	tcpOpts := TCPTransportOps{
		ListnerAddr:   listnerAddr,
		HandshakeFunc: NOHandshakeFunc,
	}
	tr := NewTCPTransport(tcpOpts)
	assert.Equal(t, tr.TCPTransportOps.ListnerAddr, listnerAddr)

	assert.Nil(t, tr.ListenAndAccept())
}
