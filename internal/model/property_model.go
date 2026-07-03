package model

type Property struct {
	ID       int64 `json:"-"` // DB id — never identity
	Path     string
	Type     string
	Optional bool
}

func (p *Property) IsSame(other *Property) bool {
	return p.Path == other.Path &&
		p.Type == other.Type &&
		p.Optional == other.Optional
}

func NewProperty(
	propertyPath string,
	propertyType string,
	optional bool,
) Property {
	return Property{
		Path:     propertyPath,
		Type:     propertyType,
		Optional: optional,
	}
}
