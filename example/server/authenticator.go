package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chenqinghe/sockit"
	"github.com/chenqinghe/sockit/codec"
)

type tokenAuthenticator struct{}

func (a *tokenAuthenticator) Auth(c sockit.Conn) (sockit.User, error) {
	p, err := c.ReadPacket()
	if err != nil {
		return nil, err
	}
	switch packet := p.(type) {
	case codec.JsonPacket:
		if packet.Subject != 1212 { // verify if login packet
			return nil, errors.New("invalid packet")
		}

		var t struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(packet.Data, &t); err != nil {
			return nil, err
		}

		user := queryUserByToken(t.Token)

		return user, nil
	case codec.TLVPacket:
		packet.Type = 0x02
		packet.Data = []byte(`{
    "subject":"client_login",
    "payload":{
      "succ":true
    }
}`)
		if err := c.SendPacket(packet); err != nil {
			fmt.Println("send packet error:", err.Error())
			return nil, err
		}

		return user{
			ID: c.RemoteAddr().String(),
		}, nil
	default:
		return nil, fmt.Errorf("unknown packet")
	}

}

func queryUserByToken(token string) user {
	return user{
		Token: token,
		ID:    "123456789",
	}
}

type user struct {
	Token string `json:"token"`

	ID string `json:"id"`
}

func (u user) Id() string {
	return u.ID
}

func (u user) Valid() bool {
	return u.ID != ""
}
