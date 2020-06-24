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
func (property *Property) Name() string {
	return property.name
}

// SetName sets the property's name.
func (property *Property) SetName(name string) {
	property.name = name
}

// Type returns the property's type.
func (property *Property) Type() string {
	return property._type
}

// SetType sets the property's type.
func (property *Property) SetType(_type string) {
	property._type = _type
}

// Comment returns the property's comment.
func (property *Property) Comment() string {
	return property.comment
}

// SetComment sets the property's comment.
func (property *Property) SetComment(comment string) {
	property.comment = comment
}

// GetterName returns the property's getter method name for the entity.
func (property *Property) GetterName() string {
	if property.name == "_type" {
		return "Type"
	}
	return strings.Title(property.name)
}

// SetterName returns the property's setter method name for the entity.
func (property *Property) SetterName() string {
	if property.name == "_type" {
		return "SetType"
	}
	return "Set" + strings.Title(property.name)
}
