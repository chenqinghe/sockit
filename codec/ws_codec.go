package codec

import (
	"errors"
	"io"
	"sync/atomic"
	"time"

	"github.com/chenqinghe/sockit"
)

type WebsocketCodec struct{}

type WebsocketPacket struct {
	id   int32
	ts   time.Time
	Type int
	Data []byte
}

func (p WebsocketPacket) Id() int32 {
	return 0
}

func (p WebsocketPacket) Time() time.Time {
	return p.ts
}

var idGen int32

func (codec *WebsocketCodec) Read(reader io.Reader) (p sockit.Packet, err error) {
	wsConn, ok := reader.(*sockit.WSConn)
	if !ok {
		return nil, errors.New("WebsocketCodec only supported reader of type '*proxy.WSConn'")
	}

	typ, data, err := wsConn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return WebsocketPacket{
		id:   atomic.AddInt32(&idGen, 1),
		ts:   time.Now(),
		Type: typ,
		Data: data,
	}, nil
}

func (codec *WebsocketCodec) Write(writer io.Writer, p sockit.Packet) error {
	wsConn, ok := writer.(*sockit.WSConn)
	if !ok {
		return errors.New("WebsocketCodec only supported writer of type '*proxy.WSConn'")
	}

	pkt, ok := p.(WebsocketPacket)
	if !ok {
		return errors.New("unsupported packet type")
	}

	return wsConn.WriteMessage(pkt.Type, pkt.Data)
}
