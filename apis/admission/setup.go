package admission

import (
	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/admission/injector"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/admission/mutator"
)

var l = logger.DefaultSLogger("admission")

func Setup() {
	l = logger.SLogger("amission")
	injector.Setup()
	mutator.Setup()
}
