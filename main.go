package main

import (
	"flag"
	"sync"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/admission"
)

var l = logger.DefaultSLogger("main")

var (
	flagServerPort = flag.Int("port", 9543, "Listen TLS server port")
	flagLogLevel   = flag.String("level", "debug", "Output log level")
)

func initlogger() {
	lopt := &logger.Option{
		Flags: logger.OPT_DEFAULT | logger.OPT_STDOUT,
		Level: *flagLogLevel,
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

		admission.Run(*flagServerPort)
	}()
	wg.Wait()
}
