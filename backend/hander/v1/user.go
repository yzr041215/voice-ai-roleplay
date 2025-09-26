package V1

import (
	_ "demo/docs"
	"demo/domain"
	"demo/hander"
	"demo/pkg/log"
	"demo/serve"
	"demo/usecase"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type UserHander struct {
	*hander.BaseHandler
	fileUsecase *usecase.FileUsecase
	logger      *log.Logger
	usercase    *usecase.UserUsecase
	wsusecase   *usecase.WsUseCase
}

func NewUserHander(s *serve.HttpServer, base *hander.BaseHandler, logger *log.Logger, userusecase *usecase.UserUsecase, fileusecase *usecase.FileUsecase, ws *usecase.WsUseCase) *UserHander {
	g := s.Echo.Group("/v1")
	g.GET("/swagger/*", echoSwagger.WrapHandler)

	g.GET("/swagger/swagger.json", func(c echo.Context) error {
		return c.File("docs/swagger.json")
	})
	u := &UserHander{
		BaseHandler: base,
		logger:      logger,
		usercase:    userusecase,
		fileUsecase: fileusecase,
		wsusecase:   ws,
	}
	g.POST("/register", u.Register)
	g.POST("/login", u.Login)
	g.POST("/upload", u.Upload)
	g.GET("/ws", u.UpgradeToWS)
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

// Upload godoc
// @Summary Upload a file
// @Description Upload a file
// @Tags File
// @Accept  multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {object} string "File uploaded successfully"
// @Failure 400 {object} string "Bad request"
// @Router /v1/upload [post]
func (u *UserHander) Upload(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	Key, err := u.fileUsecase.UploadFile(c.Request().Context(), file)
	if err != nil {
		u.NewResponseWithError(c, "Failed to upload file", err)
		return err
	}
	return u.NewResponseWithData(c, Key)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // 开发阶段放行全部来源
}

// UpgradeToWS godoc
// @Summary 升级为 WebSocket 实时对话
// @Description 握手成功后，客户端与服务端全双工通信
// @Tags User
// @Param roleid path int true "Role id"
// @Success 101 {string} string "Switching Protocols"
// @Router /v1/ws [get]
func (u *UserHander) UpgradeToWS(c echo.Context) error {
	// Echo 内置助手，一行完成 HTTP/1.1 → 101 升级
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	//	userid, ok := c.Get("userid").(string)
	//roleid, err := strconv.Atoi(c.Param("roleid"))
	//if !ok || err != nil {
	//	return errors.New("Invalid user id or role id")
	//}
	// 这里替换成你的「上下文聊天」逻辑即可
	u.wsusecase.HanderWs(ws, "1", 1)
	return nil
}
