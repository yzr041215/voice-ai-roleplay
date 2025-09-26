package domain

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// 会话消息结构体
type ConversationMessage struct {
	ID     int    `json:"id" gorm:"primaryKey"`
	RoleID int    `json:"role_id"` //AI扮演的角色
	UserID string `json:"user_id"`

	Role    schema.RoleType `json:"role"` //用户  AI
	Content string          `json:"content"`
	Time    time.Time       `json:"time"`
}
