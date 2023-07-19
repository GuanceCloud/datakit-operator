package injector

import "gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"

var l = logger.DefaultSLogger("injector")

func Setup() {
	l = logger.SLogger("injector")
}
