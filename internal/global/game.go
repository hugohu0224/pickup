package global

import (
	"github.com/spf13/viper"
	"sync"
)

var (
	Dv           *viper.Viper // default config
	Gv           *viper.Viper // google client config
	UserTokenMap sync.Map
)
