package sockit

import "github.com/gorilla/websocket"

type Handler interface {
	Handle(packet Packet, s *Session)
}

type WsHandler interface {
	Handle(c *websocket.Conn)
}
