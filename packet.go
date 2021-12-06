package sockit

import "time"

type Packet interface {
	// Id 获取包的id
	Id() int32

	// Time 获取包发送的时间
	Time() time.Time
}
