package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/chenqinghe/sockit"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/websocket"
)

func main() {
	http.HandleFunc("/sign", signJwt)
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil {
			panic(err)
		}
	}()

	srv := sockit.WsServer{
		Addr: ":8080",
		Path: "/ws",
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     checkToken,
		},
		Manager: sockit.NewManager(&handler{}, nil),
	}

	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

}

type handler struct{}

func (h *handler) Handle(p sockit.Packet, sess *sockit.Session) {
	fmt.Println("receive a packet:", string(p.(sockit.WebsocketPacket).Data), sess.RemoteAddr().String())

	var i int
	for {
		if err := sess.SendPacket(sockit.WebsocketPacket{
			Type: websocket.TextMessage,
			Data: []byte(fmt.Sprintf("this is log message: %d", i)),
		}); err != nil {
			fmt.Println("send packet error:", err)
			return
		}

		i++
		time.Sleep(time.Second)
	}
}

func checkToken(req *http.Request) bool {
	tokenString := req.URL.Query().Get("token")
	if len(tokenString) == 0 {
		return false
	}

	claims := &TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return signingKey, nil
	})
	if err != nil || !token.Valid {
		return false
	}

	return true
}

var signingKey = []byte("abc#123")

type TokenClaims struct {
	jwt.StandardClaims
	Token   string `json:"token"`
	PodName string `json:"podName"`
}

func signJwt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")

	pod := r.URL.Query().Get("pod")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		StandardClaims: jwt.StandardClaims{
			Audience:  "public",
			ExpiresAt: time.Now().Add(time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "www.example.com",
			NotBefore: time.Now().Unix(),
			Subject:   "test",
		},
		Token:   "thisisatoken",
		PodName: pod,
	})

	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(tokenString))
}
