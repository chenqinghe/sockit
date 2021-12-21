package sockit

import (
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager is a default implementation of ConnManager interface.
type Manager struct {

	// mu is a lock for concurrent read write conns
	mu *sync.RWMutex
	// conns stores all sessions
	conns map[int64]*Session

	Authenticator Authenticator
	handler       Handler

	opt             *NewManagerOptions
	keepaliveTicker *time.Ticker

	closed    chan struct{}
	closeDone chan struct{}
}

var _ ConnManager = (*Manager)(nil)

type NewManagerOptions struct {
	KeepaliveTick time.Duration
	OnLogin       func(s *Session)
	OnClose       func(s *Session)
}

func NewManager(handler Handler, opt *NewManagerOptions) *Manager {
	if opt == nil {
		opt = &NewManagerOptions{}
	}
	m := &Manager{
		mu:        &sync.RWMutex{},
		conns:     make(map[int64]*Session),
		handler:   handler,
		opt:       opt,
		closed:    make(chan struct{}),
		closeDone: make(chan struct{}),
	}

	return m
}

func (m *Manager) StoreConn(c Conn) (*Session, error) {
	var user User
	var err error
	if m.Authenticator != nil {
		user, err = m.Authenticator.Auth(c)
		if err != nil {
			logrus.Info("user auth failed: " + err.Error())
			c.Close()
			return nil, err
		}
		if !user.Valid() {
			logrus.WithFields(logrus.Fields{
				"remoteAddr": c.RemoteAddr().String(),
			}).Debugln("user not valid")
			c.Close()
			return nil, errors.New("invalid user")
		}
	}

	sess := NewSession(c, m, user, m.handler)

	m.mu.Lock()
	m.conns[sess.Id()] = sess
	m.mu.Unlock()

	logrus.Debug("accept a new connection, remote addr:" + c.RemoteAddr().String())

	if m.opt.OnLogin != nil {
		m.opt.OnLogin(sess)
	}

	return sess, nil
}

func (m *Manager) SetAuthenticator(authenticator Authenticator) {
	m.Authenticator = authenticator
}

func (m *Manager) Close() error {
	close(m.closed)
	m.RangeSession(func(s *Session) {
		m.RemoveSession(s.Id())
	})
	return nil
}

func (m *Manager) RemoveSession(id int64) error {
	m.mu.Lock()
	sess, ok := m.conns[id]
	delete(m.conns, id)
	m.mu.Unlock()

	if !ok {
		return nil
	}

	if m.opt.OnClose != nil {
		m.opt.OnClose(sess)
	}

	return sess.Close()

}

func (m *Manager) FindSession(id int64) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.conns[id]
	return sess, ok
}

func (m *Manager) RangeSession(fn func(s *Session)) {
	sessions := make([]*Session, 0, len(m.conns))

	m.mu.RLock()
	for _, v := range m.conns {
		sessions = append(sessions, v)
	}
	m.mu.RUnlock()

	for _, v := range sessions {
		fn(v)
	}
}

func (m *Manager) SetKeepAlive(b bool) {
	if b {
		go func() {
			for {
				select {
				case <-m.keepaliveTicker.C:
					now := time.Now()
					m.RangeSession(func(s *Session) {
						if now.Sub(s.lastPackTs) > m.opt.KeepaliveTick {
							logrus.WithFields(logrus.Fields{
								"remoteAddr": s.RemoteAddr().String(),
								"sessionID":  s.Id(),
							}).Debug("session keepalive timeout")
							m.RemoveSession(s.Id())
						}
					})
				case <-m.closed:
					m.keepaliveTicker.Stop()
					return
				}
			}
		}()
	}
}

func (m *Manager) SetKeepAlivePeriod(t time.Duration) {
	m.opt.KeepaliveTick = t
	if m.keepaliveTicker == nil {
		m.keepaliveTicker = time.NewTicker(t)
	} else {
		m.keepaliveTicker.Reset(t)
	}
}
