package packages

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/juju/errors"
)

// Import block line.
type Import struct {
	path string
}

// Entity information object.
type Entity struct {
	_package      *Package
	store         *Store
	imports       []*Import
	name          string
	variableName  string
	interfaceName string
	// interfaceMethods      []*Function
	extraInterfaceMethods                 []*Function
	properties                            []*Property
	tableName                             string
	tableAlias                            string
	hasPrivateNewMethod                   bool
	hasPublicNewMethod                    bool
	skipPropertiesForInterface            map[string]bool
	skipPropertiesForTranslationInterface map[string]bool
}

// IsPrimaryEntity returns if this entity is the package's primary e.
func (e *Entity) IsPrimaryEntity() bool {
	return strings.ToLower(e.name) == e.PackageName()
}

// ContainsBytesType returns if the entity contains a property of the
// byte-array type.
func (e *Entity) ContainsBytesType() bool {
	for _, property := range e.properties {
		if property._type == "[]byte" {
			return true
		}
	}
	return false
}

// HasOptionalCreator return if the entity has a creatorID field that is
// optional.
func (e *Entity) HasOptionalCreator() bool {
	for _, property := range e.properties {
		if property.name == "createdByID" && property._type == "*string" {
			return true
		}
	}
	return false
}

// IsTranslation returns if this entity is a translation type.
func (e *Entity) IsTranslation() bool {
	return strings.HasSuffix(e.name, "Translation")
}

// PackageName returns the entity's package's name.
func (e *Entity) PackageName() string {
	return e._package.name
}

// Name returns the entity's name.
func (e *Entity) Name() string {
	return e.name
}

// VariableName returns a variable name the entity uses in method bodies.
func (e *Entity) VariableName() string {
	if e.variableName == "" {
		e.variableName = getFirstLetterLowercase(e.name)
	}
	return e.variableName
}

// TestVariableName returns a variable name the entity uses in test method bodies.
func (e *Entity) TestVariableName() string {
	if e.variableName == "" {
		e.variableName = getFirstLetterLowercase(e.name)
	}
	if e.variableName == "t" {
		return "tt"
	}
	return e.variableName
}

// PublicNewFunctionName returns the public New-function for the current e.
func (e *Entity) PublicNewFunctionName() string {
	return "New" + e.interfaceName
}

