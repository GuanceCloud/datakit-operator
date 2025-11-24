// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package logging

import (
	"context"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/logging/config"
)

var log = logger.DefaultSLogger("logging")

func Setup(ctx context.Context) {
	log = logger.SLogger("logging")

	config.Setup(ctx)
}
