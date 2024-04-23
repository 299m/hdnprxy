package proxy

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestBasicWriteAndRead(t *testing.T) {
	// Create a new tunnel message
	tm := NewTunnelMessage(1024)
	// Create a new UDP address
	addr, _ := net.ResolveUDPAddr("udp", ":1234")
	// create some random data
	data := []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	// put the message onto the tunnel message
	fullmsg := tm.Write(data, addr)
	// read the message from the tunnel message
	msgdata, needmore, readdr, err := tm.Read(fullmsg)
	assert.Equal(t, false, needmore, "Need more isn't false")
	assert.Equal(t, nil, err, "Error isn't nil")
	assert.Equal(t, addr.String(), readdr.String(), "Address isn't the same")
	assert.Equal(t, data, msgdata, "Data isn't the same")
}

func TestPartialMessageData(t *testing.T) {
	// Create a new tunnel message
	tm := NewTunnelMessage(1024)
	// Create a new UDP address
	addr, _ := net.ResolveUDPAddr("udp", ":1234")
	// create some random data
	data := []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	// put the message onto the tunnel message
	fullmsg := tm.Write(data, addr)
	// read the message from the tunnel message
	msgdata, needmore, readdr, err := tm.Read(fullmsg[:getHeaderSize()+12])
	assert.Equal(t, true, needmore, "Need more isn't true")
	assert.Equal(t, nil, err, "Error isn't nil")

	/// Now give it the full message and make sure everything matches
	msgdata, needmore, readdr, err = tm.Read(fullmsg)
	assert.Equal(t, false, needmore, "Need more isn't false")
	assert.Equal(t, nil, err, "Error isn't nil")
	assert.Equal(t, addr.String(), readdr.String(), "Address isn't the same")
	assert.Equal(t, data, msgdata, "Data isn't the same")
}

func TestPartialMessageHeader(t *testing.T) {
	// Create a new tunnel message
	tm := NewTunnelMessage(1024)
	// Create a new UDP address
	addr, _ := net.ResolveUDPAddr("udp", ":1234")
	// create some random data
	data := []byte("12345678901234567890123456789012345678901234567890123456789012345678901234567890")
	// put the message onto the tunnel message
	fullmsg := tm.Write(data, addr)
	// read the message from the tunnel message
	msgdata, needmore, readdr, err := tm.Read(fullmsg[:getHeaderSize()-1])
	assert.Equal(t, true, needmore, "Need more isn't true")
	assert.Equal(t, nil, err, "Error isn't nil")

	/// Now give it the full message and make sure everything matches
	msgdata, needmore, readdr, err = tm.Read(fullmsg)
	assert.Equal(t, false, needmore, "Need more isn't false")
	assert.Equal(t, nil, err, "Error isn't nil")
	assert.Equal(t, addr.String(), readdr.String(), "Address isn't the same")
	assert.Equal(t, data, msgdata, "Data isn't the same")
}
