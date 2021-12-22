package sockit

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Conn interface {
	// LocalAddr returns the local network address.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address.
	RemoteAddr() net.Addr

	SendPacket(p Packet) error

	ReadPacket() (Packet, error)

	// Stream create a streamer, all the data was treated as raw binary data
	//Stream() (Streamer, error)

	Close() error
}

type conn struct {
	rdLock *sync.Mutex
	wrLock *sync.Mutex

	net.Conn

	codec Codec

	streamOpened int32
	cond         *sync.Cond

	closed int32
}

var _ Conn = (*conn)(nil)

func newConn(c net.Conn, codec Codec) *conn {
	return &conn{
		rdLock: &sync.Mutex{},
		wrLock: &sync.Mutex{},
		Conn:   c,
		codec:  codec,
		cond:   sync.NewCond(&sync.Mutex{}),
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

	return c.codec.Write(c, p)
}

func (c *conn) ReadPacket() (Packet, error) {
	c.rdLock.Lock()
	defer c.rdLock.Unlock()

	for c.streamOpened == 1 {
		c.cond.Wait()
	}

	return c.codec.Read(c)
}

func (c *conn) Stream() (Streamer, error) {

	c.cond.L.Lock()
	if c.streamOpened == 1 {
		return nil, errors.New("a stream already opened, please close it first")
	}
	c.cond.L.Unlock()

	c.Conn.SetDeadline(time.Now().Add(-time.Second)) // to ensure all io operations timeout and returned immediately

	c.Conn.SetDeadline(time.Time{}) // cancel timeout

	return &streamer{
		rw:  c.Conn,
		buf: bytes.NewBuffer(nil),
		releaseFunc: func() {
			c.cond.L.Lock()
			c.streamOpened = 0
			c.cond.L.Unlock()

			c.cond.Broadcast()
		},
	}, nil
}

func (c *conn) Read(p []byte) (int, error) {
RETRY:
	c.cond.L.Lock()
	for c.streamOpened == 1 {
		c.cond.Wait()
	}
	c.cond.L.Unlock()

	n, err := c.Conn.Read(p)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if n > 0 {
				return n, nil
			}
			goto RETRY
		}

		return n, err
	}
	fmt.Println(string(p[:n]))

	return n, nil
}

func (c *conn) Write(p []byte) (int, error) {
RETRY:
	c.cond.L.Lock()
	for c.streamOpened == 1 {
		c.cond.Wait()
	}
	c.cond.L.Unlock()

	n, err := c.Conn.Write(p)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			if n > 0 {
				return n, nil
			}
			goto RETRY
		}

		return n, err
	}

	return n, nil
}
