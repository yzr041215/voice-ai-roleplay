package domain

import (
	"time"

	"github.com/cloudwego/eino/schema"
)

// 会话结构体
type Conversation struct {
	ID        string    `json:"id"`
	Role      string    //AI扮演的角色
	UserID    string    `json:"user_id"`
	Subject   string    `json:"subject"` // subject for conversation, now is first question
	CreatedAt time.Time `json:"created_at"`
}

// 会话消息结构体
type ConversationMessage struct {
	ID             string `json:"id" gorm:"primaryKey"`
	ConversationID string `json:"conversation_id" gorm:"index"`

	Role    schema.RoleType `json:"role"` //用户  AI
	Content string          `json:"content"`
}
