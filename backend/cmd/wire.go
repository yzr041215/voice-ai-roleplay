//go:build wireinject
// +build wireinject

package main

import (
	"demo/config"
	V1 "demo/hander/v1"
	"demo/serve"

	"github.com/google/wire"
)

type App struct {
	Service *serve.HttpServer
	config  *config.Config
	v1      *V1.HelloHander
}

func InitializeApp() *App {
	wire.Build(
		wire.Struct(new(App), "*"),
		wire.NewSet(
			serve.NewHttpServer,
			config.NewConfig,
			V1.ProviderSet,
		),
	)
	return &App{}
}
