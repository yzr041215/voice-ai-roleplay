package usecase

import (
	"context"
	"demo/config"
	"demo/pkg/log"
	"io"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// LlmUsecase 提供大语言模型服务
type LlmUsecase struct {
	l      *log.Logger
	config *config.Config
}

// NewLlmUsecase 创建LlmUsecase实例
func NewLlmUsecase(l *log.Logger, c *config.Config) *LlmUsecase {
	return &LlmUsecase{
		l:      l.WithModule("LlmUsecase"),
		config: c,
	}
}

// Chat 调用七牛云LLM API进行对话

func (l *LlmUsecase) Chat(ctx context.Context, message string, promot string) (<-chan string, error) {
	chatConfig := &openai.ChatModelConfig{
		APIKey:  l.config.Asr.ApiKey,
		BaseURL: l.config.Asr.BaseUrl,
		Model:   "deepseek-v3",
	}
	chatModel, err := openai.NewChatModel(ctx, chatConfig)
	if err != nil {
		return nil, err
	}
	messages := []*schema.Message{
		schema.SystemMessage(promot),
		schema.UserMessage(message),
	}

	resp, err := chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	ch := make(chan string)
	go func() {
		for {
			msg, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			ch <- msg.Content
		}
	}()
	return ch, err
}
