package codec

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
	"time"

	"github.com/chenqinghe/sockit"
)

type JsonCodec struct {
	Delimiter string
}

type JsonPacket struct {
	Type      int8            `json:"type"`
	Version   uint8           `json:"version"`
	Source    int16           `json:"source"`
	Subject   int32           `json:"subject"`
	ID        int32           `json:"id"`
	Timestamp int64           `json:"time"`
	Data      json.RawMessage `json:"data"`
}

func (p JsonPacket) IsKeepAlive() bool { return p.Subject == 0 }
func (p JsonPacket) Id() int32         { return p.ID }
func (p JsonPacket) Time() time.Time   { return time.Unix(p.Timestamp, 0) }

func (codec *JsonCodec) Read(reader io.Reader) (sockit.Packet, error) {
	data := make([]byte, 0, 4096)
	buf := make([]byte, 1)

	for {
		if _, err := reader.Read(buf); err != nil {
			return nil, err
		}
		data = append(data, buf...)
		if len(data) >= len(codec.Delimiter) &&
			bytes.Equal([]byte(codec.Delimiter), data[len(data)-len(codec.Delimiter):]) {
			break
		}
	}

	logrus.Debug("final data: " + string(data))

	var p JsonPacket

	if err := json.Unmarshal(data[:len(data)-len(codec.Delimiter)], &p); err != nil {
		return nil, err
	}

	return p, nil
}

func (codec JsonCodec) Write(writer io.Writer, p sockit.Packet) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}

	data = append(data, codec.Delimiter...)

	logrus.Debug("write data: " + string(data))

	if _, err := writer.Write(data); err != nil {
		return err
	}
	return nil
}
