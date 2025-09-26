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

const (
	SampleRate    = 16000                             // 16kHz
	FrameDuration = 20                                // ms
	FrameSize     = SampleRate / 1000 * FrameDuration // 320 点
	BitDepth      = 16
	BytesPerFrame = FrameSize * BitDepth / 8 // 640 字节
	SilenceFrames = 50                       // 1s = 1000ms / 20ms = 50帧
)

type VadManager struct {
	logger      *log.Logger
	asrUsecase  *AsrUsecase
	fileUsecase *FileUsecase
	config      *config.Config

	vad          webrtcvad.VadInst
	mu           sync.Mutex
	currSeg      [][]byte
	segID        int
	silenceCount int
	vadActive    bool
}

// NewVadManager 创建 VadManager 实例
func NewVadManager(
	logger *log.Logger,
	asrUsecase *AsrUsecase,
	fileUsecase *FileUsecase,
	config *config.Config,
) *VadManager {
	vad := webrtcvad.Create()
	if vad == nil {
		panic("failed to create vad instance")
	}
	if err := webrtcvad.Init(vad); err != nil {
		panic(err)
	}
	if err := webrtcvad.SetMode(vad, 3); err != nil { // 0-3 越大越激进
		panic(err)
	}

	return &VadManager{
		logger:      logger.WithModule("VadManager"),
		asrUsecase:  asrUsecase,
		fileUsecase: fileUsecase,
		config:      config,
		vad:         vad,
	}
}

// Close 释放 VAD 实例
func (v *VadManager) Close() {
	if v.vad != nil {
		webrtcvad.Free(v.vad)
		v.vad = nil
	}
}

// IsVad 当前是否处于语音段
func (v *VadManager) IsVad() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.vadActive
}

// ProcessAudioStream 处理音频流（带 VAD）
func (v *VadManager) ProcessAudioStream(ctx context.Context, audioChunks <-chan []byte) error {
	eg, _ := errgroup.WithContext(ctx)

	for chunk := range audioChunks {
		if len(chunk) != BytesPerFrame {
			v.logger.Warn("invalid frame size",
				log.Int("got", len(chunk)),
				log.Int("want", BytesPerFrame))
			continue
		}

		active, err := webrtcvad.Process(v.vad, SampleRate, chunk, FrameSize)
		if err != nil {
			v.logger.Error("vad process error", log.Error(err))
			continue
		}

		v.mu.Lock()
		if active {
			v.vadActive = true
			v.currSeg = append(v.currSeg, chunk)
			v.silenceCount = 0
		} else if v.vadActive {
			v.silenceCount++
			v.currSeg = append(v.currSeg, chunk)

			// 达到连续静音阈值，断句
			if v.silenceCount >= SilenceFrames {
				chunks := make([][]byte, len(v.currSeg))
				copy(chunks, v.currSeg)

				eg.Go(func() error {
					return v.handleSegment(ctx, chunks)
				})

				v.segID++
				v.currSeg = nil
				v.vadActive = false
				v.silenceCount = 0
			}
		}
		v.mu.Unlock()
	}
	return eg.Wait()
}

// handleSegment 保存一句音频并调用 ASR
func (v *VadManager) handleSegment(ctx context.Context, seg [][]byte) error {
	var buf bytes.Buffer

	// 写 WAV header
	dataSize := int64(len(seg) * BytesPerFrame)
	if err := writeWavHeader(&buf, dataSize); err != nil {
		return err
	}

	// 写 PCM 数据
	for _, c := range seg {
		if _, err := buf.Write(c); err != nil {
			return err
		}
	}
	id := uuid.New().String()
	// 上传
	fileName := fmt.Sprintf("seg_%s.wav", id)
	fileKey, err := v.fileUsecase.UploadFileWithWriter(ctx, fileName, bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return err
	}
	fileUrl := fmt.Sprintf("%s/%s/%s", v.config.EndPoint, v.config.Oss.BucketName, fileKey)
	v.logger.Info("VAD segment uploaded",
		log.String("id", id),
		log.String("url", fileUrl),
	)
	// 调用 ASR
	result, err := v.asrUsecase.Asr(ctx, fileUrl)
	if err != nil {
		v.logger.Error("asr error", log.Error(err))
		return err
	}

	v.logger.Info("VAD segment result",
		log.String("id", id),
		log.String("text", result.Data.Result.Text),
	)
	return nil
}

// writeWavHeader 写 WAV PCM16 单声道 16kHz 头
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
