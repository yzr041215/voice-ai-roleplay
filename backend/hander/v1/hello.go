package V1

import (
	"demo/hander/midwire"
	"demo/serve"

	"github.com/labstack/echo/v4"
)

type HelloHander struct {
}

func NewHelloHander(s *serve.HttpServer) *HelloHander {
	g := s.Echo.Group("/v1", midwire.Mid)
	g.GET("/hello", Hello)
	return &HelloHander{}
}

func Hello(c echo.Context) error {
	userid := c.Get("user_id").(string)
	return c.String(200, "Hello, World!"+userid)
}
