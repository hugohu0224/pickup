package initial

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"pickup/internal/global"
)

func InitConfigByViper() {
	// default viper
	global.Dv = viper.New()
	global.Dv.SetConfigName("config")
	global.Dv.SetConfigType("yaml")
	global.Dv.AddConfigPath(".")
	global.Dv.AddConfigPath("../")
	global.Dv.AddConfigPath("../../")

	// read default config
	if err := global.Dv.ReadInConfig(); err != nil {
		zap.S().Fatalf("error reading default config file: %v", err)
	}
	zap.S().Infof("default config file used: %s", global.Dv.ConfigFileUsed())

	// set env
	currentEnv := global.Dv.GetString("current_env")
	if currentEnv == "" {
		panic("current_env not set in config file")
	}
	global.Dv = global.Dv.Sub(currentEnv)

	// google client viper
	global.Gv = viper.New()
	global.Gv.SetConfigName("google_client_secret")
	global.Gv.SetConfigType("json")
	global.Gv.AddConfigPath(".")
	global.Gv.AddConfigPath("../")
	global.Gv.AddConfigPath("../../")

	// read Google client config
	if err := global.Gv.ReadInConfig(); err != nil {
		zap.S().Fatalf("error reading Google client config file: %v", err)
	}
	zap.S().Infof("google client config file used: %s", global.Gv.ConfigFileUsed())
}
