package sockit

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsServer struct {
	Addr      string
	Path      string
	WsHandler WsHandler
	Manager   ConnManager
	Upgrader  *websocket.Upgrader

	server *http.Server
}

func (ws *WsServer) ListenAndServe() error {
	if ws.Path == "" {
		ws.Path = "/"
	}
	if ws.Upgrader == nil {
		ws.Upgrader = defaultUpgrader
	}
	if ws.WsHandler == nil {
		ws.WsHandler = &wsHandler{
			mgr: ws.Manager,
		}
	}

	router := http.NewServeMux()
	router.Handle("/", http.FileServer(http.Dir(".")))
	router.HandleFunc(ws.Path, func(w http.ResponseWriter, r *http.Request) {
		conn, err := ws.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("upgrade error:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		go ws.WsHandler.Handle(conn)
	})

	ws.server = &http.Server{
		Addr:    ws.Addr,
		Handler: router,
	}

	return ws.server.ListenAndServe()

}

type WSConn struct {
	*websocket.Conn
}

var _ net.Conn = &WSConn{}

var ErrNotImplement = errors.New("not implement yet")

func (c *WSConn) Read(p []byte) (int, error) {
	return 0, ErrNotImplement
}

func (c *WSConn) Write(p []byte) (int, error) {
	return 0, ErrNotImplement
}

func (c *WSConn) SetDeadline(t time.Time) error {
	return ErrNotImplement
}

func (c *WSConn) ReadPacket() (Packet, error) {
	typ, data, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}

	return WebsocketPacket{
		id:   atomic.AddInt64(&idGen, 1),
		ts:   time.Now(),
		Type: typ,
		Data: data,
	}, nil
}

func (c *WSConn) SendPacket(p Packet) error {
	pkt, ok := p.(WebsocketPacket)
	if !ok {
		return errors.New("unsupported packet type")
	}
	return c.WriteMessage(pkt.Type, pkt.Data)
}

type wsHandler struct {
	mgr ConnManager
}

func (h *wsHandler) Handle(c *websocket.Conn) {
	conn := &WSConn{
		Conn: c,
	}

	h.mgr.StoreConn(conn)
}

type WebsocketPacket struct {
	id   int64
	ts   time.Time
	Type int
	Data []byte
}

func (p WebsocketPacket) Id() int64 {
	return p.id
}

func (p WebsocketPacket) Time() time.Time {
	return p.ts
}

var idGen int64
