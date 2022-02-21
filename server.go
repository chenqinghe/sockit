package sockit

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
)

// ConnManager manage accepted connections
type ConnManager interface {
	// StoreConn store connections, and verify the connection is acceptable.
	// if acceptable, a Session will be returned.
	StoreConn(c Conn) (*Session, error)

	// FindSession find the Session specified by id
	FindSession(id int64) (*Session, bool)

	// RemoveSession find the Session and remove the Session from the pool.
	// The Session will be closed before removed.
	RemoveSession(id int64) error

	// RangeSession accepted a process function.
	// Every stored Session will be passed to function.
	RangeSession(fn func(session *Session))

	// Close close all sessions and release all resources.
	Close() error
}

type Server struct {
	listener net.Listener

	Codec   Codec
	Manager ConnManager

	closed int32
}

// NewServer create a new tcp server.
func NewServer(mgr ConnManager, codec Codec) *Server {
	return &Server{
		Codec:   codec,
		Manager: mgr,
	}
}

var ErrorServerClosed = fmt.Errorf("server closed")

// ListenAndServe is like http.ListenAndServe, it listens given address
// and accept new connections to the server, then pass the connection to ConnManager.
func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener

	for atomic.LoadInt32(&s.closed) != 1 {
		c, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		s.Manager.StoreConn(newConn(c, s.Codec))
	}

	return nil
}

// Close close the listener and ConnManager.
func (s *Server) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	if err := s.listener.Close(); err != nil {
		return err
	}

	if err := s.Manager.Close(); err != nil {
		return err
	}

	return nil
}
