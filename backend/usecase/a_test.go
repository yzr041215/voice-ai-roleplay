package usecase

import (
	"context"
	"demo/config"
	"demo/pkg/log"
	"demo/usecase/utils"
	"fmt"
	"testing"
)

func Test_A(t *testing.T) {
	c := config.NewConfig()
	c.Asr.ApiKey = "sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf"
	c.Asr.BaseUrl = "https://openai.qiniu.com/v1"

	lo := log.NewLogger(c)
	l := NewAsrUsecase(lo, c)
	//http://204.141.218.207:9000/mybucket/seg_0.wav
	ch, err := l.Asr(context.Background(), "http://204.141.218.207:9000/mybucket/seg_0.wav")
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("ch: %v\n", ch.Data.Result.Text)
}
func TestC(t *testing.T) {
	c := config.NewConfig()
	c.Asr.ApiKey = "sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf"
	c.Asr.BaseUrl = "https://openai.qiniu.com/v1"
	lo := log.NewLogger(c)
	s := utils.NewAsrUsecase(lo, c)
	ctx := context.Background()

	pcmChan := make(chan []byte)

	// 启动 ASR
	go func() {
		err := s.AsrStream(ctx, pcmChan, func(text string,is bool) {
			fmt.Println("识别结果:", text)
		})
		if err != nil {
			fmt.Println("ASR 出错:", err)
		}
	}()

	// 模拟推 PCM 块（实际中从麦克风或 VAD 拼接来的数据）
	//	pcmChan <- somePCMBytes
	//	pcmChan <- morePCMBytes
	close(pcmChan)
}
func Test_B(t *testing.T) {
	c := config.NewConfig()
	c.Tts.ApiKey = "sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf"
	c.Asr.BaseUrl = "openai.qiniu.com"

	lo := log.NewLogger(c)
	l := utils.NewTtsStream(lo, c)

	chunks := make(chan string, 1)

	go func() {
		chunks <- "第一段文本"
		chunks <- "第二段文本"
		close(chunks)
	}()

	pcmStream, errCh := l.TtsStream(context.Background(), chunks, "qiniu_zh_female_tmjxxy", 1.0)

	for {
		select {
		case pcm, ok := <-pcmStream:
			if !ok {
				fmt.Println("流结束")
				return
			}
			fmt.Printf("seq=%d pcm_samples=%d\n", pcm.Seq, len(pcm.Samples))
			// 在这里写入 wav 文件，或者直接送给音频播放设备
		case err := <-errCh:
			if err != nil {
				fmt.Println("错误:", err)
				return
			}
		}
	}
}
