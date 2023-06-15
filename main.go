package main

import (
	"sync"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

var log = logger.DefaultSLogger("main")

func initlogger() {
	lopt := &logger.Option{
		Flags: logger.OPT_DEFAULT | logger.OPT_STDOUT,
		Level: config.Cfg.LogLevel,
	}

	if err := logger.InitRoot(lopt); err != nil {
		log.Fatal(err)
	}

	log = logger.SLogger("main")
}

func main() {
	if err := config.LoadConfigWithEnv(); err != nil {
		log.Error(err)
		return
	}
	log.Info("load configuration successfully")

	initlogger()
	log.Info("datakit-operator start")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		apis.Run(config.Cfg.ServerListen)
	}()
	wg.Wait()
}