// BuildFileOutput constructs the full synthesized file output for the current e.
func (e *Entity) BuildFileOutput() ([]byte, error) {
	output := bytes.NewBufferString("// Code generated by espal-store-synthesizer. DO NOT EDIT.\n")
	output.WriteString("package " + e.PackageName() + "\n\n")

	output.WriteString("import (\n")
	if len(e.properties) > 0 {
		output.WriteString("\t" + `"time"` + "\n\n")
	}
	output.WriteString("\t" + `"github.com/espal-digital-development/espal-core/database"` + "\n")
	output.WriteString(")\n\n")

	output.WriteString("var _ " + e.interfaceName + " = &" + e.name + "{}\n\n")

	// Generate the interface
	output.WriteString("type " + e.interfaceName + " interface {\n")
	output.WriteString("\t")
	if e.IsTranslation() {
		output.WriteString("database.TranslationModel\n")
	} else {
		output.WriteString("database.Model")
		if e.HasOptionalCreator() {
			output.WriteString("WithOptionalCreator")
		}
		output.WriteString("\n")
	}
	for _, property := range e.properties {
		if _, ok := e.skipPropertiesForInterface[property.Name()]; ok {
			continue
		}

		if e.IsTranslation() {
			if _, ok := e.skipPropertiesForTranslationInterface[property.Name()]; ok {
				continue
			}
		}

		// Getter
		output.WriteString("\t")
		if property.Name() == "id" {
			output.WriteString("ID")
		} else {
			output.WriteString(property.GetterName())
		}
		output.WriteString("() " + property.Type() + "\n")

		// Setter (no Setter for `id`)
		if property.Name() == "id" {
			continue
		}
		output.WriteString("\t")
		output.WriteString(property.SetterName() + "(" + property.Name() + " " + property.Type() + ")\n")
	}

	for _, interfaceMethod := range e.extraInterfaceMethods {
		output.WriteString("\t")
		output.WriteString(interfaceMethod.name)
		output.WriteString("(")
		var firstHad bool
		for _, parameter := range interfaceMethod.parameters {
			if firstHad {
				output.WriteString(", ")
			} else {
				firstHad = true
			}
			if parameter.name != "" {
				output.WriteString(parameter.name)
				output.WriteString(" ")
			}
			output.WriteString(parameter._type)
		}
		output.WriteString(")")

		if len(interfaceMethod.returnValues) > 0 {
			output.WriteString(" ")
			if interfaceMethod.ContainsNamedReturnValue() {
				output.WriteString("(")
			}
			var firstHad bool
			for _, returnValue := range interfaceMethod.returnValues {
				if firstHad {
					output.WriteString(", ")
				} else {
					firstHad = true
				}
				if returnValue.name != "" {
					output.WriteString(returnValue.name)
					output.WriteString(" ")
				}
				output.WriteString(returnValue._type)
			}
			if interfaceMethod.ContainsNamedReturnValue() {
				output.WriteString(")")
			}
		}

		output.WriteString("\n")
	}

	output.WriteString("}")

	// Generate the Setters and Getters
	if e.IsPrimaryEntity() {
		if e.tableName == "" {
			output.WriteString("\n\n")
			output.WriteString("// TableName returns the table name that belongs to the current model.\n")
			output.WriteString("func (" + e.VariableName() + " *" + e.name + ") TableName() string {\n")
			output.WriteString("\t" + `return "` + e.name + `"` + "\n")
			output.WriteString("}")
		}
		if e.tableAlias == "" {
			lowerCase, err := regexp.Compile(`[a-z]`)
			if err != nil {
				return nil, errors.Trace(err)
			}
			alias := strings.ToLower(lowerCase.ReplaceAllString(e.interfaceName, ""))
			output.WriteString("\n\n")
			output.WriteString("// TableAlias returns the unique resolved table alias for use in queries.\n")
			output.WriteString("func (" + e.VariableName() + " *" + e.name + ") TableAlias() string {\n")
			output.WriteString("\t" + `return "` + alias + `"` + "\n")
			output.WriteString("}")
		}
	}

	output.WriteString("\n")

	if len(e.properties) > 0 {
		output.WriteString("\n")
	}

	var firstHad bool
	for _, property := range e.properties {
		if firstHad && property.Name() != "createdByID" {
			output.WriteString("\n")
		} else {
			firstHad = true
		}

		// Getter
		output.WriteString("// ")
		if property.Name() == "id" {
			output.WriteString("ID")
		} else {
			output.WriteString(property.GetterName())
		}
		output.WriteString(" returns " + property.Name() + ".\n")
		output.WriteString("func (" + e.VariableName() + " *" + e.name + ") ")
		if property.Name() == "id" {
			output.WriteString("ID")
		} else {
			output.WriteString(property.GetterName())
		}
		output.WriteString("() " + property.Type() + " {\n")
		output.WriteString("\treturn " + e.VariableName() + "." + property.Name() + "\n")
		output.WriteString("}\n\n")

		// Setter (no Setter for `id`)
		if property.Name() == "id" {
			continue
		}
		output.WriteString("// ")
		if property.Name() == "id" {
			output.WriteString("SetID")
		} else {
			output.WriteString(property.SetterName())
		}
		output.WriteString(" sets the " + property.Name() + ".\n")
		if e.variableName == property.Name() {
			output.WriteString("func (" + e.variableName + "Entity *" + e.name + ") ")
		} else {
			output.WriteString("func (" + e.variableName + " *" + e.name + ") ")
		}
		if property.Name() == "id" {
			output.WriteString("SetID ")
		} else {
			output.WriteString(property.SetterName())
		}
		output.WriteString("(" + property.Name() + " " + property.Type() + ") {\n")
		output.WriteString("\t")
		if e.variableName == property.Name() {
			output.WriteString(e.variableName + "e." + property.Name() + " = " + property.Name() + "\n")
		} else {
			output.WriteString(e.variableName + "." + property.Name() + " = " + property.Name() + "\n")
		}
		output.WriteString("}\n")

		// Add extra methods at the end.
		if property.Name() == "updatedBySurname" {
			output.WriteString("\n")
			output.WriteString("// IsUpdated returns true if UpdatedByID is set.\n")
			output.WriteString("func (" + e.VariableName() + " *" + e.name + ") IsUpdated() bool {\n")
			output.WriteString("\treturn " + e.VariableName() + ".updatedByID != nil\n")
			output.WriteString("}\n")
		}
	}

	// TODO :: 7777 Automate more tests like the internal fetch() (if used already) should be tested in all ways,
	// but this needs at least one Get* function in each store. Maybe create GetOne(), Delete(), etc. by default
	// too so those tests can be generated too

	// TODO :: 7777 Generate mocks for all stores too. Don't run go generate tho, as it will takes ages.
	// It's only a simple helper. Each run should mark the file as generated (don't edit) and remove the
	// previous folder and all it's contents first.

	if !e.hasPrivateNewMethod {
		output.WriteString("\n")
		output.WriteString("func new" + e.name + "() *" + e.name + " {\n")
		output.WriteString("\treturn &" + e.name + "{}\n")
		output.WriteString("}\n")
	}
	if !e.hasPublicNewMethod {
		output.WriteString("\n")
		output.WriteString("// New returns a new instance of " + e.interfaceName + ".\n")
		output.WriteString("func New" + e.interfaceName + "() " + e.interfaceName + " {\n")
		output.WriteString("\treturn new" + e.name + "()\n")
		output.WriteString("}\n")
	}

	return output.Bytes(), nil
}

