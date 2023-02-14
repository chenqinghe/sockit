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

	server      *sockit.Server
	srvInitOnce sync.Once
}

type PeerOption struct {
	Authenticator sockit.Authenticator

	SrvHandler sockit.Handler
	CliHandler sockit.Handler
}

var defaultOpt = &PeerOption{
	SrvHandler: NewDispatchHandler(),
	CliHandler: NewDispatchHandler(),
}

func NewPeer(opt *PeerOption) *Peer {
	if opt == nil {
		opt = defaultOpt
	}

	return &Peer{opt: opt} // todo
}

func (p *Peer) Dial(network string, addr string) (*sockit.Session, error) {
	p.cliInitOnce.Do(func() {
		p.client = sockit.NewClient(codec.TLVCodec{
			KeepaliveType:     0x01,
			KeepaliveRespType: 0x02,
		}, p.opt.CliHandler, &sockit.NewClientOptions{
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
		manager := sockit.NewManager(p.opt.SrvHandler, &sockit.NewManagerOptions{
			Authenticator: p.opt.Authenticator,
			KeepaliveTick: time.Second * 2,
		})
		manager.SetKeepAlive(true)
		p.server = sockit.NewServer(manager, codec.TLVCodec{
			KeepaliveType:     0x01,
			KeepaliveRespType: 0x02,
		})
	})

	return p.server.ListenAndServe(addr)
}
