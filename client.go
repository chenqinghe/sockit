package sockit

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type Client struct {
	mgr   ConnManager
	codec Codec

	opts *NewClientOptions

	closed chan struct{}
}

type NewClientOptions struct {
	EnableKeepalive        bool
	KeepalivePeriod        time.Duration
	HeartbeatPacketFactory func() Packet
	OnConnected            func(c Conn) error
	OnClosed               func(session *Session)
	NeedReconnect          bool
	ReconnectPolicy        ReconnectPolicy
}

func NewClient(codec Codec, handler Handler, opts *NewClientOptions) *Client {
	if opts == nil {
		opts = &NewClientOptions{
			EnableKeepalive: false,
			NeedReconnect:   false,
		}
	}

	cli := &Client{
		opts:   opts,
		codec:  codec,
		closed: make(chan struct{}),
	}
	cli.mgr = NewManager(handler, &NewManagerOptions{
		AfterSessionClosed: func(s *Session) {
			cli.reconnect(s)
		},
	})

	if cli.opts.EnableKeepalive {
		go cli.heartbeat()
	}

	return cli
}

func (cli *Client) Dial(network string, addr string) (*Session, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	conn := newConn(c, cli.codec)
	if cli.opts.OnConnected != nil {
		if err := cli.opts.OnConnected(conn); err != nil {
			return nil, err
		}
	}

	return cli.mgr.StoreConn(conn)
}

type ReconnectPolicy interface {
	Retry() bool
	Timer() *time.Timer
}

func (cli *Client) reconnect(sess *Session) {
	addr := sess.RemoteAddr()
	policy := cli.opts.ReconnectPolicy

	for policy.Retry() && cli.needReconnect(sess) {
		if s, err := cli.Dial(addr.Network(), addr.String()); err == nil {
			*sess = *s // replace old session
			return
		}

		timer := policy.Timer()
		select {
		case <-cli.closed:
			timer.Stop()
			return
		case <-timer.C:
			timer.Stop()
		}
	}
}

func (cli *Client) needReconnect(sess *Session) bool {
	if !cli.opts.NeedReconnect {
		return false
	}

	return !sess.manuallyClosed
}

func (cli *Client) heartbeat() {
	ticker := time.NewTicker(cli.opts.KeepalivePeriod)
	for {
		select {
		case <-ticker.C:
			cli.mgr.RangeSession(func(s *Session) {
				if err := s.SendPacket(cli.opts.HeartbeatPacketFactory()); err != nil {
					logrus.WithFields(logrus.Fields{
						"sessionId":  s.Id(),
						"remoteAddr": s.RemoteAddr().String(),
					}).Errorln("send heartbeat packet error:", err.Error())
					if err := cli.mgr.RemoveSession(s.Id()); err != nil {
						logrus.WithFields(logrus.Fields{
							"sessionId":  s.Id(),
							"remoteAddr": s.RemoteAddr().String(),
						}).Errorln("remove session error:", err.Error())
					}
				}
			})
		case <-cli.closed:
			ticker.Stop()
			return
		}
	}
}

func (cli *Client) FindSession(id int64) (*Session, bool) { return cli.mgr.FindSession(id) }
func (cli *Client) RangeSession(fn func(sess *Session))   { cli.mgr.RangeSession(fn) }

func (cli *Client) Close() error {
	close(cli.closed)
	return cli.mgr.Close()
}
