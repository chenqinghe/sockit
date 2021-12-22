package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chenqinghe/sockit"
	codeclib "github.com/chenqinghe/sockit/codec"
)

func main() {
	handler := &packetHandler{}
	codec := &codeclib.TLVCodec{
		KeepaliveType:     0x03,
		KeepaliveRespType: 0x04,
	}
	client := sockit.NewClient(codec, handler, &sockit.NewClientOption{
		//KeepalivePeriod: time.Second,
		//EnableKeepalive: true,
		//HeartbeatPacketFactory: func() sockit.Packet {
		//	return codeclib.TLVPacket{
		//		PacketHead: codeclib.PacketHead{
		//			Type: 0x03,
		//		},
		//	}
		//},
		OnConnected: func(c sockit.Conn) error {
			data := []byte(`{"token":"fasdfsdfasdfadsf"}`)
			return c.SendPacket(codeclib.TLVPacket{
				PacketHead: codeclib.PacketHead{
					Type:      0x01,
					Version:   0x01,
					ID:        1,
					Timestamp: time.Now().Unix(),
					Length:    uint32(len(data)),
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
		time.Sleep(time.Second * 3)

		sess.SendPacket(codeclib.TLVPacket{
			PacketHead: codeclib.PacketHead{
				Type:      0x01,
				Version:   1,
				ID:        1,
				Timestamp: time.Now().Unix(),
				Length:    54,
			},
			Data: []byte(`{"subject":"upload_file","payload":{"name":"123.txt"}}`),
		})

	}()

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)

	<-ch
}

type packetHandler struct {
	cli *sockit.Client
}

func (h *packetHandler) Handle(p sockit.Packet, s *sockit.Session) {
	packet := p.(codeclib.TLVPacket)

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
	case "upload_file_resp":
		fmt.Println("allowed to upload file")
		stream, err := s.Stream()
		if err != nil {
			panic(err)
		}
		fmt.Println("start upload")

		if _, err := stream.Write([]byte("hello world")); err != nil {
			panic(err)
		}
		fmt.Println("start upload continue")

		if _, err := stream.Write([]byte("this is a sentence")); err != nil {
			panic(err)
		}
		if err := stream.Close(); err != nil {
			panic(err)
		}

		fmt.Println("stream closed")
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
