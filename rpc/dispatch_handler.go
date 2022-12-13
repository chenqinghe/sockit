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
		fn(&Packet{pkt: pkt.(*codec.TLVPacket)}, s)
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

type HandleFunc func(pkt *Packet, s *sockit.Session)

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
