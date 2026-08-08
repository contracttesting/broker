package dsl

import "strings"

type PropertyPath string

func NewPropertyPath(propertyPath string) PropertyPath {
	return PropertyPath(propertyPath)
}

func (f *PropertyPath) String() string {
	return string(*f)
}

func (f *PropertyPath) Append(chunk string) PropertyPath {

	if string(*f) == "" {
		return PropertyPath(chunk)
	}

	return PropertyPath(strings.Join([]string{string(*f), chunk}, "."))
}

func (f *PropertyPath) AppendArray() PropertyPath {
	return PropertyPath(f.String() + "[]")
}
