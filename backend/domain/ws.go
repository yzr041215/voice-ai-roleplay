package domain

import (
	"encoding/json"
	"fmt"
)

// MsgType 枚举（新增 state/asr_result/tts_start/tts_end/error）
type MsgType int

const (
	MsgTypeIntrupt MsgType = iota // 客户端打断（interrupt）
	MsgTypeTranslate              // 服务端翻译的中文（通用文本展示 / 也可用于 ASR 文本）
	MsgTypeState                  // 状态变更（v 审态）
	MsgTypeAsrResult              // ASR 结果（详细结构）
	MsgTypeTtsStart               // TTS 开始
	MsgTypeTtsChunk               // TTS 二进制包提示（元信息，实际 audio 通过 BinaryMessage 发送）
	MsgTypeTtsEnd                 // TTS 完成
	MsgTypeError                  // 错误消息
)

// 为了可读性，序列化时转成字符串
var msgTypeName = map[MsgType]string{
	MsgTypeIntrupt:  "intrupt",
	MsgTypeTranslate: "translate",
	MsgTypeState:    "state",
	MsgTypeAsrResult: "asr_result",
	MsgTypeTtsStart: "tts_start",
	MsgTypeTtsChunk: "tts_chunk",
	MsgTypeTtsEnd:   "tts_end",
	MsgTypeError:    "error",
}

var msgTypeValue = map[string]MsgType{
	"intrupt":    MsgTypeIntrupt,
	"translate":  MsgTypeTranslate,
	"state":      MsgTypeState,
	"asr_result": MsgTypeAsrResult,
	"tts_start":  MsgTypeTtsStart,
	"tts_chunk":  MsgTypeTtsChunk,
	"tts_end":    MsgTypeTtsEnd,
	"error":      MsgTypeError,
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

// Msg 结构体：Data 建议为 JSON bytes（上层可自定义结构体序列化到 Data）
type Msg struct {
	Type MsgType `json:"type"`
	Data []byte  `json:"data"`
}

// Encode 序列化
func (m *Msg) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// Decode 反序列化
func Decode(b []byte) (*Msg, error) {
	var m Msg
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
