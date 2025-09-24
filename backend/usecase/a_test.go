package usecase

import (
	"context"
	"demo/config"
	"demo/pkg/log"
	"testing"
)

/*
BASE_URL=https://openai.qiniu.com/v1
ASR_MODEL_ID=ch_asr_nli_wpe_conformer_large_multi_cn
API_KEY=sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf
*/
func Test_A(t *testing.T) {
	c := config.NewConfig()
	c.Asr.ApiKey = "sk-9d77b19cf27454ab47ea9b0d9d2db30c7139c5712c0e553bc31eb6fa4dc90bcf"
	c.Asr.BaseUrl = "https://openai.qiniu.com/v1"
	lo := log.NewLogger(c)
	l := NewLlmUsecase(lo, c)
	ch, err := l.Chat(context.Background(), "你好啊", "")
	if err != nil {
		t.Error(err)
	}
	for msg := range ch {
		t.Log(msg)
	}
}
