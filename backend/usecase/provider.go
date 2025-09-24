package usecase

import (
	"demo/repo"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewUserUsecase, NewRoleUsecase, repo.ProviderSet)
