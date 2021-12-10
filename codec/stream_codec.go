package codec

import (
	"github.com/chenqinghe/sockit"
	"io"
	"time"
)


type StreamCodec struct{}

func (sc StreamCodec) Read(reader io.Reader) (sockit.Packet, error) {

}

func (sc StreamCodec) Write(writer io.Writer, p sockit.Packet) error {

}

type StreamPacket struct {
	data []byte
}

func (sp StreamPacket) Id() int32       { return 0 }
func (sp StreamPacket) Time() time.Time { return time.Now() }
