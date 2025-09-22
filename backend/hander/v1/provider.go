package V1

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewHelloHander)
