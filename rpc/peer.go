package rpc

import (
	"sync"
	"time"

	"github.com/chenqinghe/sockit"
	"github.com/chenqinghe/sockit/codec"
	"github.com/chenqinghe/sockit/reconnectpolicy"
)

type Peer struct {
	opt *PeerOption

	client      *sockit.Client
	cliInitOnce sync.Once
	cliHandler  sockit.Handler

	server      *sockit.Server
	srvInitOnce sync.Once
	srvHandler  sockit.Handler
}

type PeerOption struct {
	Authenticator sockit.Authenticator
}

var defaultOpt = &PeerOption{}

func NewPeer(opt *PeerOption) *Peer {
	if opt == nil {
		opt = defaultOpt
	}

	return &Peer{} // todo
}

func (p *Peer) Dial(network string, addr string) (*sockit.Session, error) {
	p.cliInitOnce.Do(func() {
		p.cliHandler = NewDispatchHandler()
		p.client = sockit.NewClient(codec.TLVCodec{}, p.cliHandler, &sockit.NewClientOptions{
			EnableKeepalive: true,
			KeepalivePeriod: time.Second,
			HeartbeatPacketFactory: func() sockit.Packet {
				return codec.TLVPacket{PacketHead: codec.PacketHead{
					Type: 0x01,
					ID:   time.Now().UnixNano(),
				}}
			},
			//OnConnected:     agentAuth{}.Login, // todo: 登录认证
			OnClosed:        func(session *sockit.Session) {},
			NeedReconnect:   true,
			ReconnectPolicy: reconnectpolicy.NewConstTime(time.Second),
		})
	})
	return p.client.Dial(network, addr)
}

func (p *Peer) ListenAndServe(addr string) error {
	p.srvInitOnce.Do(func() {
		p.srvHandler = NewDispatchHandler()
		manager := sockit.NewManager(p.srvHandler, &sockit.NewManagerOptions{
			Authenticator: p.opt.Authenticator,
			KeepaliveTick: time.Second * 2,
		})
		manager.SetKeepAlive(true)
		p.server = sockit.NewServer(manager, codec.TLVCodec{})
	})

	return p.server.ListenAndServe(addr)
}

func (p *Peer) ServerRegister(typ int32, fn HandleFunc) {
	p.srvHandler.(*DispatchHandler).Register(typ, fn)
}

func (p *Peer) ClientRegister(typ int32, fn HandleFunc) {
	p.cliHandler.(*DispatchHandler).Register(typ, fn)
}
