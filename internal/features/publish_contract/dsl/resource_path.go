package dsl

import (
	"regexp"
	"strings"
)

type ResourcePath string

func NewResourcePath(resourcePath string) ResourcePath {
	return ResourcePath(resourcePath)
}

func (rp *ResourcePath) Append(parts ...string) ResourcePath {
	separator := ";"

	if string(*rp) == "" {
		return ResourcePath(strings.Join(parts, separator))
	}

	chunks := strings.Join(parts, separator)

	return ResourcePath(strings.Join([]string{string(*rp), chunks}, separator))
}

func (rp *ResourcePath) String() string {
	return string(*rp)
}

func (rp *ResourcePath) Split() []string {
	return strings.Split(rp.String(), ";")
}

func (rp *ResourcePath) IsConsumer() bool {
	return rp.Split()[0] == "consumes"
}

func (rp *ResourcePath) IsProvider() bool {
	return rp.Split()[0] == "provides"
}

func (rp *ResourcePath) ExtractNamedArgs(regex *regexp.Regexp) (map[string]string, bool) {
	match := regex.FindStringSubmatch(rp.String())
	if match == nil {
		return nil, false
	}

	args := make(map[string]string, len(regex.SubexpNames()))
	for i, name := range regex.SubexpNames() {
		if name == "" {
			continue
		}

		args[name] = match[i]
	}

	return args, true
}
