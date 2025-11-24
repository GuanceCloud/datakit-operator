// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"context"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook/injector"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook/mutator"
)

var log = logger.DefaultSLogger("admission")

func Setup(_ context.Context) {
	log = logger.SLogger("admission")

	injector.Setup()
	mutator.Setup()
}
