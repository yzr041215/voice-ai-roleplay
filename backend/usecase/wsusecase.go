package usecase

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/usecase/utils"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type WsUseCase struct {
	logger      *log.Logger
	config      *config.Config
	asrusecase  *AsrUsecase
	llmusecase  *LlmUsecase
	fileusecase *FileUsecase
}

func NewWsUsecase(l *log.Logger, c *config.Config, asr *AsrUsecase, llm *LlmUsecase, file *FileUsecase) *WsUseCase {
	return &WsUseCase{
		logger:      l,
		config:      c,
		asrusecase:  asr,
		llmusecase:  llm,
		fileusecase: file,
	}

}

// WsStatePayload 前端解析用
type WsStatePayload struct {
	State string `json:"state"`
	IsVad bool   `json:"isVad"`
	SegID int    `json:"seg_id,omitempty"`
}

// AsrResultPayload ASR 文本数据结构
type AsrResultPayload struct {
	Text    string `json:"text"`
	SegID   int    `json:"seg_id"`
	FileURL string `json:"file_url,omitempty"`
}

// helper: 把 usecase.VadState 转成字符串
func vadStateToString(s VadState) string {
	switch s {
	case StateIdle:
		return "idle"
	case StateListening:
		return "listening"
	case StateProcessing:
		return "processing"
	case StateResponding:
		return "responding"
	default:
		return "unknown"
	}
}

