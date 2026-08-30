package schemamapper

type propertyPath string

func (p propertyPath) String() string {
	return string(p)
}

func (p propertyPath) Append(chunk string) propertyPath {
	return propertyPath(string(p) + "." + chunk)
}

func (p propertyPath) AppendArray() propertyPath {
	return propertyPath(string(p) + "[]")
}
