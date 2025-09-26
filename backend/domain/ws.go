package domain

import (
	"encoding/json"
	"fmt"
)

// MsgType 枚举
type MsgType int

const (
	MsgTypeIntrupt  MsgType = iota // 客户端打断
	MsgTypeTanslate                // 服务端翻译的中文
)

// 为了可读性，序列化时转成字符串
var msgTypeName = map[MsgType]string{
	MsgTypeIntrupt:  "intrupt",
	MsgTypeTanslate: "translate",
}

var msgTypeValue = map[string]MsgType{
	"intrupt":  MsgTypeIntrupt,
	"translate": MsgTypeTanslate,
}

// MarshalJSON 把枚举变成字符串
func (t MsgType) MarshalJSON() ([]byte, error) {
	if name, ok := msgTypeName[t]; ok {
		return json.Marshal(name)
	}
	return nil, fmt.Errorf("unknown MsgType: %d", t)
}

// UnmarshalJSON 把字符串还原成枚举
func (t *MsgType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if v, ok := msgTypeValue[s]; ok {
		*t = v
		return nil
	}
	return fmt.Errorf("unknown MsgType string: %s", s)
}

// Msg 结构体
type Msg struct {
	Type MsgType `json:"type"`
	Data []byte  `json:"data"`
}

// 序列化
func (m *Msg) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// 反序列化
func Decode(b []byte) (*Msg, error) {
	var m Msg
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}