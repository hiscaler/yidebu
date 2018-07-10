package config

import (
	"github.com/go-ozzo/ozzo-config"
	"path/filepath"
	"fmt"
)

func Instance() *config.Config {
	conf := config.New()
	confFile, _ := filepath.Abs("src/conf.json")
	error := conf.Load(confFile)
	if error != nil {
		fmt.Println(error)
	}
	return conf
}
