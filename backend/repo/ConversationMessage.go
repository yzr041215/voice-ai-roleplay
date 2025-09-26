package repo

import (
	"context"
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"demo/pkg/store"
)

type ConversationMessageRepo struct {
	log    *log.Logger
	config *config.Config
	db     *store.MySQL
}

func NewConversationRepo(log *log.Logger, config *config.Config, db *store.MySQL) *ConversationMessageRepo {
	return &ConversationMessageRepo{
		log:    log.WithModule("ConversationMessageRepo"),
		config: config,
		db:     db,
	}
}

func (c *ConversationMessageRepo) CreateMessage(ctx context.Context, m domain.ConversationMessage) error {
	return c.db.DB.WithContext(ctx).Create(&m).Error
}

func (c ConversationMessageRepo) GetMessagesByUserIDAndRoleID(ctx context.Context, userID string, roleID int) ([]domain.ConversationMessage, error) {
	var messages []domain.ConversationMessage
	err := c.db.DB.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Order("id ASC").Find(&messages).Error
	if err != nil {
		c.log.Error("err ", log.Error(err))
		return nil, err
	}
	return messages, nil
}
