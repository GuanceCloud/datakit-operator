// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/git"
)

var log = logger.DefaultSLogger("main")

func initLogger() {
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
	log.Info("parse configuration successfully")

	initLogger()
	log.Info("datakit-operator start")
	log.Infof("buildAt: %s, version: %s, commit: %s, branch: %s",
		git.BuildAt, git.Version, git.Commit, git.Branch)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := apis.Run(ctx, config.Cfg.ServerListen); err != nil {
		log.Error(err)
	}

	time.Sleep(100 * time.Millisecond)
}
