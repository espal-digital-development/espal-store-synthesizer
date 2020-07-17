package packages

// FunctionParameter function's parameter structure.
type FunctionParameter struct {
	name  string
	_type string
}

// FunctionReturnValue function's return value structure.
type FunctionReturnValue struct {
	name  string // Name being set means its a `named return value`
	_type string
}

// Function information object.
type Function struct {
	name         string
	parameters   []*FunctionParameter
	returnValues []*FunctionReturnValue
}

// ContainsNamedReturnValue returns whether the given function contains a named return value like `(err error)`.
func (f *Function) ContainsNamedReturnValue() bool {
	for _, returnValue := range f.returnValues {
		if returnValue.name != "" {
			return true
		}
	}
	return false
}
