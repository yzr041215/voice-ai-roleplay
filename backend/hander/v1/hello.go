package V1

import (
	"demo/serve"
	"demo/usecase"
	"fmt"

	"github.com/labstack/echo/v4"
)

type HelloHander struct {
	l *usecase.LlmUsecase
}

func NewHelloHander(s *serve.HttpServer, l *usecase.LlmUsecase) *HelloHander {
	h := &HelloHander{
		l: l,
	}
	//加载html文件
	s.Echo.Static("/static", "static")
	g := s.Echo.Group("/v1")
	g.GET("/hello", Hello)
	g.POST("/chat", h.Chat)
	g.GET("/index", h.index)
	return h
}

func (h *HelloHander) index(c echo.Context) error {

	return c.HTML(200, string(index))
}

func Hello(c echo.Context) error {
	userid := c.Get("user_id").(string)
	return c.String(200, "Hello, World!"+userid)
}

type req struct {
	Roleid   int    `json:"roleid"`
	Userid   string `json:"userid"`
	Question string `json:"question"`
}

// chat
// @Summary chat with ai
// @Description chat with ai
// @Tags chat
// @Accept  json
// @Produce  json
func (h *HelloHander) Chat(c echo.Context) error {
	var r req
	if err := c.Bind(&r); err != nil {
		return c.JSON(400, err)
	}
	messages, err := h.l.FormatMessage(c.Request().Context(), r.Userid, r.Roleid, r.Question)
	if err != nil {
		return c.JSON(500, err)
	}
	ch, err := h.l.Chat(c.Request().Context(), messages)
	if err != nil {
		fmt.Println("chat err", err)
		return c.JSON(500, err)
	}
	var res string
	for c := range ch {
		res += c
	}
	return c.JSON(200, res)
}
