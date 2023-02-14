package main

import (
	"github.com/chenqinghe/sockit"
	"github.com/chenqinghe/sockit/codec"
	"github.com/chenqinghe/sockit/rpc"
	"time"
)

func main() {

	sockit.DebugReadSend = true

	mux := rpc.NewDispatchHandler()

	mux.Register(100, func(pkt *rpc.Packet, s *rpc.Session) {
		if err := s.SendPacket(codec.TLVPacket{
			PacketHead: codec.PacketHead{
				Type:      101,
				ID:        pkt.Id(),
				Timestamp: time.Now().UnixMilli(),
			},
			Data: []byte("hello world2!"),
		}); err != nil {
			panic(err)
		}
	})
	mux.Register(0x01, func(pkt *rpc.Packet, s *rpc.Session) {
		if err := s.SendPacket(codec.TLVPacket{
			PacketHead: codec.PacketHead{
				ID:   pkt.Id(),
				Type: 0x02,
			},
		}); err != nil {
			panic(err)
		}
	})

	p := rpc.NewPeer(&rpc.PeerOption{
		SrvHandler: mux,
	})

	go func() {
		if err := p.ListenAndServe(":9094"); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)

	cliMux := rpc.NewDispatchHandler()
	cliMux.Register(0x02, func(pkt *rpc.Packet, s *rpc.Session) {})

	p2 := rpc.NewPeer(&rpc.PeerOption{
		CliHandler: cliMux,
	})

	sess, err := p2.Dial("tcp", "127.0.0.1:9094")
	if err != nil {
		panic(err)
	}

	if err := sess.SendPacket(codec.TLVPacket{
		PacketHead: codec.PacketHead{
			Type:      100,
			ID:        1,
			Timestamp: time.Now().UnixMilli(),
			Length:    11,
		},
		Data: []byte("hello world"),
	}); err != nil {
		panic(err)
	}

	select {}
}
