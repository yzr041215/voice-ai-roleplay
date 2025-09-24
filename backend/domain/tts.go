package domain

// TtsResponse 表示TTS API的响应
type TtsResponse struct {
	Reqid     string `json:"reqid"`
	Operation string `json:"operation"`
	Sequence  int    `json:"sequence"`
	Data      string `json:"data"` // base64编码的音频数据
	Addition  struct {
		Duration string `json:"duration"` // 音频时长(毫秒)
	} `json:"addition"`
}
