package election

import "gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"

var l = logger.DefaultSLogger("election")

func Setup() {
	l = logger.SLogger("election")
}
