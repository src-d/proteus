package scanner

import (
	"regexp"
	"strings"
)

var protoTagRegex = regexp.MustCompile(`proteus:"([^"]+)"`)

func findProtoTags(tag string) []string {
	if !protoTagRegex.MatchString(tag) {
		return nil
	}

	tags := strings.Split(protoTagRegex.FindStringSubmatch(tag)[1], ",")
	for i, t := range tags {
		tags[i] = strings.TrimSpace(t)
	}
	return tags
}