// HanderWs2 使用 VadManager 与 domain.Msg 完成全流程
func (w *WsUseCase) HanderWs2(ws *websocket.Conn, userid string, roleid int) error {
	w.logger.Info("new ws connection (HanderWs2)", log.String("userid", userid))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// channel: 音频数据推给 VAD
	audioChan := make(chan []byte, 200)
	defer close(audioChan)

	// channel: 从 VAD 收到 ASR 结果（缓冲防止阻塞）
	resultChan := make(chan ASRResult, 8)
	defer close(resultChan)

	// 创建 VadManager，回调用于推送状态给前端
	var vadMgr *VadManager

	vadMgr = NewVadManagerWithResult(
		w.logger,
		w.asrusecase,
		w.fileusecase,
		w.config,
		resultChan,
		func(st VadState) {
			payload := WsStatePayload{
				State: vadStateToString(st),
				IsVad: vadMgrIsVadSafe(vadMgr), // 这里才用到 vadMgr
			}
			b, _ := json.Marshal(payload)
			msg := &domain.Msg{Type: domain.MsgTypeState, Data: b}
			if data, err := msg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}
		},
	)
	defer vadMgr.Close()

	// 启动 vad 处理（后台 goroutine）
	go func() {
		_ = vadMgr.ProcessAudioStream(ctx, audioChan)
	}()

	// responseCancel 管理当前正在处理的 LLM->TTS 的取消函数（单个会话串行）
	var responseCancelMu sync.Mutex
	var responseCancel func()

	// helper：开始处理一个 ASR 结果（串行处理 resultChan 的每个消息）
	go func() {
		for asr := range resultChan {
			// 每次开始处理新的 ASR 时，确保没有旧的 responseCancel 未清理
			responseCancelMu.Lock()
			if responseCancel != nil {
				// 如果已有旧的处理（理论上不应该，因为 vadMgr 进入 Responding 会阻止新段产生）
				// 先取消旧的，以防僵尸
				responseCancel()
				responseCancel = nil
			}
			// 创建响应上下文，可被打断（interrupt）
			respCtx, cancelFn := context.WithCancel(context.Background())
			responseCancel = cancelFn
			responseCancelMu.Unlock()

			// 1) 向前端发送 ASR 结果（文本展示）
			asrPayload := AsrResultPayload{
				Text:    asr.Text,
				SegID:   asr.SegID,
				FileURL: asr.FileURL,
			}
			b, _ := json.Marshal(asrPayload)
			msg := &domain.Msg{Type: domain.MsgTypeAsrResult, Data: b}
			if data, err := msg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

			// 2) LLM 生成回复（参考你原 HanderWs）
			ms, err := w.llmusecase.FormatMessage(respCtx, userid, roleid, asr.Text)
			if err != nil {
				w.logger.Error("format message failed", log.Error(err))
				// 恢复 VAD 并清理 responseCancel
				vadMgr.OnResponseDone()
				responseCancelMu.Lock()
				responseCancel = nil
				responseCancelMu.Unlock()
				continue
			}
			anCh, err := w.llmusecase.Chat(respCtx, ms)
			if err != nil {
				w.logger.Error("llm chat failed", log.Error(err))
				vadMgr.OnResponseDone()
				responseCancelMu.Lock()
				responseCancel = nil
				responseCancelMu.Unlock()
				continue
			}

			// 3) TTS 流式合成并推给前端
			tts := utils.NewTtsStream(w.logger, w.config)
			// 传入 respCtx，方便外部 cancel
			pcmStream, errCh := tts.TtsStream(respCtx, anCh, "qiniu_zh_female_tmjxxy")

			// 发送 tts_start 事件
			startMsg := &domain.Msg{Type: domain.MsgTypeTtsStart, Data: []byte(`{}`)}
			if data, err := startMsg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

			// 读流并发送 PCM（二进制）; 任何错误或 ctx cancel 都会中断
			sendErr := false
		LOOP:
			for {
				select {
				case <-respCtx.Done():
					// 被上层打断
					w.logger.Info("response ctx canceled (interrupt)")
					sendErr = true
					break LOOP
				case err := <-errCh:
					if err != nil {
						w.logger.Error("tts stream error", log.Error(err))
					}
					// tts 测试流结束或发生错误
					break LOOP
				case pcm, ok := <-pcmStream:
					if !ok {
						// 正常结束
						break LOOP
					}
					// 将 int16 samples 转为 bytes（与你现有逻辑一致）
					results := make([]byte, len(pcm.Samples)*2) // int16 -> 2 bytes each (注意：之前示例直接 byte(s) 可能丢精度)
					// 我这里采用小端编码，将 int16 写成两字节
					for i, s := range pcm.Samples {
						// little endian
						results[2*i] = byte(s)
						results[2*i+1] = byte(s >> 8)
					}
					// 发送二进制 PCM
					if err := ws.WriteMessage(websocket.BinaryMessage, results); err != nil {
						w.logger.Error("write pcm to ws failed", log.Error(err))
						sendErr = true
						break LOOP
					}
					// 可选：也发送一个 TtsChunk 事件（meta）
					meta := &domain.Msg{Type: domain.MsgTypeTtsChunk, Data: []byte(`{}`)}
					if md, err := meta.Encode(); err == nil {
						_ = ws.WriteMessage(websocket.TextMessage, md)
					}
				}
			}

			// 发送 tts_end（无论是正常结束还是中断）
			endMsg := &domain.Msg{Type: domain.MsgTypeTtsEnd, Data: []byte(`{}`)}
			if data, err := endMsg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

			// 清理 responseCancel 并让 VAD 恢复 Idle（即允许新一轮语音）
			responseCancelMu.Lock()
			if responseCancel != nil {
				// 如果是因为正常结束，respCtx 已经 Done；如果因外部中断，这里同样清理
				responseCancel()
				responseCancel = nil
			}
			responseCancelMu.Unlock()

			// 让 VadManager 进入 Idle（等待新段）
			vadMgr.OnResponseDone()

			// 如果发送出错，可能需要断开连接或记录
			if sendErr {
				// 如果需要断开 ws，取消最外层 ctx
				// cancel()
				// 但在这里我们仅记录并继续
				w.logger.Info("tts send ended with sendErr")
			}
		}
	}()

	// 主读循环：接收前端消息（全部使用 domain.Msg 格式）
	for {
		t, raw, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		switch t {
		case websocket.BinaryMessage:
			// 音频帧推到 audioChan（非阻塞）
			select {
			case audioChan <- raw:
			default:
				// 丢帧（channel 满）以保证不会阻塞
			}

			// 每帧同时回传当前状态（state + isVad），利用 domain.MsgTypeState
			statePayload := WsStatePayload{
				State: vadStateToString(vadMgr.GetState()),
				IsVad: vadMgr.IsVad(),
			}
			b, _ := json.Marshal(statePayload)
			msg := &domain.Msg{Type: domain.MsgTypeState, Data: b}
			if data, err := msg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

		case websocket.TextMessage:
			// 所有文本消息必须为 domain.Msg 格式
			m, derr := domain.Decode(raw)
			if derr != nil {
				// 非 domain.Msg，忽略或发错误
				errPayload := []byte(`{"error":"invalid message format"}`)
				errMsg := &domain.Msg{Type: domain.MsgTypeError, Data: errPayload}
				if data, e := errMsg.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			switch m.Type {
			case domain.MsgTypeIntrupt:
				// 客户端发起打断：取消当前正在进行的 LL M/TTS（如果有）
				responseCancelMu.Lock()
				if responseCancel != nil {
					responseCancel() // 触发 respCtx.Done()，上面的 goroutine 会处理清理与恢复
					responseCancel = nil
				} else {
					// 若没有正在处理，我们也令 VAD 回 Idle（保险）
					vadMgr.OnResponseDone()
				}
				responseCancelMu.Unlock()

				// 向前端回 ack
				ack := &domain.Msg{Type: domain.MsgTypeIntrupt, Data: []byte(`{"ack":true}`)}
				if data, e := ack.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}

			case domain.MsgTypeTranslate:
				// 如果前端发送 "translate" 类型（可能是客户端文本输入），你可以把它当作即时文本处理。
				// 这里只是示例：把文本再发回前端确认
				_ = ws.WriteMessage(websocket.TextMessage, raw)

			default:
				// 未知类型：忽略或回 error
				errPayload := []byte(`{"error":"unsupported msg type"}`)
				errMsg := &domain.Msg{Type: domain.MsgTypeError, Data: errPayload}
				if data, e := errMsg.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
			}
		}
	}
}

