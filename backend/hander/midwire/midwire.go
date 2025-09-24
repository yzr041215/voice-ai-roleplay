package midwire

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	echoMiddleware "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

// 从token中获取user_id
//
//	func Authorization(c echo.Context) error {
//		return Mid(c)
//	}
func Mid(next echo.HandlerFunc) echo.HandlerFunc {
	jwtMiddleware := echoMiddleware.WithConfig(echoMiddleware.Config{
		SigningKey:  []byte("secret"),
		TokenLookup: "header:Authorization", // 也可以 query:token 等
		ContextKey:  "user",                 // 默认就是 user
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusUnauthorized, map[string]string{"msg": "invalid token" + err.Error()})
		},
	})
	return jwtMiddleware(func(c echo.Context) error {
		token, ok := c.Get("user").(*jwt.Token)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"msg": "invalid token type"})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"msg": "invalid claims type"})
		}

		// 安全取值
		userId, ok := claims["user_id"].(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]string{"msg": "user_id not found or type error"})
		}

		c.Set("user_id", userId)
		return next(c)
	})
}
