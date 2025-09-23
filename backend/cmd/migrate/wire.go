//go:build wireinject
// +build wireinject

package main

import (
	"demo/config"
	V1 "demo/hander/v1"
	"demo/pkg/log"
	"demo/pkg/store"

	"github.com/google/wire"
)

type App struct {
	db     *store.MySQL
	config *config.Config
}

func InitializeApp() *App {
	wire.Build(
		wire.Struct(new(App), "*"),
		wire.NewSet(
			store.NewMySQL,
			config.NewConfig,
			V1.ProviderSet,
			log.ProviderSet,
		),
	)
	return &App{}
}
