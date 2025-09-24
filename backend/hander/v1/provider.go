package V1

import (
	"demo/hander"
	"demo/usecase"

	"github.com/google/wire"
)

type Handers struct {
	Hello *HelloHander
	User  *UserHander
	Role  *RoleHander
}

var ProviderSet = wire.NewSet(
	hander.NewBaseHandler,
	NewHelloHander,
	NewUserHander,
	NewRoleHander,
	usecase.ProviderSet,

	wire.Struct(new(Handers), "*"),
)
