package usecase

import (
	"context"
	"demo/config"
	"demo/pkg/log"
	"fmt"

	"github.com/gorilla/websocket"
)

type WsUseCase struct {
	logger      *log.Logger
	config      *config.Config
	asrusecase  *AsrUsecase
	ttsusecase  *TtsUsecase
	llmusecase  *LlmUsecase
	fileusecase *FileUsecase
}

func NewWsUseCase(l *log.Logger, c *config.Config, asrusecase *AsrUsecase, ttsusecase *TtsUsecase, llmusecase *LlmUsecase, fileusecase *FileUsecase) *WsUseCase {
	return &WsUseCase{
		logger:      l,
		config:      c,
		asrusecase:  asrusecase,
		ttsusecase:  ttsusecase,
		llmusecase:  llmusecase,
		fileusecase: fileusecase,
	}
}
func (w *WsUseCase) HanderWs(ws *websocket.Conn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	audioChan := make(chan []byte, 100)
	defer close(audioChan)

	// 创建 VadManager
	vadMgr := NewVadManager(w.logger, w.asrusecase, w.fileusecase, w.config)
	defer vadMgr.Close()

	// 在后台跑 VAD 处理
	go func() {
		_ = vadMgr.ProcessAudioStream(ctx, audioChan)
	}()

	for {
		t, msg, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		switch t {
		case websocket.BinaryMessage:
			// 音频帧 (640字节，PCM16 16kHz 单声道 20ms)
			audioChan <- msg

			// 每帧检查 vad 状态，通知前端
			isVad := vadMgr.IsVad()
			resp := fmt.Sprintf(`{"isVad":%v}`, isVad)
			if err := ws.WriteMessage(websocket.TextMessage, []byte(resp)); err != nil {
				w.logger.Error("send ws message failed", log.Error(err))
			}

		case websocket.TextMessage:
			// 文本消息 (打断信号)
			// 前端发来 "stop" 之类的控制命令
			if string(msg) == "stop" {
				w.logger.Info("receive stop signal")
				cancel() // 结束音频处理
			}
		}
	}
}
