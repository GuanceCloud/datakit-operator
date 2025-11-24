// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package envbuilder

import (
	"regexp"
	"strings"
)

const (
	qnameCharFmt           string = "[A-Za-z0-9]"
	qnameExtCharFmt        string = "[-A-Za-z0-9_.]"
	qualifiedNameFmt       string = "(" + qnameCharFmt + qnameExtCharFmt + "*)?" + qnameCharFmt
	qualifiedNameMaxLength int    = 63
)

var qualifiedNameRegexp = regexp.MustCompile("^" + qualifiedNameFmt + "$")

func IsQualifiedName(value string) bool {
	parts := strings.Split(value, "/")
	var name string
	switch len(parts) {
	case 1:
		name = parts[0]
	case 2:
		var prefix string
		prefix, name = parts[0], parts[1]

		switch len(prefix) {
		case 0:
			return false
		default:
			if !IsDNS1123Subdomain(prefix) {
				return false
			}
		}
	default:
		return false
	}

	if len(name) == 0 {
		return false
	} else if len(name) > qualifiedNameMaxLength {
		return false
	}
	if !qualifiedNameRegexp.MatchString(name) {
		return false
	}

	return true
}

const (
	dns1123LabelFmt           string = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt       string = dns1123LabelFmt + "(\\." + dns1123LabelFmt + ")*"
	DNS1123SubdomainMaxLength int    = 253
)

var dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")

func IsDNS1123Subdomain(value string) bool {
	if len(value) > DNS1123SubdomainMaxLength {
		return false
	}
	if !dns1123SubdomainRegexp.MatchString(value) {
		return false
	}
	return true
}
