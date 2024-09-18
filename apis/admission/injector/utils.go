package injector

import (
	"path/filepath"
	"strconv"
	"strings"
)

// CheckAnnotationIsTrue If no annotation is found or parsing fails, it will return true.
func CheckAnnotationIsTrue(annotations map[string]string, key string) bool {
	str, found := annotations[key]
	if !found {
		return true
	}
	b, err := strconv.ParseBool(str)
	if err != nil {
		return true
	}
	return b
}

func unique(slice []string) []string {
	var uniqMap = make(map[string]struct{})
	var uniqSlice []string
	for _, s := range slice {
		_, exist := uniqMap[s]
		if exist {
			continue
		}
		uniqMap[s] = struct{}{}
		uniqSlice = append(uniqSlice, s)
	}
	return uniqSlice
}

func appendKVPairs(origin, neww string) string {
	var originKVPairs [][]string
	for _, pair := range strings.Split(origin, ",") {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			continue
		}
		originKVPairs = append(originKVPairs, []string{kv[0], kv[1]})
	}

	for _, pair := range strings.Split(neww, ",") {
		kv := strings.Split(pair, ":")
		if len(kv) != 2 {
			continue
		}

		exists := false
		for _, originKV := range originKVPairs {
			if len(originKV) == 0 {
				continue
			}
			if kv[0] == originKV[0] {
				exists = true
				break
			}
		}

		if !exists {
			originKVPairs = append(originKVPairs, kv)
		}
	}

	var res []string

	for _, kv := range originKVPairs {
		res = append(res, strings.Join(kv, ":"))
	}

	return strings.Join(res, ",")
}

// ParseImage adapts some of the logic from the actual Docker library's image parsing
// routines:
// https://github.com/docker/distribution/blob/release/2.7/reference/normalize.go
func ParseImage(image string) (string, string, string) {
	var domain, remainder string

	i := strings.IndexRune(image, '/')

	if i == -1 || (!strings.ContainsAny(image[:i], ".:") && image[:i] != "localhost") {
		remainder = image
	} else {
		domain, remainder = image[:i], image[i+1:]
	}

	var imageName string
	imageVersion := "unknown"

	i = strings.LastIndex(remainder, ":")
	if i > -1 {
		imageVersion = remainder[i+1:]
		imageName = remainder[:i]
	} else {
		imageName = remainder
	}

	if domain != "" {
		imageName = domain + "/" + imageName
	}

	shortName := imageName
	if imageBlock := strings.Split(imageName, "/"); len(imageBlock) > 0 {
		// there is no need to do
		// Split not return empty slice
		shortName = imageBlock[len(imageBlock)-1]
	}

	return imageName, shortName, imageVersion
}

func getMountPaths(paths []string) []string {
	var res []string
	for _, path := range paths {
		if s := getMountPath(path); s != "" {
			res = append(res, s)
		}
	}
	return res
}

func getMountPath(path string) string {
	arr := strings.Split(path, "*")
	if len(arr) != 0 {
		return filepath.Clean(filepath.Dir(arr[0]))
	}
	return ""
}
