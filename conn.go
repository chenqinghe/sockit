package sockit

import (
	"bytes"
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

	// Stream create s streamer, all the data was treated as raw binary data
	Stream() (Streamer, error)

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

func (c *conn) SendPacket(p Packet) error {
	c.wrLock.Lock()
	defer c.wrLock.Unlock()

	var wr io.Writer = c.Conn
	if DebugReadSend {
		wr = debugWriter{c.Conn}
	}

	return c.codec.Write(wr, p)
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

func (c *conn) Stream() (Streamer, error) {
	c.rdLock.Lock()
	c.wrLock.Lock()

	return &streamer{
		conn: c.Conn,
		buf:  bytes.NewBuffer(nil),
		releaseFunc: func() {
			c.wrLock.Unlock()
			c.rdLock.Unlock()
		},
	}, nil
}

var DebugReadSend = false

type debugWriter struct {
	w io.Writer
}

func (dw debugWriter) Write(p []byte) (int, error) {
	n, err := dw.w.Write(p)
	fmt.Println("write data:", p[:n])
	fmt.Println("write data hex:", hex.EncodeToString(p[:n]))
	return n, err
}

type debugReader struct {
	rd io.Reader
}

func (r debugReader) Read(p []byte) (int, error) {
	n, err := r.rd.Read(p)
	fmt.Println("read data:", p[:n])
	return n, err
}

type priporityReader struct {
	rwc io.ReadWriteCloser

	rdbuf [4096]byte
	wrbuf [4096]byte

	r, w int

	readable chan struct{}
	writable chan struct{}

	err error

	takeover chan struct{}
	returned chan struct{}
}

func newPriporityReader(rw io.ReadWriter) *priporityReader {
	r := &priporityReader{
		rwc: rw,
	}

	go r.readLoop()
	go r.writeLoop()

	return r
}

func (pr *priporityReader) readLoop() {
	for {
		_, err := pr.rwc.Read(pr.rdbuf[pr.r:])
		if err != nil {
			pr.err = err
			pr.rwc.Close()
			return
		}

		pr.readable <- struct{}{}
	}
}

func (pr *priporityReader) writeLoop() {
	for {
		_, err := pr.rwc.Write(pr.rdbuf[:pr.w])
		if err != nil {
			pr.err = err
			pr.rwc.Close()
			return
		}

		pr.readable <- struct{}{}
	}
}

func (pr *priporityReader) Read(p []byte) (int, error) {

RETRY:
	if len(p) <= pr.r {
		copy(p, pr.rdbuf[:pr.r])
		pr.r -= len(p)
		return len(p), nil
	}

	n := copy(p, pr.rdbuf[:pr.r])
	pr.r -= n

	if n > 0 {
		return n, nil
	}

	if pr.err != nil {
		return 0, pr.err
	}

	select {
	case <-pr.readable:
		goto RETRY
	case <-pr.takeover:
		<-pr.returned
		goto RETRY
	}
}

func (pr *priporityReader) Write(p []byte) (int, error) {
	pr.rwc.(*net.TCPConn).SetLinger( )

	return pr.rwc.Write(p)
}
