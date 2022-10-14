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

	// users stores all login user ids
	ulock *sync.RWMutex
	users map[string]*Session

	authenticator Authenticator
	handler       Handler

	opts            *NewManagerOptions
	keepaliveTicker *time.Ticker

	closed    chan struct{}
	closeDone chan struct{}
}

var _ ConnManager = (*Manager)(nil)

type NewManagerOptions struct {
	// Authenticator is used to verify accepted connection is valid
	Authenticator Authenticator

	// ExclusiveUser indicates that only one client connect permitted with same user id
	// if user with same id login again, the previous one will be kicked out.
	ExclusiveUser bool

	// KeepaliveTick indicates time duration between every connection checking
	KeepaliveTick time.Duration

	// OnLogin specify session login event callback
	OnSessionCreated func(s *Session)

	// BeforeSessionClosed specify a pre-hook of session closed
	BeforeSessionClosed func(s *Session)

	// AfterSessionClosed specify a post-hook of session closed
	AfterSessionClosed func(s *Session)
}

func NewManager(handler Handler, opts *NewManagerOptions) *Manager {
	if opts == nil {
		opts = &NewManagerOptions{}
	}
	m := &Manager{
		mu:            &sync.RWMutex{},
		conns:         make(map[int64]*Session),
		ulock:         &sync.RWMutex{},
		users:         make(map[string]*Session),
		handler:       handler,
		authenticator: opts.Authenticator,
		opts:          opts,
		closed:        make(chan struct{}),
		closeDone:     make(chan struct{}),
	}
	if opts.KeepaliveTick != 0 {
		m.keepaliveTicker = time.NewTicker(opts.KeepaliveTick)
	}

	return m
}

func (m *Manager) StoreConn(c Conn) (*Session, error) {
	var user User
	var err error
	if m.authenticator != nil {
		user, err = m.authenticator.Auth(c)
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

		if m.opts.ExclusiveUser {
			m.ulock.Lock()
			if s, ok := m.users[user.Id()]; ok {
				s.Close()
				logrus.WithFields(logrus.Fields{
					"remoteAddr": c.RemoteAddr().String(),
				}).Infoln("same user login again, close previous")
			}
			delete(m.users, user.Id())
			m.ulock.Unlock()
		}
	}

	sess := NewSession(c, m, user, m.handler)

	m.mu.Lock()
	m.conns[sess.Id()] = sess
	m.mu.Unlock()

	if user != nil {
		m.ulock.Lock()
		m.users[user.Id()] = sess
		m.ulock.Unlock()
	}

	logrus.Debug("accept a new connection, remote addr:" + c.RemoteAddr().String())

	if m.opts.OnSessionCreated != nil {
		m.opts.OnSessionCreated(sess)
	}

	return sess, nil
}

func (m *Manager) SetAuthenticator(authenticator Authenticator) {
	m.authenticator = authenticator
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

	if sess.User() != nil {
		m.ulock.Lock()
		delete(m.users, sess.User().Id())
		m.ulock.Unlock()
	}

	if m.opts.BeforeSessionClosed != nil {
		m.opts.BeforeSessionClosed(sess)
	}

	if err := sess.close(); err != nil {
		return err
	}

	if m.opts.AfterSessionClosed != nil {
		m.opts.AfterSessionClosed(sess)
	}

	return nil
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
						if now.Sub(s.lastPackTs) > m.opts.KeepaliveTick {
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
	m.opts.KeepaliveTick = t
	if m.keepaliveTicker == nil {
		m.keepaliveTicker = time.NewTicker(t)
	} else {
		m.keepaliveTicker.Reset(t)
	}
}
