package sockit

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

type Conn interface {
	// LocalAddr returns the local network address.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr

	SendPacket(p Packet) error

	ReadPacket() (Packet, error)

	Close() error
}

type conn struct {
	rdLock *sync.Mutex
	wrLock *sync.Mutex

	net.Conn

	codec Codec

	closed int32
}

var _ Conn = (*conn)(nil)

func newConn(c net.Conn, codec Codec) *conn {
	return &conn{
		rdLock: &sync.Mutex{},
		wrLock: &sync.Mutex{},
		Conn:   c,
		codec:  codec,
		closed: 0,
	}
}

func (c *conn) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return c.Conn.Close()
	}
	return nil
}

var DebugReadSend = false

func (c *conn) SendPacket(p Packet) error {
	c.wrLock.Lock()
	defer c.wrLock.Unlock()

	var wr io.Writer = c.Conn
	if DebugReadSend {
		wr = debugWriter{c.Conn}
	}

	return c.codec.Write(wr, p)
}

type debugWriter struct {
	w io.Writer
}

func (dw debugWriter) Write(p []byte) (int, error) {
	n, err := dw.w.Write(p)
	fmt.Println("write data:", p[:n])
	fmt.Println("write data hex:", hex.EncodeToString(p[:n]))
	return n, err
}

func (c *conn) ReadPacket() (Packet, error) {
	c.rdLock.Lock()
	defer c.rdLock.Unlock()

	var rd io.Reader = c.Conn
	if DebugReadSend {
		rd = debugReader{c}
	}
	return c.codec.Read(rd)
}

type debugReader struct {
	rd io.Reader
}

func (r debugReader) Read(p []byte) (int, error) {
	n, err := r.rd.Read(p)
	fmt.Println("read data:", p[:n])
	return n, err
}
