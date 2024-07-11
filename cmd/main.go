package main

import (
	"go.uber.org/zap"
	"pickup/internal/initial"
)

func main() {

	initial.InitLogger()
	initial.InitRouters()
	hubs := initial.InitHubs()
	initial.InitHubManager(hubs)
	Router := initial.InitRouters()

	err := Router.Run(":8080")
	if err != nil {
		zap.S().Panicf("fail to start web server")
	}
}
