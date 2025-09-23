//go:build wireinject
// +build wireinject

package main

import (
	"demo/config"
	V1 "demo/hander/v1"
	"demo/pkg/log"
	"demo/serve"

	"github.com/google/wire"
)

type App struct {
	Service *serve.HttpServer
	config  *config.Config
	v1      *V1.Handers
}

func InitializeApp() *App {
	wire.Build(
		wire.Struct(new(App), "*"),
		wire.NewSet(
			serve.NewHttpServer,
			config.NewConfig,
			V1.ProviderSet,
			log.ProviderSet,
		),
	)
	return &App{}
}
