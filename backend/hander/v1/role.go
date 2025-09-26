package V1

import (
	"demo/domain"
	"demo/hander"
	"demo/pkg/log"
	"demo/serve"
	"demo/usecase"

	"github.com/labstack/echo/v4"
)

type RoleHander struct {
	*hander.BaseHandler

	log         *log.Logger
	roleUsecase usecase.RoleUsecase
}

func NewRoleHander(s *serve.HttpServer, log *log.Logger, base *hander.BaseHandler, roleUsecase usecase.RoleUsecase) *RoleHander {
	r := &RoleHander{
		BaseHandler: base,
		log:         log.WithModule("RoleHander"),
		roleUsecase: roleUsecase,
	}
	s.Echo.GET("/v1/roles", r.ListRoles)
	return r
}

// ListRoles lists all roles
// @Summary lists all roles
// @Description lists all roles
// @Tags Role
// @Produce json
// @Success 200 {object} domain.RoleList
// @Router /v1/roles [get]
func (h *RoleHander) ListRoles(c echo.Context) error {
	roles, err := h.roleUsecase.ListRoles(c.Request().Context())
	if err != nil {
		h.NewResponseWithError(c, "Failed to list roles", err)
		return err
	}
	h.NewResponseWithData(c, domain.RoleList{Roles: roles})
	return nil
}


