package repo

import (
	"demo/pkg/store"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	store.ProviderSet,

	NewUserRepo,
)
