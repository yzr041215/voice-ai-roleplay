package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"demo/config"
	"demo/pkg/log"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AsrUsecase 提供语音识别服务（流式）
type AsrUsecase struct {
	l      *log.Logger
	config *config.Config
}

func NewAsrUsecase(l *log.Logger, c *config.Config) *AsrUsecase {
	return &AsrUsecase{
		l:      l.WithModule("AsrUsecase"),
		config: c,
	}
}

// --- 协议工具 ---
func (a *AsrUsecase) generateHeader(messageType, flags, serial, compress int) []byte {
	header := make([]byte, 4)
	header[0] = (1 << 4) | 1
	header[1] = byte((messageType << 4) | flags)
	header[2] = byte((serial << 4) | compress)
	header[3] = 0
	return header
}

func (a *AsrUsecase) generateBeforePayload(seq int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(seq))
	return buf
}

func gzipData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sendConfig 与 sendAudioChunk（seq 自增使用 (*seq)++）
func (a *AsrUsecase) sendConfig(ws *websocket.Conn, seq *int) error {
	req := map[string]interface{}{
		"user": map[string]string{"uid": uuid.NewString()},
		"audio": map[string]interface{}{
			"format":      "pcm",
			"sample_rate": 16000,
			"bits":        16,
			"channel":     1,
			"codec":       "raw",
		},
		"request": map[string]interface{}{
			"model_name":  "asr",
			"enable_punc": true,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	payload, err = gzipData(payload)
	if err != nil {
		return err
	}

	(*seq)++
	var msg bytes.Buffer
	msg.Write(a.generateHeader(1, 1, 1, 1))
	msg.Write(a.generateBeforePayload(*seq))
	msg.Write(make([]byte, 4)) // 占位
	msg.Write(payload)

	if len(msg.Bytes()) >= 12 {
		binary.BigEndian.PutUint32(msg.Bytes()[8:12], uint32(len(payload)))
	}
	return ws.WriteMessage(websocket.BinaryMessage, msg.Bytes())
}

func (a *AsrUsecase) sendAudioChunk(ws *websocket.Conn, seq *int, pcm []byte) error {
	(*seq)++
	compressed, err := gzipData(pcm)
	if err != nil {
		return err
	}

	var msg bytes.Buffer
	msg.Write(a.generateHeader(2, 1, 1, 1))
	msg.Write(a.generateBeforePayload(*seq))
	msg.Write(make([]byte, 4)) // 占位
	msg.Write(compressed)

	if len(msg.Bytes()) >= 12 {
		binary.BigEndian.PutUint32(msg.Bytes()[8:12], uint32(len(compressed)))
	}
	return ws.WriteMessage(websocket.BinaryMessage, msg.Bytes())
}

// parseTextFromResponse 解析并返回 (text, isFinal)
func (a *AsrUsecase) parseTextFromResponse(data []byte) (string, bool) {
	if len(data) < 4 {
		return "", false
	}

	headerSize := int(data[0] & 0x0f)
	headerBytes := headerSize * 4
	if headerBytes <= 0 || headerBytes > len(data) {
		return "", false
	}

	payload := data[headerBytes:]
	if len(payload) == 0 {
		return "", false
	}

	messageTypeSpecificFlags := data[1] & 0x0f
	if (messageTypeSpecificFlags & 0x01) != 0 {
		if len(payload) < 4 {
			return "", false
		}
		payload = payload[4:]
		if len(payload) == 0 {
			return "", false
		}
	}

	if len(payload) >= 4 {
		payloadSize := binary.BigEndian.Uint32(payload[:4])
		if payloadSize > 0 {
			totalNeeded := int(4 + payloadSize)
			if totalNeeded <= len(payload) {
				payload = payload[4 : 4+payloadSize]
			} else {
				payload = payload[4:]
			}
		} else {
			payload = payload[4:]
		}
	}

	messageCompression := data[2] & 0x0f
	if (messageCompression & 0x01) != 0 {
		if r, err := gzip.NewReader(bytes.NewReader(payload)); err == nil {
			if unzipped, err2 := io.ReadAll(r); err2 == nil {
				_ = r.Close()
				payload = unzipped
			} else {
				_ = r.Close()
			}
		}
	}

	if len(payload) == 0 {
		return "", false
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(payload, &obj); err == nil {
		if r, ok := obj["result"].(map[string]interface{}); ok {
			txt := ""
			if t, ok2 := r["text"].(string); ok2 {
				txt = t
			} else if t2, ok3 := r["text"]; ok3 {
				txt = fmt.Sprint(t2)
			}
			if t, ok := r["type"].(string); ok && strings.EqualFold(t, "final") {
				return txt, true
			}
			if f, ok := r["is_final"].(bool); ok && f {
				return txt, true
			}
			if s, ok := r["status"]; ok {
				if sf, ok2 := s.(float64); ok2 && int(sf) == 2 {
					return txt, true
				}
				if ss, ok3 := s.(string); ok3 &&
					(strings.Contains(strings.ToLower(ss), "final") || strings.Contains(strings.ToLower(ss), "completed")) {
					return txt, true
				}
			}
			return txt, false
		}
		if pm, ok := obj["payload_msg"].(map[string]interface{}); ok {
			if r, ok := pm["result"].(map[string]interface{}); ok {
				txt := ""
				if t, ok2 := r["text"].(string); ok2 {
					txt = t
				} else if t2, ok3 := r["text"]; ok3 {
					txt = fmt.Sprint(t2)
				}
				if f, ok := r["is_final"].(bool); ok && f {
					return txt, true
				}
				if t, ok := r["type"].(string); ok && strings.EqualFold(t, "final") {
					return txt, true
				}
				return txt, false
			}
		}
		if s, ok := obj["text"].(string); ok {
			return s, false
		}
	} else {
		str := strings.TrimSpace(string(payload))
		if str != "" {
			return str, false
		}
	}

	return "", false
}

// AsrStream 流式 ASR (带停顿触发 final)
func (a *AsrUsecase) AsrStream(ctx context.Context, pcmStream <-chan []byte, onResult func(text string, isFinal bool)) error {
	u := "wss://openai.qiniu.com/v1/voice/asr"
	header := http.Header{}
	header.Set("Authorization", "Bearer "+a.config.Asr.ApiKey)

	ws, _, err := websocket.DefaultDialer.DialContext(ctx, u, header)
	if err != nil {
		return fmt.Errorf("dial websocket fail: %w", err)
	}
	defer ws.Close()
	a.l.Info("asr websocket connected")

	seq := 0
	if err := a.sendConfig(ws, &seq); err != nil {
		return fmt.Errorf("send config fail: %w", err)
	}

	partialCh := make(chan string, 8)
	errCh := make(chan error, 1)
	forceFinalCh := make(chan struct{}, 1)

	// debounce goroutine：处理 partial → final + 静音触发 final
	go func() {
		var pending string
		var lastFinal string
		var lastPartial string
		timer := time.NewTimer(0)
		<-timer.C // 清掉初始触发

		for {
			select {
			case p, ok := <-partialCh:
				if !ok {
					if pending != "" && pending != lastFinal {
						onResult(pending, true)
					}
					return
				}
				pending = p
				if pending != lastPartial {
					onResult(pending, false)
					lastPartial = pending
				}
				// 重置静音触发 final
				timer.Reset(500 * time.Millisecond)

			case <-timer.C:
				if pending != "" && pending != lastFinal {
					onResult(pending, true)
					lastFinal = pending
					pending = ""
					lastPartial = ""
				}
				timer.Reset(500 * time.Millisecond)

			case <-forceFinalCh:
				if pending != "" && pending != lastFinal {
					onResult(pending, true)
					lastFinal = pending
					pending = ""
					lastPartial = ""
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// 读取服务端返回
	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			text, isFinal := a.parseTextFromResponse(msg)
			if text == "" {
				continue
			}
			if isFinal {
				onResult(text, true)
				select {
				case partialCh <- "":
				default:
				}
			} else {
				select {
				case partialCh <- text:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// 主发送循环
	for {
		select {
		case <-ctx.Done():
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			close(partialCh)
			return ctx.Err()
		case err := <-errCh:
			close(partialCh)
			return fmt.Errorf("ws read loop error: %w", err)
		case chunk, ok := <-pcmStream:
			if !ok {
				_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				close(partialCh)
				return nil
			}
			if err := a.sendAudioChunk(ws, &seq, chunk); err != nil {
				close(partialCh)
				return fmt.Errorf("send audio chunk fail: %w", err)
			}
			// ⚡ 每次发送音频都尝试重置静音 final
			select {
			case forceFinalCh <- struct{}{}:
			default:
			}
		}
	}
}
