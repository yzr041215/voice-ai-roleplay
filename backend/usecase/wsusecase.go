package usecase

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/usecase/utils"
	"fmt"

	"github.com/gorilla/websocket"
)

type WsUseCase struct {
	logger      *log.Logger
	config      *config.Config
	asrusecase  *AsrUsecase
	llmusecase  *LlmUsecase
	fileusecase *FileUsecase
}

func NewWsUseCase(l *log.Logger, c *config.Config, asrusecase *AsrUsecase, llmusecase *LlmUsecase, fileusecase *FileUsecase) *WsUseCase {
	return &WsUseCase{
		logger:      l,
		config:      c,
		asrusecase:  asrusecase,
		llmusecase:  llmusecase,
		fileusecase: fileusecase,
	}
}
func (w *WsUseCase) HanderWss(ws *websocket.Conn, userid string, roleid int) error {
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
			//w.logger.Info("rece bin data", log.Int("len",len(msg)))
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

func (w *WsUseCase) HanderWs(ws *websocket.Conn, userid string, roleid int) error {
	w.logger.Info("new ws connection", log.String("userid", userid), log.Int("roleid", roleid))
	s := utils.NewAsrUsecase(w.logger, w.config)
	ctx := context.Background()
	pcmChan := make(chan []byte)
	go func() {
		err := s.AsrStream(ctx, pcmChan, func(text string, isFinal bool) {

			if isFinal {
				w.logger.Info("final result", log.String("text", text))
				ms, err := w.llmusecase.FormatMessage(ctx, userid, roleid, text)
				if err != nil {
					return
				}
				anCh, err := w.llmusecase.Chat(ctx, ms)
				if err != nil {
					return
				}
				tts := utils.NewTtsStream(w.logger, w.config)
				pcmStream, errCh := tts.TtsStream(ctx, anCh, "qiniu_zh_female_tmjxxy")
				for {
					select {
					case pcm, ok := <-pcmStream:
						if !ok {
							fmt.Println("流结束")
							return
						}
						fmt.Printf("seq=%d pcm_samples=%d\n", pcm.Seq, len(pcm.Samples))
						results := make([]byte, len(pcm.Samples))
						for i, s := range pcm.Samples {
							results[i] = byte(s)
						}
						ws.WriteMessage(websocket.BinaryMessage, results)
						// 在这里写入 wav 文件，或者直接送给音频播放设备
					case err := <-errCh:
						if err != nil {
							fmt.Println("错误:", err)
							return
						}
					}
				}
			}
			m := &domain.Msg{
				Type: domain.MsgTypeTanslate,
				Data: []byte(text),
			}
			w.logger.Info("send ws message", log.String("text", text))
			data, _ := m.Encode()
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				w.logger.Error("send ws message failed", log.Error(err))
				return
			}

		})
		if err != nil {
			fmt.Println("ASR 出错:", err)
		}
	}()

	for {
		t, msg, err := ws.ReadMessage()
		if err != nil {
			return err
		}

		switch t {
		case websocket.BinaryMessage:
			// 音频帧 (640字节，PCM16 16kHz 单声道 20ms)
			pcmChan <- msg
			//w.logger.Info("rece bin data", log.Int("len",len(msg)))
			// 每帧检查 vad 状态，通知前端

		case websocket.TextMessage:
			// 文本消息 (打断信号)
			// 前端发来 "stop" 之类的控制命令
			if string(msg) == "stop" {
				w.logger.Info("receive stop signal")
			}
		}
	}
}
