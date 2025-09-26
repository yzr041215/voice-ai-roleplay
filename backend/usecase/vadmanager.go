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
	"golang.org/x/sync/errgroup"
)

const (
	SampleRate    = 16000                             // 16kHz
	FrameDuration = 20                                // ms
	FrameSize     = SampleRate / 1000 * FrameDuration // 320 点
	BitDepth      = 16
	BytesPerFrame = FrameSize * BitDepth / 8 // 640 字节
)

type VadManager struct {
	logger      *log.Logger
	asrUsecase  *AsrUsecase
	fileUsecase *FileUsecase
	config      *config.Config

	vad       webrtcvad.VadInst
	mu        sync.Mutex
	currSeg   [][]byte
	segID     int
	vadActive bool

	eg *errgroup.Group
}

// NewVadManager 创建VadManager实例
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

	eg, _ := errgroup.WithContext(context.Background())

	return &VadManager{
		logger:      logger.WithModule("VadManager"),
		asrUsecase:  asrUsecase,
		fileUsecase: fileUsecase,
		config:      config,
		vad:         vad,
		eg:          eg,
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

// ProcessAudioStream 处理音频流（带VAD）
func (v *VadManager) ProcessAudioStream(ctx context.Context, audioChunks <-chan []byte) error {
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
		} else {
			if v.vadActive && len(v.currSeg) > 0 {
				segID := v.segID
				chunks := make([][]byte, len(v.currSeg))
				copy(chunks, v.currSeg)

				// 提交到 errgroup
				v.eg.Go(func() error {
					return v.handleSegment(ctx, segID, chunks)
				})

				v.segID++
				v.currSeg = nil
			}
			v.vadActive = false
		}
		v.mu.Unlock()
	}

	// flush 最后一段
	v.mu.Lock()
	if v.vadActive && len(v.currSeg) > 0 {
		segID := v.segID
		chunks := make([][]byte, len(v.currSeg))
		copy(chunks, v.currSeg)
		v.eg.Go(func() error {
			return v.handleSegment(ctx, segID, chunks)
		})
		v.segID++
		v.currSeg = nil
		v.vadActive = false
	}
	v.mu.Unlock()

	// 等待所有 segment 完成
	return v.eg.Wait()
}

// handleSegment 保存一句音频并调用ASR
func (v *VadManager) handleSegment(ctx context.Context, id int, seg [][]byte) error {
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

	// 上传
	fileName := fmt.Sprintf("seg_%d.wav", id)
	fileKey, err := v.fileUsecase.UploadFileWithWriter(ctx, fileName, bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return err
	}
	fileUrl := fmt.Sprintf("%s/%s/%s", v.config.EndPoint, v.config.Oss.BucketName, fileKey)

	// 调用 ASR
	result, err := v.asrUsecase.Asr(ctx, fileUrl)
	if err != nil {
		return err
	}

	v.logger.Info("VAD segment result",
		log.Int("id", id),
		log.String("text", result.Text),
	)
	return nil
}

// writeWavHeader 写 WAV PCM16 单声道 16kHz 头
func writeWavHeader(w io.Writer, dataSize int64) error {
	var header [44]byte
	// ChunkID "RIFF"
	copy(header[0:], "RIFF")
	// ChunkSize = 36 + Subchunk2Size
	binary.LittleEndian.PutUint32(header[4:], uint32(36+dataSize))
	// Format "WAVE"
	copy(header[8:], "WAVE")
	// Subchunk1ID "fmt "
	copy(header[12:], "fmt ")
	// Subchunk1Size 16 for PCM
	binary.LittleEndian.PutUint32(header[16:], 16)
	// AudioFormat 1 = PCM
	binary.LittleEndian.PutUint16(header[20:], 1)
	// NumChannels 1
	binary.LittleEndian.PutUint16(header[22:], 1)
	// SampleRate
	binary.LittleEndian.PutUint32(header[24:], SampleRate)
	// ByteRate = SampleRate * NumChannels * BitsPerSample/8
	binary.LittleEndian.PutUint32(header[28:], SampleRate*1*BitDepth/8)
	// BlockAlign = NumChannels * BitsPerSample/8
	binary.LittleEndian.PutUint16(header[32:], 1*BitDepth/8)
	// BitsPerSample
	binary.LittleEndian.PutUint16(header[34:], BitDepth)
	// Subchunk2ID "data"
	copy(header[36:], "data")
	// Subchunk2Size
	binary.LittleEndian.PutUint32(header[40:], uint32(dataSize))

	_, err := w.Write(header[:])
	return err
}
