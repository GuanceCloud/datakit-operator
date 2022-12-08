package main

import (
	"flag"
	"os"
	"sync"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/admission"
)

var l = logger.DefaultSLogger("main")

var (
	envServerListen = envString("DATAKIT_OPERATOR_SERVER_LISTEN", "0.0.0.0:9543")
	envLogLevel     = envString("DATAKIT_OPERATOR_LOG_LEVEL", "debug")
)

func initlogger() {
	lopt := &logger.Option{
		Flags: logger.OPT_DEFAULT | logger.OPT_STDOUT,
		Level: envLogLevel,
	}

	if err := logger.InitRoot(lopt); err != nil {
		l.Fatal(err)
	}

	l = logger.SLogger("main")
}

func main() {
	flag.Parse()
	initlogger()

	l.Info("datakit-operator start")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		admission.Run(envServerListen)
	}()
	wg.Wait()
}

func envString(name string, value string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return value
}
