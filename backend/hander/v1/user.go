package V1

import (
	_ "demo/docs"
	"demo/domain"
	"demo/hander"
	"demo/pkg/log"
	"demo/serve"
	"demo/usecase"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type UserHander struct {
	*hander.BaseHandler

	logger   *log.Logger
	usercase *usecase.UserUsecase
}

func NewUserHander(s *serve.HttpServer, base *hander.BaseHandler, logger *log.Logger, userusecase *usecase.UserUsecase) *UserHander {
	g := s.Echo.Group("/v1")
	g.GET("/swagger/*", echoSwagger.WrapHandler)

	g.GET("/swagger/swagger.json", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})
	u := &UserHander{
		BaseHandler: base,
		logger:      logger,
		usercase:    userusecase,
	}
	g.POST("/register", u.Register)
	g.POST("/login", u.Login)
	return u
}

// Register godoc
// @Summary Register a new user
// @Description Register a new user
// @Tags User
// @Accept  json
// @Produce json
// @Param user body domain.CreateUserReq true "User data"
// @Success 200 {object} string "User registered successfully"
// @Failure 400 {object} string "Bad request"
// @Router /v1/register [post]
func (u *UserHander) Register(c echo.Context) error {
	var user domain.CreateUserReq
	if err := c.Bind(&user); err != nil {
		return err
	}
	err := u.usercase.Register(c.Request().Context(), &user)
	if err != nil {
		u.NewResponseWithError(c, "Failed to register user", err)
		return err
	}
	return u.NewResponseWithData(c, "User registered successfully")
}

// Login godoc
// @Summary Login a user
// @Description Login a user
// @Tags User
// @Accept  json
// @Produce json
// @Param user body domain.LoginReq true "User data"
// @Success 200 {object} domain.LoginResp "User logged in successfully"
// @Failure 400 {object} string "Bad request"
// @Router /v1/login [post]
func (u *UserHander) Login(c echo.Context) error {
	var req domain.LoginReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	resp, err := u.usercase.Login(c.Request().Context(), &req)
	if err != nil {
		u.NewResponseWithError(c, "Failed to login user", err)
		return err
	}
	return u.NewResponseWithData(c, resp)
}