// 辅助函数：安全读取 vadMgr.IsVad（可能 nil）
func vadMgrIsVadSafe(v *VadManager) bool {
	if v == nil {
		return false
	}
	return v.IsVad()
}

func (w *WsUseCase) HanderWs(ws *websocket.Conn, userid string, roleid int) error {
	w.logger.Info("new ws connection (HanderWs)", log.String("userid", userid), log.Int("roleid", roleid))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 音频帧输入给 ASR
	pcmChan := make(chan []byte, 200)
	defer close(pcmChan)

	// responseCancel 控制当前 LLM->TTS 的取消（单个串行响应）
	var responseCancelMu sync.Mutex
	var responseCancel func()

	// 启动流式 ASR（在后台 goroutine）
	go func() {
		s := utils.NewAsrUsecase(w.logger, w.config)
		err := s.AsrStream(ctx, pcmChan, func(text string, isFinal bool) {
			// 收到 ASR 中间或最终结果

			// 记录 ASR 到日志，方便排查
			w.logger.Info("asr callback", log.String("text", text), log.Any("isFinal", isFinal))

			// 推送 ASR 结果给前端（无论中间/最终都推送，前端可决定展示）
			asrPayload := AsrResultPayload{
				Text:  text,
				SegID: 0,
			}
			if b, _ := json.Marshal(asrPayload); true {
				msg := &domain.Msg{Type: domain.MsgTypeAsrResult, Data: b}
				if data, err := msg.Encode(); err == nil {
					if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
						w.logger.Error("write asr_result to ws failed", log.Error(err))
					}
				}
			}

			// 仅在最终结果时触发 LLM -> 合句 -> TTS 流式合成
			if !isFinal {
				return
			}

			// 新的最终结果到来，先取消可能存在的旧响应（打断旧的 LLM/TTS）
			responseCancelMu.Lock()
			if responseCancel != nil {
				w.logger.Info("cancel previous response (new final asr arrived)")
				responseCancel()
				responseCancel = nil
			}
			respCtx, cancelFn := context.WithCancel(context.Background())
			responseCancel = cancelFn
			responseCancelMu.Unlock()

			// 格式化并调用 LLM（返回 token 流 channel <-chan string）
			ms, err := w.llmusecase.FormatMessage(respCtx, userid, roleid, text)
			if err != nil {
				w.logger.Error("format message failed", log.Error(err))
				// 清理
				responseCancelMu.Lock()
				if responseCancel != nil {
					responseCancel()
					responseCancel = nil
				}
				responseCancelMu.Unlock()
				return
			}
			tokenCh, err := w.llmusecase.Chat(respCtx, ms)
			if err != nil {
				w.logger.Error("llm chat failed", log.Error(err))
				responseCancelMu.Lock()
				if responseCancel != nil {
					responseCancel()
					responseCancel = nil
				}
				responseCancelMu.Unlock()
				return
			}

			// 合句：把 token 流合并为句子流（遇标点或超时 flush）
			sentenceCh := utils.MergeSentences(respCtx, tokenCh) // <-chan string

			// 调用 TTS：输入 sentenceCh（句子），输出 PCMChunk channel
			tts := utils.NewTtsStream(w.logger, w.config)
			pcmStream, errCh := tts.TtsStream(respCtx, sentenceCh, "qiniu_zh_female_tmjxxy")

			// 发送 tts_start 事件（前端可据此清 UI）
			startMsg := &domain.Msg{Type: domain.MsgTypeTtsStart, Data: []byte(`{}`)}
			if data, err := startMsg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

			// 消费 PCMChunk 流：把每个 PCMChunk 序列化成内层 JSON，然后放到 domain.Msg.Data 字段里发给前端
			// 内层 JSON 结构： {"seq": <int>, "pcm": "<base64 of raw pcm bytes>", "text": "<sentence>"}
			seqCounter := 0
		PCM_LOOP:
			for {
				select {
				case <-respCtx.Done():
					w.logger.Info("respCtx done -> stop sending tts")
					break PCM_LOOP
				case terr := <-errCh:
					// errCh 可能会被写入错误或被关闭; 这里处理错误（如果非 nil 就记录）
					if terr != nil {
						w.logger.Error("tts.TtsStream error", log.Error(terr))
					}
					// 无论是否有错，退出本次 TTS 循环（流结束）
					break PCM_LOOP
				case pcmChunk, ok := <-pcmStream:
					if !ok {
						// tts 输出通道关闭 => 正常结束
						break PCM_LOOP
					}
					// 转 int16 samples -> []byte (little endian)
					raw := make([]byte, len(pcmChunk.Samples)*2)
					for i, s := range pcmChunk.Samples {
						binary.LittleEndian.PutUint16(raw[2*i:], uint16(s))
					}

					// base64 编码 raw bytes 放到内层 JSON
					encPCM := base64.StdEncoding.EncodeToString(raw)
					seqCounter++
					payload := map[string]interface{}{
						"seq":  seqCounter,
						"pcm":  encPCM,
						"text": "", // 可选：你也可以传回 sentence（若需要需把 sentenceCh 的句子传到这里）
					}
					// 如果你需要把句子文本也回传给前端，需要在 MergeSentences -> TtsStream 流程中保留映射。
					plB, _ := json.Marshal(payload)

					chunkMsg := &domain.Msg{Type: domain.MsgTypeTtsChunk, Data: plB}
					if md, err := chunkMsg.Encode(); err == nil {
						if err := ws.WriteMessage(websocket.TextMessage, md); err != nil {
							w.logger.Error("write tts_chunk to ws failed", log.Error(err))
							// 如果写失败，可能客户端断开，结束发送
							break PCM_LOOP
						}
					}
					// 你也可以同时发送裸 Binary PCM（如果前端支持直接播放裸二进制）：
					// _ = ws.WriteMessage(websocket.BinaryMessage, raw)
				}
			}

			// TTS 完成，发送 tts_end
			endMsg := &domain.Msg{Type: domain.MsgTypeTtsEnd, Data: []byte(`{}`)}
			if data, err := endMsg.Encode(); err == nil {
				_ = ws.WriteMessage(websocket.TextMessage, data)
			}

			// 清理当前响应取消器（respCtx）——如果尚未清理
			responseCancelMu.Lock()
			if responseCancel != nil {
				responseCancel()
				responseCancel = nil
			}
			responseCancelMu.Unlock()
		})
		if err != nil {
			w.logger.Error("asr stream failed", log.Error(err))
		}
	}()

	// 主循环：接收前端消息（音频帧、打断等）
	for {
		t, raw, err := ws.ReadMessage()
		if err != nil {
			return err
		}
		switch t {
		case websocket.BinaryMessage:
			// 音频帧 push 给 ASR
			select {
			case pcmChan <- raw:
			default:
				// channel 满了就丢帧，避免阻塞
				w.logger.Warn("pcmChan full, drop frame")
			}
		case websocket.TextMessage:
			// 文本消息应为 domain.Msg 格式
			m, derr := domain.Decode(raw)
			if derr != nil {
				errMsg := &domain.Msg{Type: domain.MsgTypeError, Data: []byte(`{"error":"invalid msg"}`)}
				if data, e := errMsg.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
				continue
			}

			switch m.Type {
			case domain.MsgTypeIntrupt:
				// 前端发起中断：取消正在进行的 LLM/TTS
				responseCancelMu.Lock()
				if responseCancel != nil {
					responseCancel()
					responseCancel = nil
					w.logger.Info("client intrupt: cancelled current response")
				}
				responseCancelMu.Unlock()
				ack := &domain.Msg{Type: domain.MsgTypeIntrupt, Data: []byte(`{"ack":true}`)}
				if data, e := ack.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
			case domain.MsgTypeTranslate:
				// 前端文本直接转发或处理（按需）
				_ = ws.WriteMessage(websocket.TextMessage, raw)
			default:
				errMsg := &domain.Msg{Type: domain.MsgTypeError, Data: []byte(`{"error":"unsupported msg type"}`)}
				if data, e := errMsg.Encode(); e == nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
			}
		}
	}
}
