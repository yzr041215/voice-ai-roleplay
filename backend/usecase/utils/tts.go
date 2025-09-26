package utils

import (
	"bytes"
	"context"
	"demo/config"
	"demo/pkg/log"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

// ---- WebSocket 请求/响应结构 ----
type ttsRequest struct {
	Audio   audioParam   `json:"audio"`
	Request requestParam `json:"request"`
}
type audioParam struct {
	VoiceType  string  `json:"voice_type"`
	Encoding   string  `json:"encoding"`
	SpeedRatio float64 `json:"speed_ratio"`
}
type requestParam struct {
	Text string `json:"text"`
}

type relayTTSResponse struct {
	Reqid     string    `json:"reqid"`
	Operation string    `json:"operation"`
	Sequence  int       `json:"sequence"`
	Data      string    `json:"data"`
	Addition  *addition `json:"addition,omitempty"`
}
type addition struct {
	Duration string `json:"duration"`
}

type TtsStream struct {
	l      *log.Logger
	config *config.Config
}

func NewTtsStream(l *log.Logger, c *config.Config) *TtsStream {
	return &TtsStream{
		l:      l.WithModule("TtsUsecase"),
		config: c,
	}
}

// PCMChunk 表示流式输出的 PCM 数据
type PCMChunk struct {
	Seq     int     // 序号（服务端的 Sequence）
	Samples []int16 // 解码后的 PCM 采样数据
}

// TtsStream 流式输入 text，流式输出 PCM
func (t *TtsStream) TtsStream(
	ctx context.Context,
	textChunks <-chan string, // 输入文本块
	voiceType string,
) (<-chan PCMChunk, <-chan error) {

	// 返回的两个 channel
	out := make(chan PCMChunk, 8)
	errCh := make(chan error, 1)

	u := url.URL{Scheme: "wss", Host: "openai.qiniu.com", Path: "/v1/voice/tts"}
	header := http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", t.config.Tts.ApiKey)},
		"VoiceType":     []string{voiceType},
	}

	c, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		errCh <- fmt.Errorf("dial websocket fail: %w", err)
		close(out)
		close(errCh)
		return out, errCh
	}

	// 写入协程
	go func() {
		defer c.WriteMessage(websocket.CloseMessage, []byte{})
		for chunk := range textChunks {
			params := &ttsRequest{
				Audio: audioParam{
					VoiceType:  voiceType,
					Encoding:   "pcm", // ⚠️ 注意这里设置 pcm 输出
					SpeedRatio: 1.0,
				},
				Request: requestParam{
					Text: chunk,
				},
			}
			data, _ := json.Marshal(params)
			if err := c.WriteMessage(websocket.BinaryMessage, data); err != nil {
				errCh <- fmt.Errorf("send text chunk fail: %w", err)
				return
			}
		}
	}()

	// 读取协程
	go func() {
		defer close(out)
		defer close(errCh)
		defer c.Close()

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				// websocket 关闭时正常结束
				return
			}

			var resp relayTTSResponse
			if err := json.Unmarshal(message, &resp); err != nil {
				t.l.Error("unmarshal fail: ", log.Error(err))
				continue
			}

			if resp.Data != "" {
				raw, err := base64.StdEncoding.DecodeString(resp.Data)
				if err != nil {
					t.l.Error("decode fail: ", log.Error(err))
					continue
				}

				// 转换成 []int16
				samples := make([]int16, len(raw)/2)
				_ = binary.Read(
					bytes.NewReader(raw),
					binary.LittleEndian,
					&samples,
				)

				out <- PCMChunk{
					Seq:     resp.Sequence,
					Samples: samples,
				}
			}

			// Sequence < 0 表示流结束
			if resp.Sequence < 0 {
				return
			}
		}
	}()

	return out, errCh
}
