package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/chenqinghe/sockit"
	"github.com/sirupsen/logrus"
)

type TLVCodec struct {
	KeepaliveType     int32
	KeepaliveRespType int32
}

type TLVPacket struct {
	PacketHead

	isKeepAlive bool
	Data        []byte
}

type PacketHead struct {
	Label     uint16 // 协议标识
	Version   uint16
	Type      int32
	ID        int64
	Timestamp int64
	Length    uint64
}

var headSize = binary.Size(&PacketHead{})

func (p TLVPacket) Id() int64 {
	return p.ID
}

func (p TLVPacket) Time() time.Time {
	return time.Unix(0, p.Timestamp*1e6)
}

func (p TLVPacket) IsKeepAlive() bool {
	return p.isKeepAlive
}

func (c TLVCodec) Read(reader io.Reader) (p sockit.Packet, err error) {
	var head PacketHead

	headData := make([]byte, headSize)
	if _, err := io.ReadFull(reader, headData); err != nil {
		return nil, err
	}

	if err := binary.Read(bytes.NewReader(headData), binary.BigEndian, &head); err != nil {
		return nil, err
	}

	logrus.WithFields(logrus.Fields{
		"reqID": head.ID,
	}).Debugln("packet data length:", strconv.Itoa(int(head.Length)))

	data := make([]byte, head.Length+1) // data and sum byte

	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	logrus.Debugln("packet data:", string(data[:len(data)-1]))

	sum := (&checksum{}).Write(headData).Write(data[:len(data)-1]).Sum()
	if sum != data[len(data)-1] {
		return nil, fmt.Errorf("invalid checksum")
	}

	return TLVPacket{
		isKeepAlive: head.Type == c.KeepaliveType || head.Type == c.KeepaliveRespType,
		PacketHead:  head,
		Data:        data[:len(data)-1],
	}, nil
}

func (c TLVCodec) Write(writer io.Writer, p sockit.Packet) error {
	if p == nil {
		return fmt.Errorf("cannot send nil packet")
	}
	pkt, ok := p.(TLVPacket)
	if !ok {
		return fmt.Errorf("unknown packet type: %s", reflect.TypeOf(p).String())
	}
	pkt.Timestamp = time.Now().UnixNano() / 1e6
	pkt.Length = uint64(len(pkt.Data))

	buf := bytes.NewBuffer(nil)

	if err := binary.Write(buf, binary.BigEndian, pkt.PacketHead); err != nil {
		return err
	}

	if _, err := writer.Write(buf.Bytes()); err != nil {
		return err
	}

	if _, err := writer.Write(pkt.Data); err != nil {
		return err
	}

	sum := (&checksum{}).Write(buf.Bytes()).Write(pkt.Data).Sum()

	if _, err := writer.Write([]byte{sum}); err != nil {
		return err
	}

	return nil
}

type checksum struct {
	sum uint32
}

func (cs *checksum) Write(p []byte) *checksum {
	for _, v := range p {
		cs.sum += uint32(v)
	}
	return cs
}

func (cs *checksum) Sum() uint8 {
	return uint8(cs.sum % 256)
}
