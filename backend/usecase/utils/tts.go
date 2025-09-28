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
	"strings"
	"sync"
	"time"

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

// --- 核心：合句逻辑 ---
// 收到 LLM 的 token 流，先合成句子再发给 TTS
func MergeSentences(ctx context.Context, tokens <-chan string) <-chan string {
	out := make(chan string, 8)

	go func() {
		defer close(out)

		var buf strings.Builder
		timer := time.NewTimer(2 * time.Second) // 超时强制触发
		defer timer.Stop()

		flush := func() {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				out <- s
			}
			buf.Reset()
		}

		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case tk, ok := <-tokens:
				if !ok {
					flush()
					return
				}
				buf.WriteString(tk)

				// 如果遇到标点或句子结束符 -> 立刻 flush
				if strings.ContainsAny(tk, "。！？!?") {
					flush()
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(2 * time.Second)
				} else {
					// reset timer
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(2 * time.Second)
				}
			case <-timer.C:
				flush()
				timer.Reset(2 * time.Second)
			}
		}
	}()

	return out
}

// --- TTS 调用 ---
func (t *TtsStream) TtsStream(
	ctx context.Context,
	textChunks <-chan string, // 输入句子块
	voiceType string,
) (<-chan PCMChunk, <-chan error) {

	out := make(chan PCMChunk, 16)
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
					Encoding:   "pcm",
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

	// 读取协程（并发 PCM 解码 + 顺序输出）
	go func() {
		defer close(out)
		defer close(errCh)
		defer c.Close()

		var mu sync.Mutex
		seqBuf := make(map[int][]int16)
		expectSeq := 0

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
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

				// 并发解码
				go func(seq int, raw []byte) {
					samples := make([]int16, len(raw)/2)
					_ = binary.Read(bytes.NewReader(raw), binary.LittleEndian, &samples)

					mu.Lock()
					seqBuf[seq] = samples

					// 顺序发送
					for {
						if pcm, ok := seqBuf[expectSeq]; ok {
							out <- PCMChunk{Seq: expectSeq, Samples: pcm}
							delete(seqBuf, expectSeq)
							expectSeq++
						} else {
							break
						}
					}
					mu.Unlock()
				}(resp.Sequence, raw)
			}

			if resp.Sequence < 0 {
				return
			}
		}
	}()

	return out, errCh
}
