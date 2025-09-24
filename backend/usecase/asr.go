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

// AsrUsecase 提供语音识别服务
type AsrUsecase struct {
	l      *log.Logger
	config *config.Config
}

// NewAsrUsecase 创建AsrUsecase实例
func NewAsrUsecase(l *log.Logger, c *config.Config) *AsrUsecase {
	return &AsrUsecase{
		l:      l.WithModule("AsrUsecase"),
		config: c,
	}
}

// Asr 调用七牛云ASR API进行语音识别
// 参数:
//   - ctx: 上下文
//   - audioUrl: 音频文件URL
//
// 返回:
//   - *domain.AsrResponse: 识别结果
//   - error: 错误信息
func (a *AsrUsecase) Asr(ctx context.Context, audioUrl string) (*domain.AsrResponse, error) {
	// 构造请求体
	requestBody := map[string]interface{}{
		"model": "asr",
		"audio": map[string]string{
			"format": "mp3", // 可根据实际需求修改格式
			"url":    audioUrl,
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
		a.config.Asr.BaseUrl,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.Asr.ApiKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("asr api returned non-200 status: %s", resp.Status)
	}

	// 解析响应
	var result domain.AsrResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
