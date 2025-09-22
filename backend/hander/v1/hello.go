package V1

import (
	"demo/serve"

	"github.com/labstack/echo/v4"
)

type HelloHander struct {
}

func NewHelloHander(s *serve.HttpServer) *HelloHander {
	g := s.Echo.Group("/v1")
	g.GET("/hello", Hello)
	return &HelloHander{}
}

func Hello(c echo.Context) error {
	return c.String(200, "Hello, World!")
}
