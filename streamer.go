package sockit

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync/atomic"
)

type Streamer interface {
	io.ReadWriteCloser
}

const (
	StreamerLengthByteWidth = 2
)

// streamer is NOT concurrent safe
type streamer struct {
	conn io.ReadWriter
	buf  *bytes.Buffer

	closed      int32
	releaseFunc func()
}

var _ Streamer = (*streamer)(nil)

var ErrStreamerClosed = errors.New("stream already closed")

// Read data from streamer
// only if the streamer was closed, io.EOF returned
// otherwise, any EOF will be wrapped to io.ErrUnexpectedEOF
func (s *streamer) Read(p []byte) (int, error) {
	if atomic.LoadInt32(&s.closed) == 1 {
		return 0, ErrStreamerClosed
	}

	if s.buf.Len() > 0 {
		return s.buf.Read(p)
	}

	lengthBytes := make([]byte, StreamerLengthByteWidth)
	_, err := io.ReadFull(s.conn, lengthBytes)
	if err != nil {
		if err == io.EOF {
			return 0, io.ErrUnexpectedEOF
		}
		return 0, err
	}

	length := binary.BigEndian.Uint16(lengthBytes)
	if length == 0 {
		return 0, io.EOF
	}

	if n, err := io.CopyN(s.buf, s.conn, int64(length)); err != nil {
		if err == io.EOF {
			return int(n), io.ErrUnexpectedEOF
		}
		return int(n), err
	}

	return s.buf.Read(p)
}

var ErrSegmentSizeOverflow = errors.New("segment size overflow")

func (s *streamer) Write(p []byte) (int, error) {
	if atomic.LoadInt32(&s.closed) == 1 {
		return 0, ErrStreamerClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > 0xffff {
		return 0, ErrSegmentSizeOverflow
	}

	length := make([]byte, StreamerLengthByteWidth)
	binary.BigEndian.PutUint16(length, uint16(len(p)))
	if _, err := s.conn.Write(length); err != nil {
		return 0, err
	}
	return s.conn.Write(p)
}

func (s *streamer) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}
	s.closed = 1

	if _, err := s.Write([]byte{0, 0}); err != nil {
		return err
	}

	s.releaseFunc()

	return nil
}
