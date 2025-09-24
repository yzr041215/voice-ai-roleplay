package store

import (
	"demo/config"
	"demo/domain"
	"demo/pkg/log"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQL struct {
	*gorm.DB
}

func NewMySQL(config *config.Config) *MySQL {
	db, err := gorm.Open(mysql.Open(config.MySQL.Dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// 自动迁移表结构
	db.AutoMigrate(domain.User{})
	db.AutoMigrate(domain.Role{})

	// 分割SQL语句并执行
	sqlStatements := strings.Split(initSQL, ";")
	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			log.NewLogger(config).WithModule("MySQL").Error("Failed to execute SQL: ", "ERROR: ", err.Error(), "\nStatement: ", stmt)
		}
	}

	return &MySQL{db}
}

const initSQL = `
-- 创建role表
CREATE TABLE IF NOT EXISTS roles (
    id     INT UNSIGNED NOT NULL AUTO_INCREMENT,
    name   VARCHAR(127) NOT NULL,
    prompt VARCHAR(2048) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_role_name (name)
) ENGINE = InnoDB;
-- 插入初始数据
INSERT INTO roles (name, prompt) VALUES
('苏格拉底', '你是一位古希腊哲学家苏格拉底，以问答法闻名。你的对话风格应该是引导性的，通过提问帮助对方思考。避免直接给出答案，而是通过一系列问题引导对方自己得出结论。你的语言应该简洁而深刻，充满智慧。'),
('孔子', '你是一位中国古代思想家孔子，儒家学派的创始人。你的对话应该体现"仁"的思想，强调道德修养和人际关系。使用简洁而富有哲理的语言，可以适当引用《论语》中的经典语句。'),
('爱因斯坦', '你是一位理论物理学家爱因斯坦，以相对论闻名。你的对话应该充满科学精神，但也要用通俗易懂的方式解释复杂概念。可以表现出幽默感和对人类命运的关怀。'),
('达芬奇', '你是一位文艺复兴时期的博学者达芬奇，既是艺术家也是科学家。你的对话应该展现跨学科的思维方式，将艺术与科学结合起来。可以谈论绘画、解剖学、工程学等不同领域。'),
('莎士比亚', '你是一位英国剧作家莎士比亚，以戏剧和诗歌闻名。你的对话应该富有文学性，可以适当引用戏剧中的经典台词。表现出对人性的深刻理解和对语言的精湛掌握。');
`
