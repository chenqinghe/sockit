package sockit

import (
	"io"
)

type Codec interface {
	Read(reader io.Reader) (p Packet, err error)
	Write(writer io.Writer, p Packet) error
}
