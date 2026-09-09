// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"fmt"
	"os"
	"strings"

	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
)

func ResolveLeaseNamespace() (string, error) {
	namespace := strings.TrimSpace(os.Getenv("POD_NAMESPACE"))
	if namespace == "" {
		data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return "", fmt.Errorf("resolve Operator namespace: %w", err)
		}
		namespace = strings.TrimSpace(string(data))
	}
	if problems := utilvalidation.IsDNS1123Label(namespace); len(problems) > 0 {
		return "", fmt.Errorf("invalid Operator namespace %q: %s", namespace, strings.Join(problems, "; "))
	}
	return namespace, nil
}