// BuildTestFileOutput constructs the full synthesized test file output for the current e.
func (e *Entity) BuildTestFileOutput() ([]byte, error) {
	output := bytes.NewBufferString("// Code generated by espal-store-synthesizer. DO NOT EDIT.\n")
	output.WriteString("package " + e.PackageName() + "_test\n\n")

	output.WriteString("import (\n")
	if e.ContainsBytesType() {
		output.WriteString("\t" + `"bytes"` + "\n")
	}
	output.WriteString("\t" + `"testing"` + "\n")
	if len(e.properties) > 0 {
		output.WriteString("\t" + `"time"` + "\n\n")
	}
	output.WriteString("\t" + `"` + e._package.importPath + `"` + "\n")
	output.WriteString(")\n\n")

	output.WriteString("func Test" + e.name + "Table(t *testing.T) {\n")
	output.WriteString("\t" + e.TestVariableName() + " := " + e.PackageName() + "." + e.PublicNewFunctionName() + "()\n")
	output.WriteString("\t" + `if ` + e.TestVariableName() + `.TableName() == "" {` + "\n")
	output.WriteString("\t\tt.Fatal(" + `"TableName shouldn't be empty"` + ")\n")
	output.WriteString("\t}\n")
	output.WriteString("}\n\n")

	output.WriteString("func Test" + e.name + "TableAlias(t *testing.T) {\n")
	output.WriteString("\t" + e.TestVariableName() + " := " + e.PackageName() + "." + e.PublicNewFunctionName() + "()\n")
	output.WriteString("\t" + `if ` + e.TestVariableName() + `.TableName() == "" {` + "\n")
	output.WriteString("\t\tt.Fatal(" + `"TableAlias shouldn't be empty"` + ")\n")
	output.WriteString("\t}\n")
	output.WriteString("}\n\n")

	output.WriteString("func Test" + e.name + "IsUpdated(t *testing.T) {\n")
	output.WriteString("\t" + e.TestVariableName() + " := " + e.PackageName() + "." + e.PublicNewFunctionName() + "()\n")
	output.WriteString("\t" + e.TestVariableName() + ".IsUpdated()\n")
	output.WriteString("}\n\n")

	output.WriteString("func Test" + e.name + "ID(t *testing.T) {\n")
	output.WriteString("\t" + e.TestVariableName() + " := " + e.PackageName() + "." + e.PublicNewFunctionName() + "()\n")
	output.WriteString("\t" + e.TestVariableName() + ".ID()\n")
	output.WriteString("}\n")

	e.processProperties(output)

	return output.Bytes(), nil
}

