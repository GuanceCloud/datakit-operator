package admission

import "strings"

func Unique(slice []string) []string {
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
