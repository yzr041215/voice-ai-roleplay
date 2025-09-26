package domain

type Role struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Prompt   string `json:"prompt"`
	ImageUrl string `json:"image_url"`
	//音色
	Voice string `json:"voice"` //eg：qiniu_zh_female_tmjxxy
	//浏览量
	Views int `json:"views"`
	//点赞量
	Likes int `json:"likes"`
}
type RoleWithoutPrompt struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ImageUrl string `json:"image_url"`
	//音色
	Voice string `json:"voice"` //eg：qiniu_zh_female_tmjxxy
	//浏览量
	Likes int `json:"likes"`
}
type RoleList struct {
	Roles []RoleWithoutPrompt `json:"roles"`
}

const VoicePromot = `
你正在参与实时语音对话，用户只能听到纯语音。请遵守：
只输出应说的句子，禁止任何旁白、舞台指示或动作描写（如 微笑、转身、轻声说 等）。
不使用引号、括号、破折号等标点符号包裹文字。
语言简洁口语化，长短适中，方便立即 TTS 播报。
直接开始回答，不要重复用户问题。
必须结合上下文聊天记录，保持话题连贯、语气自然，实现平滑无缝的对话体验。
`
