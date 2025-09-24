package usecase

import (
	"bytes"
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
)

// TtsUsecase 提供文本转语音服务
type TtsUsecase struct {
	l      *log.Logger
	config *config.Config
}

// NewTtsUsecase 创建TtsUsecase实例
func NewTtsUsecase(l *log.Logger, c *config.Config) *TtsUsecase {
	return &TtsUsecase{
		l:      l.WithModule("TtsUsecase"),
		config: c,
	}
}

// Tts 调用七牛云TTS API进行文本转语音
// 参数:
//   - ctx: 上下文
//   - text: 要转换的文本
//   - voiceType: 语音类型(如"qiniu_zh_female_wwxkjx")
//   - encoding: 音频编码(如"mp3")
//   - speedRatio: 语速(1.0为正常速度)
// 返回:
//   - *domain.TtsResponse: 转换结果
//   - error: 错误信息
func (t *TtsUsecase) Tts(
	ctx context.Context,
	text string,
	voiceType string,
	encoding string,
	speedRatio float64,
) (*domain.TtsResponse, error) {
	// 构造请求体
	requestBody := map[string]interface{}{
		"audio": map[string]interface{}{
			"voice_type":  voiceType,
			"encoding":    encoding,
			"speed_ratio": speedRatio,
		},
		"request": map[string]string{
			"text": text,
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		t.config.Tts.BaseUrl,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.config.Tts.ApiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts api returned non-200 status: %s", resp.Status)
	}

	// 解析响应
	var result domain.TtsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
