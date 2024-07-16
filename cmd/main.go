package main

import (
	"go.uber.org/zap"
	"pickup/internal/initial"
)

func main() {
	initial.InitLogger()
	zap.S().Infof("logger initialized")

	initial.InitConfigByViper()
	zap.S().Infof("config initialized")

	initial.InitRouters()
	zap.S().Infof("router initialized")

	hubs := initial.InitHubs()
	initial.InitHubManager(hubs)
	zap.S().Infof("game hubs initialized")

	Router := initial.InitRouters()
	zap.S().Infof("router initialized")

	err := Router.Run(":8080")
	if err != nil {
		zap.S().Panicf("fail to start web server")
	}
}
