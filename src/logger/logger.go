package logger

import (
	"github.com/go-ozzo/ozzo-log"
	"config"
)

var Instance = log.NewLogger()

func init() {
	cfg := config.Instance()
	cfg.Register("ConsoleTarget", log.NewConsoleTarget)
	cfg.Register("FileTarget", log.NewFileTarget)
	if err := cfg.Configure(Instance, "Logger"); err != nil {
		panic(err)
	}
	Instance.Open()
}
