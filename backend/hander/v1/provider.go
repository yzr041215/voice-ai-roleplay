package V1

import (
	"demo/hander"
	"demo/usecase"

	"github.com/google/wire"
)

type Handers struct {
	Hello *HelloHander
	User  *UserHander
}

var ProviderSet = wire.NewSet(
	hander.NewBaseHandler,
	NewHelloHander,
	NewUserHander,
	usecase.ProviderSet,

	wire.Struct(new(Handers), "*"),
)
