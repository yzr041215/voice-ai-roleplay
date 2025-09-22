package serve

import (
	"github.com/labstack/echo/v4"
)

type HttpServer struct {
	Echo *echo.Echo
}

func NewHttpServer() *HttpServer {
	e := echo.New()
	return &HttpServer{
		Echo: e,
	}
}
