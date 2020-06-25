package packages

import (
	"strings"
)

// Property for an entity structure.
type Property struct {
	name    string
	_type   string
	comment string
}

// Name returns the property's name.
func (p *Property) Name() string {
	return p.name
}

// SetName sets the property's name.
func (p *Property) SetName(name string) {
	p.name = name
}

// Type returns the property's type.
func (p *Property) Type() string {
	return p._type
}

// SetType sets the property's type.
func (p *Property) SetType(_type string) {
	p._type = _type
}

// Comment returns the property's comment.
func (p *Property) Comment() string {
	return p.comment
}

// SetComment sets the property's comment.
func (p *Property) SetComment(comment string) {
	p.comment = comment
}

// GetterName returns the property's getter method name for the entity.
func (p *Property) GetterName() string {
	if p.name == "_type" {
		return "Type"
	}
	return strings.Title(p.name)
}

// SetterName returns the property's setter method name for the entity.
func (p *Property) SetterName() string {
	if p.name == "_type" {
		return "SetType"
	}
	return "Set" + strings.Title(p.name)
}
