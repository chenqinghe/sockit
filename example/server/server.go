package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chenqinghe/sockit"
	"github.com/chenqinghe/sockit/codec"
)

func main() {
	handler := &packetHandler{}

	//proxy.DebugReadSend = true

	mgr := sockit.NewManager(handler, &sockit.NewManagerOptions{
		Authenticator: &tokenAuthenticator{},
	})
	mgr.SetKeepAlivePeriod(time.Second * 30)
	mgr.SetKeepAlive(true)

	handler.mgr = mgr

	srv := sockit.NewServer(mgr, &codec.TLVCodec{
		KeepaliveType:     0x03,
		KeepaliveRespType: 0x04,
	})

	go func() {
		if err := srv.ListenAndServe(":9090"); err != nil {
			panic(err)
		}
	}()

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	<-ch
	if err := srv.Close(); err != nil {
		panic(err)
	}

}

type packetHandler struct {
	mgr *sockit.Manager
}

func (ph *packetHandler) Handle(p sockit.Packet, sess *sockit.Session) {
	packet := p.(codec.TLVPacket)

	fmt.Println("handle a packet:", packet)

	if packet.IsKeepAlive() {
		packet.Type = 0x04
		if err := sess.SendPacket(packet); err != nil {
			fmt.Println("send packet error:", err)
			sess.Close()
		}
	}

}
