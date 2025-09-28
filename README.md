# voice-ai-roleplay
议题二 开发一个利用 AI 来做角色扮演的网站，用户可以搜索自己感兴趣的角色例如哈利波特、苏格拉底等并可与其进行语音聊天。
## 全部使用七牛云的asr，tts,llm（llm使用的字节的eino库sdk，另外的都是http/wss拼接请求）
## 项目功能大概实现，但是asr流式的文档太乱了，写不下去了，vad断句的bug没时间解决了
使用ddd领域驱动设计

wire依赖注入

通过websocket连接，wsusecase.go处理连接，pcm数据帧+调用asr->llm （不同role使用不同的prompt）->tts

通过聊天记录表根据userid和roleid使用FormatMessage函数拼接Message实现聊天上下文

使用role表+prompt+voicetable实现不同音色和角色性格回答

wsusecase内有两个handerws函数
* handerws：使用websocket的asr服务，但是内置的vad断句不准确，final信号几乎没有（文档不详细，可能没告知详细配置细节）后续使用手动vad，但是无法清除asr内的数据缓存，导致上一句依旧吐词，没法写了，
* handerws2：使用手动vad实现断句并且合并+minIo存储wav文件，整体使用文件公网地址调用asr服务，处理太慢了
## 项目启动
### 一键运行
`cd backend && make dev`
## 吐槽
(这七牛云asr的文档也太难用了。。。。。。。。。。。。。。。。，流式api还不给文档，给响应式的文档，流式的js demo，我想刷新asr的vad断句也没办法，不给字段文档，我直接写崩了。。。。，没时间换阿里云的模型了，还得新换文档和sdk).
七天感觉前面都在踩坑和调试，确实浪费了



### 目录结构
├── cmd
│   ├── cert.pem
│   ├── key.pem
│   ├── main.go
│   ├── migrate
│   │   ├── migrate.go
│   │   ├── wire_gen.go
│   │   └── wire.go
│   ├── wire_gen.go
│   └── wire.go
├── config
│   ├── config.go
│   └── provider.go
├── data
├── docker-compose.yml
├── Dockerfile
├── docs
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── domain
│   ├── asr.go
│   ├── hello.go
│   ├── messgae.go
│   ├── role.go
│   ├── tts.go
│   ├── user.go
│   └── ws.go
├── go.mod
├── go.sum
├── hander
│   ├── base.go
│   ├── midwire
│   │   └── midwire.go
│   └── v1
│       ├── hello.go
│       ├── index.go
│       ├── provider.go
│       ├── role.go
│       └── user.go
├── makefile
├── pkg
│   ├── log
│   │   ├── log.go
│   │   └── provider.go
│   └── store
│       ├── mysql.go
│       ├── oss.go
│       └── provider.go
├── repo
│   ├── ConversationMessage.go
│   ├── provider.go
│   ├── role.go
│   └── user.go
├── serve
│   └── http.go
└── usecase
    ├── asr.go
    ├── a_test.go
    ├── fileusecase.go
    ├── llm.go
    ├── provider.go
    ├── role.go
    ├── user.go
    ├── utils
    │   ├── asr_stream.go
    │   └── tts.go
    ├── vadmanager.go
    └── wsusecase.go
