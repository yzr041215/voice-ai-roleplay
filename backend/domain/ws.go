package domain

type MsgType int

const (
	//ws为websocket.BinaryMessage实际的mp3格式二进制流
	
	//ws为websocket.TextMessage实际的文本消息
	MsgTypeIntrupt  MsgType = iota //客户端打断
	MsgTypeTanslate                //服务端翻译的中文

)

// ws通信结构体
type Msg struct {
	Type MsgType
}
