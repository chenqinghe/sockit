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
	"github.com/chenqinghe/sockit/reconnectpolicy"
)

func main() {
	handler := &packetHandler{}
	tlvCodec := &codec.TLVCodec{
		KeepaliveType:     0x03,
		KeepaliveRespType: 0x04,
	}
	client := sockit.NewClient(tlvCodec, handler, &sockit.NewClientOptions{
		KeepalivePeriod: time.Second,
		EnableKeepalive: true,
		HeartbeatPacketFactory: func() sockit.Packet {
			return codec.TLVPacket{
				PacketHead: codec.PacketHead{
					Type: 0x03,
				},
			}
		},
		NeedReconnect: true,
		ReconnectPolicy: reconnectpolicy.ConstTime{
			Duration: time.Second,
		},
		OnConnected: func(c sockit.Conn) error {
			data := []byte(`{"token":"here is your token"}`)
			return c.SendPacket(codec.TLVPacket{
				PacketHead: codec.PacketHead{
					Type:      0x01,
					Version:   0x01,
					ID:        1,
					Timestamp: time.Now().Unix(),
					Length:    uint64(len(data)),
				},
				Data: data,
			})
		},
	})

	defer client.Close()

	handler.cli = client

	sess, err := client.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			if err := sess.SendPacket(codec.TLVPacket{
				PacketHead: codec.PacketHead{
					Type:      0x01,
					Version:   0x01,
					ID:        1,
					Timestamp: time.Now().Unix(),
					Length:    11,
				},
				Data: []byte("hello world"),
			}); err != nil {
				fmt.Println("send packet error:", err)
			}
		}
	}()

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	<-ch
}

type packetHandler struct {
	cli *sockit.Client
}

func (h *packetHandler) Handle(p sockit.Packet, s *sockit.Session) {
	packet := p.(codec.TLVPacket)

	if packet.IsKeepAlive() {
		return
	}

	var proto Protocol
	if err := json.Unmarshal(packet.Data, &proto); err != nil {
		fmt.Println("unmarshal json error:", err)
		s.Close()
		return
	}

	switch proto.Subject {
	case "client_login":
		resp := LoginResp{}
		if err := json.Unmarshal(proto.Payload, &resp); err != nil {
			fmt.Println("unmarshal payload error:", err)
			s.Close()
			return
		}

		if resp.Succ {
			fmt.Println("login success!")
		} else {
			fmt.Println("login failed!")
		}
	}
}

type Protocol struct {
	Subject string          `json:"subject"`
	Source  string          `json:"source"`
	Payload json.RawMessage `json:"payload"`
}

type LoginResp struct {
	Succ bool `json:"succ"`
}
