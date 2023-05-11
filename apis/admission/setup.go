package admission

import "gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"

var l = logger.DefaultSLogger("admission")

func Setup() {
	l = logger.SLogger("amission")
}