func (e *Entity) addImport(path string) {
	for _, imp := range e.imports {
		if imp.path == path {
			return
		}
	}
	e.imports = append(e.imports, &Import{path: path})
}

func (e *Entity) processProperties(output io.StringWriter) {
	for _, property := range e.properties {
		if property.Name() == "id" {
			continue
		}
		if strings.Contains(property.Comment(), "@synthesize-no-db-field") {
			continue
		}

		output.WriteString("\n")
		output.WriteString("func Test" + e.name + property.GetterName() + "(t *testing.T) {\n")
		output.WriteString("\t" + e.TestVariableName() + " := " + e.PackageName() + "." + e.PublicNewFunctionName() + "()\n")
		output.WriteString("\ttestValue := ")

		var isBytesType bool

		switch property.Type() {
		case "float32", "*float32":
			output.WriteString(`float32(3.14)`)
		case "float64", "*float64":
			output.WriteString(`6.28`)
		case "uint8", "*uint8":
			output.WriteString(`uint8(255)`)
		case "uint16", "*uint16":
			output.WriteString(`uint16(65000)`)
		case "uint32", "*uint32":
			output.WriteString(`uint32(1e6)`)
		case "uint", "*uint":
			output.WriteString(`uint(1e9)`)
		case "int", "*int":
			output.WriteString(`int(1e8)`)
		case "string", "*string":
			output.WriteString(`"testValue"`)
		case "bool", "*bool":
			output.WriteString(`true`)
		case "time.Time", "*time.Time":
			output.WriteString(`time.Now()`)
		case "time.Duration", "*time.Duration":
			output.WriteString(`time.Second*8`)
		case "[]byte":
			isBytesType = true
			output.WriteString(`[]byte("testData")`)
		default:
			output.WriteString("#FAULT")
		}
		output.WriteString("\n")

		output.WriteString("\t" + e.TestVariableName() + "." + property.SetterName() + "(")

		if []byte(property.Type())[0] == '*' {
			output.WriteString("&")
		}

		output.WriteString("testValue)\n")
		output.WriteString("\tif ")
		if isBytesType {
			output.WriteString("!bytes.Equal(testValue, " + e.TestVariableName() + "." + property.GetterName() + "()) {\n")
		} else {
			if []byte(property.Type())[0] == '*' {
				output.WriteString("&")
			}
			output.WriteString("testValue != " + e.TestVariableName() + "." + property.GetterName() + "() {\n")
		}
		output.WriteString("\t\tt.Fatal(" + `"Getter did not return the Set value"` + ")\n")
		output.WriteString("\t}\n")
		output.WriteString("}\n")
	}
}

func newEntity(p *Package, bytesData []byte) *Entity {
	return &Entity{
		_package:            p,
		properties:          []*Property{},
		hasPrivateNewMethod: bytes.Contains(bytesData, []byte("\nfunc new(")),
		hasPublicNewMethod:  bytes.Contains(bytesData, []byte("\nfunc New(")),
		skipPropertiesForInterface: map[string]bool{
			"id":                 true,
			"createdByID":        true,
			"updatedByID":        true,
			"createdAt":          true,
			"updatedAt":          true,
			"createdByFirstName": true,
			"createdBySurname":   true,
			"updatedByFirstName": true,
			"updatedBySurname":   true,
		},
		skipPropertiesForTranslationInterface: map[string]bool{
			"language": true,
			"field":    true,
			"value":    true,
		},
	}
}
