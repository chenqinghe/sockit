package main

import (
	"encoding/json"
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

	mgr := sockit.NewManager(handler, nil)
	mgr.Authenticator = &tokenAuthenticator{}
	mgr.SetKeepAlivePeriod(time.Second * 30)
	//mgr.SetKeepAlive(true)

	handler.mgr = mgr

	srv := sockit.Server{
		Manager: mgr,
		Codec: &codec.TLVCodec{
			KeepaliveType:     0x03,
			KeepaliveRespType: 0x04,
		},
	}

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

	//fmt.Println("handle a packet:", packet)

	if packet.IsKeepAlive() {
		packet.Type = 0x04
		if err := sess.SendPacket(packet); err != nil {
			fmt.Println("send packet error:", err)
			sess.Close()
		}
		return
	}

	var proto = Protocol{
		Payload: struct {
			Name string `json:"name"`
		}{},
	}
	if err := json.Unmarshal(packet.Data, &proto); err != nil {
		panic(err)
	}

	fmt.Println("subject:", proto.Subject, "name:", proto.Payload)

	if err := sess.SendPacket(codec.TLVPacket{
		PacketHead: codec.PacketHead{
			Type:      0x01,
			Version:   1,
			ID:        1,
			Timestamp: time.Now().Unix(),
			Length:    54,
		},
		Data: []byte(`{"subject":"upload_file_resp","payload":{"name":"123.txt"}}`),
	}); err != nil {
		panic(err)
	}

	fmt.Println("ready to open stream")

	stream, err := sess.Stream()
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	fmt.Println("stream opened")

	bytes := make([]byte, 4096)
	for {
		n, err := stream.Read(bytes)
		if err != nil {
			panic(err)
		}
		fmt.Println(bytes[:n])
	}



}

type Protocol struct {
	Subject string      `json:"subject"`
	Source  string      `json:"source"`
	Payload interface{} `json:"payload"`
}
