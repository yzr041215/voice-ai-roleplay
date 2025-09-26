package usecase

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/repo"
	"io"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// LlmUsecase 提供大语言模型服务
type LlmUsecase struct {
	l                *log.Logger
	config           *config.Config
	conversationRepo *repo.ConversationMessageRepo
	rolerepo         *repo.RoleRepo
}

// NewLlmUsecase 创建LlmUsecase实例
func NewLlmUsecase(l *log.Logger, c *config.Config, conversationRepo *repo.ConversationMessageRepo, rolerepo *repo.RoleRepo) *LlmUsecase {
	return &LlmUsecase{
		l:                l.WithModule("LlmUsecase"),
		config:           c,
		conversationRepo: conversationRepo,
		rolerepo:         rolerepo,
	}
}

// Chat 调用七牛云LLM API进行对话

func (l *LlmUsecase) Chat(ctx context.Context, messages []*schema.Message) (<-chan string, error) {
	chatConfig := &openai.ChatModelConfig{
		APIKey:  l.config.Asr.ApiKey,
		BaseURL: l.config.Asr.BaseUrl,
		Model:   "deepseek-v3",
	}
	chatModel, err := openai.NewChatModel(ctx, chatConfig)
	if err != nil {
		return nil, err
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
			l.l.Info("receive message", log.String("message", msg.Content))
			ch <- msg.Content
		}
		close(ch)
	}()
	return ch, err
}

func (l *LlmUsecase) FormatMessage(ctx context.Context, userid string, roleid int, question string) ([]*schema.Message, error) {
	messages, err := l.conversationRepo.GetMessagesByUserIDAndRoleID(ctx, userid, roleid)
	if err != nil {
		l.l.Error("error get messgaes", log.Error(err))
		return nil, err
	}
	role, err := l.rolerepo.GetroleById(ctx, roleid)

	var formattedMessages []*schema.Message
	formattedMessages = append(formattedMessages, &schema.Message{
		Role:    schema.System,
		Content: role.Prompt,
	},
		&schema.Message{
			Role:    schema.System,
			Content: domain.VoicePromot,
		},
	)
	for _, m := range messages {
		if m.Role == schema.Assistant {
			formattedMessages = append(formattedMessages, &schema.Message{
				Role:    schema.Assistant,
				Content: m.Content,
			})
			continue
		}
		if m.Role == schema.User {
			formattedMessages = append(formattedMessages, &schema.Message{
				Role:    schema.User,
				Content: m.Content,
			})
			continue
		}
	}
	formattedMessages = append(formattedMessages, &schema.Message{
		Role:    schema.User,
		Content: question,
	})
	return formattedMessages, nil
}
