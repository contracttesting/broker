package dsl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/contracttesting/broker/internal/model"
)

var (
	consumerRestRequestRegex = regexp.MustCompile(
		`^consumes;(?P<provider>[^;]*);rest;(?P<endpoint>[^;]+);(?P<method>[^;]+);request$`,
	)
	consumerRestResponseRegex = regexp.MustCompile(
		`^consumes;(?P<provider>[^;]*);rest;(?P<endpoint>[^;]+);(?P<method>[^;]+);responses;(?P<status>\d+)$`,
	)
	providerRestRequestRegex = regexp.MustCompile(
		`^provides;rest;(?P<endpoint>[^;]+);(?P<method>[^;]+);request$`,
	)
	providerRestResponseRegex = regexp.MustCompile(
		`^provides;rest;(?P<endpoint>[^;]+);(?P<method>[^;]+);responses;(?P<status>\d+)$`,
	)
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
	return strings.Contains(rp.String(), "consumes")
}

func (rp *ResourcePath) IsProvider() bool {
	return strings.Contains(rp.String(), "provides")
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

func (rp *ResourcePath) ToResource(properties map[string]model.Property) *model.UploadedResource {
	if args, ok := rp.ExtractNamedArgs(consumerRestRequestRegex); ok {
		return model.NewRestRequestConsumer(
			args["provider"],
			args["endpoint"],
			args["method"],
			properties,
		)
	}

	if args, ok := rp.ExtractNamedArgs(consumerRestResponseRegex); ok {
		return model.NewRestResponseConsumer(
			args["provider"],
			args["endpoint"],
			args["method"],
			args["status"],
			properties,
		)
	}

	if args, ok := rp.ExtractNamedArgs(providerRestRequestRegex); ok {
		return model.NewRestRequestProvider(
			args["endpoint"],
			args["method"],
			properties,
		)
	}

	if args, ok := rp.ExtractNamedArgs(providerRestResponseRegex); ok {
		return model.NewRestResponseProvider(
			args["endpoint"],
			args["method"],
			args["status"],
			properties,
		)
	}

	panic(fmt.Errorf("unrecognized resource path: %q", rp.String()))
}
