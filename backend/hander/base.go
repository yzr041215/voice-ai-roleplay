package hander

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
type BaseHandler struct {
}

func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}
func (h *BaseHandler) NewResponseWithData(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func (h *BaseHandler) NewResponseWithError(c echo.Context, msg string, err error) error {

	return c.JSON(http.StatusOK, Response{
		Success: false,
		Message: fmt.Sprintf("%s: %s", msg, err.Error()),
	})
}
