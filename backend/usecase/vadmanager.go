package usecase

import (
	"bytes"
	"context"
	"demo/config"
	"demo/pkg/log"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/baabaaox/go-webrtcvad"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// 常量
const (
	SampleRate    = 16000
	FrameDuration = 20
	FrameSize     = SampleRate / 1000 * FrameDuration
	BitDepth      = 16
	BytesPerFrame = FrameSize * BitDepth / 8
	SilenceFrames = 50
)

// 状态枚举（导出用于 handler 中判断）
type VadState int

const (
	StateIdle VadState = iota
	StateListening
	StateProcessing
	StateResponding
)

// ASRResult 发送到上层 handler 用
type ASRResult struct {
	Text    string
	SegID   int
	FileURL string
}

// StateChangeFn 当状态变化时回调（上层可把状态推给前端）
type StateChangeFn func(st VadState)

// VadManager 结构体（导出）
type VadManager struct {
	logger      *log.Logger
	asrUsecase  *AsrUsecase
	fileUsecase *FileUsecase
	config      *config.Config

	vad          webrtcvad.VadInst
	mu           sync.Mutex // 保护 currSeg/vadActive/silenceCount
	currSeg      [][]byte
	segID        int
	silenceCount int
	vadActive    bool

	state   VadState
	stateMu sync.Mutex

	// 通信
	resultChan    chan<- ASRResult
	onStateChange StateChangeFn
}

// NewVadManagerWithResult 创建实例
func NewVadManagerWithResult(
	logger *log.Logger,
	asrUsecase *AsrUsecase,
	fileUsecase *FileUsecase,
	config *config.Config,
	resultChan chan<- ASRResult,
	onStateChange StateChangeFn,
) *VadManager {
	vad := webrtcvad.Create()
	if vad == nil {
		panic("failed to create vad instance")
	}
	if err := webrtcvad.Init(vad); err != nil {
		panic(err)
	}
	if err := webrtcvad.SetMode(vad, 3); err != nil {
		panic(err)
	}
	return &VadManager{
		logger:        logger.WithModule("VadManager"),
		asrUsecase:    asrUsecase,
		fileUsecase:   fileUsecase,
		config:        config,
		vad:           vad,
		state:         StateIdle,
		resultChan:    resultChan,
		onStateChange: onStateChange,
	}
}

func (v *VadManager) Close() {
	if v.vad != nil {
		webrtcvad.Free(v.vad)
		v.vad = nil
	}
}

// IsVad 当前是否处于语音段（线程安全）
func (v *VadManager) IsVad() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.vadActive
}

// GetState 获取状态（线程安全）
func (v *VadManager) GetState() VadState {
	v.stateMu.Lock()
	defer v.stateMu.Unlock()
	return v.state
}

// setState 内部使用，并触发回调（非阻塞）
func (v *VadManager) setState(s VadState) {
	v.stateMu.Lock()
	old := v.state
	if old == s {
		v.stateMu.Unlock()
		return
	}
	v.state = s
	v.stateMu.Unlock()

	if v.onStateChange != nil {
		go func() {
			// recover 防止回调 panic
			defer func() {
				if r := recover(); r != nil {
					v.logger.Error("state callback panic")
				}
			}()
			v.onStateChange(s)
		}()
	}
	v.logger.Info("vad state changed", log.Int("from", int(old)), log.Int("to", int(s)))
}

// OnResponseDone 由上层在 TTS 播报完成或中断后调用，使状态回到 Idle
func (v *VadManager) OnResponseDone() {
	v.setState(StateIdle)
}

// ProcessAudioStream 音频处理主循环（读 channel）
// 注意：当处于 Processing/Responding 时，新的帧会被丢弃
func (v *VadManager) ProcessAudioStream(ctx context.Context, audioChunks <-chan []byte) error {
	eg, ctx := errgroup.WithContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-audioChunks:
			if !ok {
				return eg.Wait()
			}

			// 如果在处理或回答，丢弃帧以避免并发识别
			st := v.GetState()
			if st == StateProcessing || st == StateResponding {
				continue
			}

			if len(chunk) != BytesPerFrame {
				v.logger.Warn("invalid frame size", log.Int("got", len(chunk)), log.Int("want", BytesPerFrame))
				continue
			}

			active, err := webrtcvad.Process(v.vad, SampleRate, chunk, FrameSize)
			if err != nil {
				v.logger.Error("vad process error", log.Error(err))
				continue
			}

			v.mu.Lock()
			if active {
				if v.GetState() == StateIdle {
					v.setState(StateListening)
				}
				v.vadActive = true
				v.currSeg = append(v.currSeg, chunk)
				v.silenceCount = 0
			} else if v.vadActive {
				v.silenceCount++
				v.currSeg = append(v.currSeg, chunk)

				if v.silenceCount >= SilenceFrames {
					// 断句：把当前段复制并异步处理
					chunks := make([][]byte, len(v.currSeg))
					copy(chunks, v.currSeg)
					v.segID++
					segID := v.segID

					v.setState(StateProcessing)
					eg.Go(func() error {
						return v.handleSegment(ctx, segID, chunks)
					})

					// reset
					v.currSeg = nil
					v.vadActive = false
					v.silenceCount = 0
				}
			}
			v.mu.Unlock()
		}
	}
}

// handleSegment 上传 WAV 并调用 asrUsecase.Asr，然后通过 resultChan 抛出结果，最后切到 Responding
func (v *VadManager) handleSegment(ctx context.Context, segID int, seg [][]byte) error {
	var buf bytes.Buffer
	dataSize := int64(len(seg) * BytesPerFrame)
	if err := writeWavHeader(&buf, dataSize); err != nil {
		v.setState(StateIdle)
		return err
	}
	for _, c := range seg {
		if _, err := buf.Write(c); err != nil {
			v.setState(StateIdle)
			return err
		}
	}
	id := uuid.New().String()
	fileName := fmt.Sprintf("seg_%s.wav", id)

	fileKey, err := v.fileUsecase.UploadFileWithWriter(ctx, fileName, bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		v.logger.Error("upload failed", log.Error(err))
		v.setState(StateIdle)
		return err
	}
	fileUrl := fmt.Sprintf("%s/%s/%s", v.config.EndPoint, v.config.Oss.BucketName, fileKey)
	v.logger.Info("VAD segment uploaded", log.String("id", id), log.String("url", fileUrl))

	// 调用 ASR（如果 asrUsecase.Asr 支持 ctx，传入 ctx）
	result, err := v.asrUsecase.Asr(ctx, fileUrl)
	if err != nil {
		v.logger.Error("asr error", log.Error(err))
		v.setState(StateIdle)
		return err
	}

	text := ""
	if result != nil {
		text = result.Data.Result.Text
	}

	// 发回上层，不阻塞主 loop
	if v.resultChan != nil {
		select {
		case v.resultChan <- ASRResult{Text: text, SegID: segID, FileURL: fileUrl}:
		default:
			// 如果上层接收慢，避免阻塞
			v.logger.Warn("resultChan full, dropping asr result")
		}
	}

	// 进入 Responding，并等待上层调用 OnResponseDone()
	v.setState(StateResponding)
	return nil
}

func writeWavHeader(w io.Writer, dataSize int64) error {
	var header [44]byte
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+dataSize))
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], SampleRate)
	binary.LittleEndian.PutUint32(header[28:], SampleRate*1*BitDepth/8)
	binary.LittleEndian.PutUint16(header[32:], 1*BitDepth/8)
	binary.LittleEndian.PutUint16(header[34:], BitDepth)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(dataSize))
	_, err := w.Write(header[:])
	return err
}
