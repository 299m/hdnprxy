package proxy

import (
	"bytes"
	"encoding/binary"
	"github.com/299m/util/util"
	"net"
)

const (
	ISUDP  = 1
	ISIPV6 = 1 << 1
)

type ErrNotUdp struct{}

func (e ErrNotUdp) Error() string {
	return "Not a UDP packet"
}

// / For UDP we need to convey additional info - so put a header in the packet that goes down the tunnel
type TunnelMessage struct {
	udpheader []byte
	buf       *bytes.Buffer
}

func NewTunnelMessage(bufsize int) *TunnelMessage {
	return &TunnelMessage{
		udpheader: make([]byte, 16+4+2), /// 8 bytes for the IP and 4 bytes for the port and 2 bytes for additional info
		buf:       bytes.NewBuffer(make([]byte, bufsize)),
	}
}

func (t *TunnelMessage) putUdpHeader(addr *net.UDPAddr, datasize int) {
	t.buf.Reset()
	ip4 := addr.IP.To4()
	info := int16(ISUDP)
	addrsize := 16 //// fixed size header
	if ip4 == nil {
		info = int16(ISUDP | ISIPV6)
	}

	headersize := binary.Size(info)
	binary.LittleEndian.PutUint16(t.udpheader, uint16(info))
	copy(t.udpheader[headersize:], addr.IP[:addrsize])
	headersize += addrsize
	port := uint32(addr.Port)
	binary.LittleEndian.PutUint32(t.udpheader[headersize:], port)
	headersize += binary.Size(port)
	binary.LittleEndian.PutUint32(t.udpheader[headersize:], uint32(datasize))
	t.buf.Write(t.udpheader)
}

func getHeaderSize() int {
	headersize := binary.Size(int16(0))
	headersize += 16 //// fixed size header
	headersize += binary.Size(uint32(0))
	headersize += binary.Size(uint32(0))
	return headersize
}

func (t *TunnelMessage) retrieveUdpHeader(data []byte) (addr *net.UDPAddr, msgdata []byte, needmore bool, err error) {
	if len(data) < getHeaderSize() {
		///// Not an error - just wait for more data and try again
		return nil, nil, true, nil
	}
	headersize := 0
	info := binary.LittleEndian.Uint16(data)
	headersize += binary.Size(info)
	if info&ISUDP == 0 {
		return nil, nil, false, ErrNotUdp{}
	}
	addrsize := 16
	ipaddr := make([]byte, addrsize)
	copy(ipaddr, data[headersize:headersize+addrsize])
	headersize += addrsize

	port := binary.LittleEndian.Uint32(data[headersize:])
	headersize += binary.Size(port)
	size := binary.LittleEndian.Uint32(data[headersize:])
	headersize += binary.Size(size)

	addr = &net.UDPAddr{
		IP:   ipaddr,
		Port: int(port),
	}
	msgdata = data[headersize:]
	needmore = len(msgdata) < int(size)
	return
}

func (t *TunnelMessage) Write(data []byte, from *net.UDPAddr) (fullmsg []byte) {
	t.putUdpHeader(from, len(data))
	t.buf.Write(data)
	return t.buf.Bytes()
}

// // If need more is set, read in more data and pass this current data + the new data in again
func (t *TunnelMessage) Read(data []byte) (msgdata []byte, needmore bool, addr *net.UDPAddr, err error) {
	addr, msgdata, needmore, err = t.retrieveUdpHeader(data)
	util.CheckError(err)
	return msgdata, needmore, addr, err
}
