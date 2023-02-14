package rpc

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/chenqinghe/sockit"
	"github.com/chenqinghe/sockit/codec"
)

type DispatchHandler struct {
	handlers map[int32]func(pkt sockit.Packet, s *sockit.Session)
}

func NewDispatchHandler() *DispatchHandler {
	return &DispatchHandler{
		handlers: make(map[int32]func(pkt sockit.Packet, s *sockit.Session)),
	}
}

func (dh *DispatchHandler) Register(typ int32, fn HandleFunc) {
	dh.handlers[typ] = func(pkt sockit.Packet, s *sockit.Session) {
		tlvPkt := pkt.(codec.TLVPacket)
		buf := bytes.NewBuffer(nil)
		fn(&Packet{pkt: &tlvPkt}, &Session{
			Writer:  buf,
			Session: s,
		})
		s.SendPacket(codec.TLVPacket{
			PacketHead: codec.PacketHead{
				Type:      int32(1<<30) | tlvPkt.Type,
				ID:        pkt.Id(),
				Timestamp: time.Now().UnixNano(),
				Length:    uint64(buf.Len()),
			},
			Data: buf.Bytes(),
		})
	}
}

func (dh *DispatchHandler) Handle(p sockit.Packet, s *sockit.Session) {
	pkt := p.(codec.TLVPacket)
	if fn, ok := dh.handlers[pkt.Type]; ok {
		fn(pkt, s)
	} else {
		log.Printf("unknown packet type: %d\n", pkt.Type)
	}
}

type HandleFunc func(pkt *Packet, s *Session)

type Packet struct {
	pkt *codec.TLVPacket
}

func (p Packet) Data() io.Reader {
	return bytes.NewBuffer(p.pkt.Data)
}

func (p Packet) Time() time.Time {
	return p.pkt.Time()
}

func (p Packet) Id() int64 {
	return p.pkt.Id()
}

func (p Packet) Length() int {
	return int(p.pkt.Length)
}

func (p Packet) BindJson(v interface{}) error {
	return json.Unmarshal(p.pkt.Data, v)
}

type Session struct {
	Writer *bytes.Buffer
	*sockit.Session
}
