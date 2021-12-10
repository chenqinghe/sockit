package sockit

import (
	"errors"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Session struct {
	id int64

	c   Conn
	mgr ConnManager

	user User

	dataLock *sync.RWMutex
	data     map[string]interface{} // 用户自定义数据

	lastPackTs time.Time // 最后一个收到包的时间

	manuallyClosed bool
	closed         chan struct{}
}

var idGenerator int64

func newSession(c Conn, mgr ConnManager, user User) *Session {
	sess := &Session{
		id:       atomic.AddInt64(&idGenerator, 1),
		c:        c,
		mgr:      mgr,
		user:     user,
		dataLock: &sync.RWMutex{},
		data:     make(map[string]interface{}),
		closed:   make(chan struct{}),
	}

	go sess.readPacket()

	return sess
}

func (s *Session) readPacket() {
	handler := s.mgr.Handler()

	for {
		packet, err := s.c.ReadPacket()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				logrus.WithFields(logrus.Fields{
					"remoteAddr": s.c.RemoteAddr().String(),
					"sessionId":  s.Id(),
				}).Errorln("read packet error:", err.Error())
			}
			s.mgr.RemoveSession(s.Id())
			return
		}

		logrus.WithFields(logrus.Fields{
			"remoteAddr": s.c.RemoteAddr().String(),
			"sessionId":  s.Id(),
		}).Debug("receive a packet")

		s.lastPackTs = time.Now()

		go handler.Handle(packet, s)
	}
}

// Id returns current session id
func (s *Session) Id() int64 {
	return s.id
}

func (s *Session) User() User {
	return s.user
}

func (s *Session) Set(key string, value interface{}) {
	s.dataLock.Lock()
	defer s.dataLock.Unlock()

	s.data[key] = value
}

func (s *Session) Get(key string) (interface{}, bool) {
	s.dataLock.RLock()
	defer s.dataLock.RUnlock()

	val, ok := s.data[key]
	return val, ok
}

func (s *Session) Close() error {
	select {
	case <-s.closed:
		return nil
	default:
	}
	s.manuallyClosed = true
	close(s.closed)
	if err := s.mgr.RemoveSession(s.Id()); err != nil {
		return err
	}
	return s.c.Close()
}

func (s *Session) SendPacket(p Packet) error {
	return s.c.SendPacket(p)
}

func (s *Session) LocalAddr() net.Addr {
	return s.c.LocalAddr()
}

func (s *Session) RemoteAddr() net.Addr {
	return s.c.RemoteAddr()
}

func (s *Session) Stream() (Streamer, error) {
	//return s.c.Stream()
}
