package mutator

import "gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"

var l = logger.DefaultSLogger("mutator")

func Setup() {
	l = logger.SLogger("mutator")
}
