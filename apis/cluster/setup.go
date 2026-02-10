// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package cluster

import (
	"context"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
)

var log = logger.DefaultSLogger("cluster")

func Setup(_ context.Context) {
	log = logger.SLogger("cluster")
}
