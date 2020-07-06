package packages

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/juju/errors"
	"github.com/mattn/go-zglob"
)

// Package wrapping store structure.
type Package struct {
	mainEntity *Entity
	entities   []*Entity
	store      *Store
	path       string
	importPath string
	name       string

	reLowerCase                 *regexp.Regexp
	rePropertyLinesCheck        *regexp.Regexp
	reSynthesizeOccurrences     *regexp.Regexp
	reStructBlockCheck          *regexp.Regexp
	reExtraInterfaceMethods     *regexp.Regexp
	reInterfaceMethodLinesCheck *regexp.Regexp
	reTableNameCheck            *regexp.Regexp
	reTableAliasCheck           *regexp.Regexp

	rePublicMethodsCheck     *regexp.Regexp
	reImportBlockCheck       *regexp.Regexp
	reImportStatementsCheck  *regexp.Regexp
	reStoreStructBlockCheck  *regexp.Regexp
	reServicesCheck          *regexp.Regexp
	rePackagesInMethodsCheck *regexp.Regexp
}

// Path returns the package's location on the disk.
func (p *Package) Path() string {
	return p.path
}

// Store returns the package's store object.
func (p *Package) Store() *Store {
	return p.store
}

// MainEntity returns the package's main entity.
func (p *Package) MainEntity() *Entity {
	return p.mainEntity
}

// Entities returns all the package's entities, excluding the main entity.
func (p *Package) Entities() []*Entity {
	return p.entities
}

// BuildMetaData collects all the package information from the path
// and builds and fills the necessary objects.
func (p *Package) BuildMetaData(path string) error {
	entries, err := zglob.Glob(path + "/*.go")
	if err != nil {
		return errors.Trace(err)
	}

	chunks := strings.Split(path, "/")
	p.path = path
	p.name = chunks[len(chunks)-1]

	importPath := strings.Builder{}
	importPath.WriteString("github.com/espal-digital-development/espal-core/stores")
	var startCollecting bool
	for _, chunk := range chunks {
		if chunk == "stores" {
			startCollecting = true
			continue
		} else if !startCollecting {
			continue
		}
		importPath.WriteString("/" + chunk)
	}
	p.importPath = importPath.String()

	// Wipe any existing synthetized files in case a new structure is chosen
	for _, entry := range entries {
		if strings.Contains(entry, "_synthesized") {
			if err := os.Remove(entry); err != nil {
				return errors.Trace(err)
			}
		}
	}

	// Store and primary entity first
	var hasStoreFile bool
	var hasEntityFile bool
	var mainEntityFileName string
	for _, entry := range entries {
		switch {
		case strings.HasSuffix(entry, "store.go"):
			hasStoreFile = true
		case strings.HasSuffix(entry, p.name+".go"):
			hasEntityFile = true
			mainEntityFileName = p.name
			// TODO :: These extra checks could just become:
			// - @synthesize-store
			// - @synthesize-main-entity
			// This would be much more implicit and doesn't need exceptions in this parser
		case strings.HasSuffix(entry, p.name+"entity.go"):
			// This suffix variant is unique as Store as an entity is equally reserved as the Store (datastore) object name
			hasEntityFile = true
			mainEntityFileName = p.name + "entity"
		case p.name == "product" && strings.HasSuffix(entry, "model.go"):
			// A weird spinoff for a more complex store where the Model is the main entity and not the package name's equivalent
			hasEntityFile = true
			mainEntityFileName = "model"
		}
	}

	if !hasStoreFile {
		return errors.Errorf("`%s` doesn't have a store file", path)
	}
	if !hasEntityFile {
		return errors.Errorf("`%s` doesn't have an entity file", path)
	}

	entityFileBytes, err := ioutil.ReadFile(path + "/" + mainEntityFileName + ".go")
	if err != nil {
		return errors.Trace(err)
	}
	if err = p.setMainEntityFromFile(entityFileBytes); err != nil {
		return errors.Trace(err)
	}
	storeFileBytes, err := ioutil.ReadFile(path + "/store.go")
	if err != nil {
		return errors.Trace(err)
	}
	if err := p.storeFromFile(storeFileBytes); err != nil {
		return errors.Trace(err)
	}

	// Check any other files for methods for the store
	for _, entry := range entries {
		if strings.HasSuffix(entry, "_synthesized.go") {
			continue
		}
		if strings.HasSuffix(entry, "_test.go") {
			continue
		}
		if strings.HasSuffix(entry, "store.go") {
			continue
		}
		if strings.HasSuffix(entry, p.name+".go") {
			continue
		}
		fileBytes, err := ioutil.ReadFile(entry)
		if err != nil {
			return errors.Trace(err)
		}
		// Skip files that already doing synthesis or have `@synthesize-ignore`
		if bytes.Contains(fileBytes, []byte("@synthesize")) {
			continue
		}

		importBlock := p.reImportBlockCheck.FindAllSubmatch(fileBytes, 1)
		importStatements := p.reImportStatementsCheck.FindAllSubmatch(importBlock[0][1], -1)
		for k := range importStatements {
			// fmt is a bogus import and unneeded for interface methods
			if string(importStatements[k][1]) == "fmt" {
				continue
			}
			p.store.addImport(&Import{path: string(importStatements[k][1])})
		}

		storeMethods := p.rePublicMethodsCheck.FindAllSubmatch(fileBytes, -1)
		for _, method := range storeMethods {
			function := &Function{
				name: string(method[1]),
			}

			parameters := bytes.Split(method[2], []byte(", "))
			for _, parameter := range parameters {
				parameterParts := bytes.SplitN(parameter, []byte(" "), 2)
				function.parameters = append(function.parameters, &FunctionParameter{
					name:  string(parameterParts[0]),
					_type: string(parameterParts[1]),
				})
			}

			returnValues := bytes.Split(bytes.TrimRight(bytes.TrimRight(bytes.TrimLeft(
				bytes.Trim(method[3], " "), "("), " {"), ")"), []byte(", "))
			for _, returnValue := range returnValues {
				returnValueParts := bytes.SplitN(returnValue, []byte(" "), 2)
				function.returnValues = append(function.returnValues, &FunctionReturnValue{
					name:  string(returnValueParts[0]),
					_type: string(returnValueParts[1]),
				})
			}

			p.store.methods = append(p.store.methods, function)
		}
	}

	p.store.mainEntity = p.mainEntity
	p.mainEntity.store = p.store

	for _, entry := range entries {
		// Extra safety measure to always ignore synthetized files
		if strings.Contains(entry, "_synthesized") {
			continue
		}
		// No need to inspect test files
		if strings.Contains(entry, "_test") {
			continue
		}

		// Skip store and main entity
		if entry == path+"/store.go" {
			continue
		}
		if entry == path+"/"+p.name+".go" {
			continue
		}

		fileBytes, err := ioutil.ReadFile(entry)
		if err != nil {
			return errors.Trace(err)
		}

		// Cheap checks, but ok as consistent pattern is upheld
		if bytes.Contains(fileBytes, []byte("// @synthesize\n")) {
			if err := p.addEntityFromFile(fileBytes); err != nil {
				return errors.Trace(err)
			}
		} else {
			// // TODO :: 77777 :: Misc files need to add to the correct Entity to also share things like the imports
			if err := p.addEntityFromFile(fileBytes); err != nil {
				continue
			}
		}
	}

	return nil
}

func (p *Package) storeFromFile(b []byte) error {
	if p.mainEntity == nil {
		return errors.Errorf("cannot set the store before the main entity is known")
	}
	p.store = newStore(p, p.mainEntity, bytes.Contains(b, []byte("\nfunc new(")),
		bytes.Contains(b, []byte("\nfunc New(")), bytes.Contains(b, []byte(" buildQueries() ")))

	structBlockMatches := p.reStoreStructBlockCheck.FindAllSubmatch(b, 1)
	if len(structBlockMatches) != 1 {
		return errors.Errorf("not one struct found in `%s`", p.name)
	}
	p.store.structName = string(structBlockMatches[0][1])

	alreadyImported := make(map[string]bool)
	importBlock := p.reImportBlockCheck.FindAllSubmatch(b, 1)
	importStatements := p.reImportStatementsCheck.FindAllSubmatch(importBlock[0][1], -1)

	methods := p.rePublicMethodsCheck.FindAllSubmatch(b, -1)
	services := p.reServicesCheck.FindAllSubmatch(structBlockMatches[0][2], -1)

	if (len(services) > 0 || len(methods) > 0) && !bytes.Contains(b, []byte("\nfunc New(")) {
		for _, service := range services {
			p.store.services = append(p.store.services, &Service{
				name:        string(service[1]),
				packageName: string(service[2]),
			})
			for _, statement := range importStatements {
				if _, ok := alreadyImported[string(statement[1])]; ok {
					continue
				}
				parts := strings.Split(string(statement[1]), "/")
				packagePart := parts[len(parts)-1]
				if strings.HasPrefix(string(service[2]), packagePart) {
					p.store.imports = append(p.store.imports, &Import{
						path: string(statement[1]),
					})
					alreadyImported[string(statement[1])] = true
					break
				}
			}
		}

		for _, method := range methods {
			// Build the method meta data
			function := &Function{
				name:         string(method[1]),
				parameters:   make([]*FunctionParameter, 0),
				returnValues: make([]*FunctionReturnValue, 0),
			}
			// Any parameters?
			if len(method[2]) > 0 {
				parametersChunks := bytes.Split(method[2], []byte(", "))
				for _, parametersChunk := range parametersChunks {
					parameterChunks := bytes.SplitN(parametersChunk, []byte(" "), 2)
					function.parameters = append(function.parameters, &FunctionParameter{
						name:  string(parameterChunks[0]),
						_type: string(parameterChunks[1]),
					})
				}
			}
			// Any return values? (position 5 = grouped return values)
			if len(method[5]) > 0 {
				returnValuesChunks := bytes.Split(method[5], []byte(", "))
				for _, returnValuesChunk := range returnValuesChunks {
					returnValueChunks := bytes.SplitN(returnValuesChunk, []byte(" "), 2)
					returnValue := &FunctionReturnValue{}
					switch {
					case len(returnValueChunks) == 1:
						returnValue._type = string(returnValueChunks[0])
					case len(returnValueChunks) == 2:
						returnValue.name = string(returnValueChunks[0])
						returnValue._type = string(returnValueChunks[1])
					default:
						return errors.Errorf("return values for `%s` : `%s` should be either 1 or 2, not %d",
							p.name, p.store.structName, len(returnValueChunks))
					}
					function.returnValues = append(function.returnValues, returnValue)
				}
			} else if len(method[4]) > 0 { // (position 4 = single free floating return value)
				function.returnValues = append(function.returnValues, &FunctionReturnValue{
					_type: string(method[4]),
				})
			}
			p.store.methods = append(p.store.methods, function)

			// If the method definition doesn't contain a dot (package call) then skip
			if !bytes.Contains(method[2], []byte(".")) {
				continue
			}

			packagesInMethod := p.rePackagesInMethodsCheck.FindAllSubmatch(method[0], -1)
			for _, packageInMethod := range packagesInMethod {
				for _, statement := range importStatements {
					if _, ok := alreadyImported[string(statement[1])]; ok {
						continue
					}
					parts := strings.Split(string(statement[1]), "/")
					packagePart := parts[len(parts)-1]
					if strings.HasPrefix(string(packageInMethod[0]), packagePart) {
						p.store.imports = append(p.store.imports, &Import{
							path: string(statement[1]),
						})
						alreadyImported[string(statement[1])] = true
						break
					}
				}
			}
		}
	}

	if !p.store.ContainsFetchMethod() {
		// Stores generally always handle errors so it uses the main wrapping library
		p.store.imports = append(p.store.imports, &Import{
			path: "github.com/juju/errors",
		})

		var databaseAlreadyImported bool
		var databaseSQLAlreadyImported bool
		for _, importChunk := range p.store.imports {
			if importChunk.path == "github.com/espal-digital-development/espal-core/database" {
				databaseAlreadyImported = true
			}
			if importChunk.path == "database/sql" {
				databaseSQLAlreadyImported = true
			}
		}
		if !databaseAlreadyImported {
			p.store.imports = append(p.store.imports, &Import{
				path: "github.com/espal-digital-development/espal-core/database",
			})
		}
		if !databaseSQLAlreadyImported {
			p.store.imports = append(p.store.imports, &Import{
				path: "database/sql",
			})
		}
	}

	return nil
}

func (p *Package) setMainEntityFromFile(b []byte) (err error) {
	p.mainEntity, err = p.entityFromFile(b)
	return
}

func (p *Package) addEntityFromFile(b []byte) error {
	entity, err := p.entityFromFile(b)
	if err != nil {
		return errors.Trace(err)
	}
	p.entities = append(p.entities, entity)
	return nil
}

func (p *Package) entityFromFile(b []byte) (*Entity, error) {
	entity := newEntity(p, b)

	occurrences := len(p.reSynthesizeOccurrences.FindAll(b, 2))
	if occurrences != 1 {
		return nil, errors.Errorf("`%s` entity files must have one and only one @synthesize marking", p.path)
	}

	structBlockMatches := p.reStructBlockCheck.FindSubmatch(b)

	entity.name = string(structBlockMatches[1])
	entity.interfaceName = strings.Title(entity.name) + "Entity"

	structLines := p.reStructBlockCheck.Find(b)
	lines := p.rePropertyLinesCheck.FindAllSubmatch(structLines, -1)
	for _, line := range lines {
		property := &Property{
			name: string(line[1]),
		}
		if bytes.Contains(line[2], []byte("//")) {
			chunks := bytes.Split(line[2], []byte("//"))
			if len(chunks) == 2 {
				property._type = string(bytes.TrimRight(chunks[0], " "))
				property.comment = string(bytes.Trim(chunks[1], " "))
			} else {
				return nil, errors.Errorf("found more than 2 chunks. This is probably caused by multiple comment //.")
			}
		} else {
			property._type = string(bytes.TrimRight(line[2], " "))
		}
		entity.properties = append(entity.properties, property)
	}

	if entity.ContainsBytesType() {
		entity.addImport("bytes")
	}

	extraMethods := p.reExtraInterfaceMethods.FindSubmatch(b)
	if len(extraMethods) == 3 && bytes.Equal(extraMethods[1], []byte(strings.ToLower(entity.name))) {
		methods := p.reInterfaceMethodLinesCheck.FindAllSubmatch(b, -1)
		for _, method := range methods {
			interfaceMethod := &Function{
				parameters: []*FunctionParameter{},
			}
			interfaceMethod.name = string(method[1])

			parametersChunks := strings.Split(string(bytes.Trim(method[2], " ")), ",")
			if len(parametersChunks) > 0 && parametersChunks[0] != "" {
				for _, parametersChunk := range parametersChunks {
					parameter := strings.Split(strings.Trim(parametersChunk, " "), " ")
					interfaceMethod.parameters = append(interfaceMethod.parameters, &FunctionParameter{
						name:  parameter[0],
						_type: parameter[1],
					})
				}
			}

			returnValuesChunks := strings.Split(string(bytes.TrimRight(bytes.TrimLeft(
				bytes.Trim(method[3], " "), "("), ")")), ",")
			if len(returnValuesChunks) > 0 && returnValuesChunks[0] != "" {
				for _, returnValuesChunk := range returnValuesChunks {
					returnValue := strings.Split(strings.Trim(returnValuesChunk, " "), " ")
					if len(returnValue) == 0 || len(returnValue) > 2 {
						return nil, errors.Errorf("interface method return value should have 1 or 2 parts, got %d", len(returnValue))
					}
					returnValueObj := &FunctionReturnValue{
						name: returnValue[0],
					}
					if len(returnValue) == 2 {
						returnValueObj._type = returnValue[1]
					}
					interfaceMethod.returnValues = append(interfaceMethod.returnValues, returnValueObj)
				}
			}

			entity.extraInterfaceMethods = append(entity.extraInterfaceMethods, interfaceMethod)
		}
	}

	// Register the table name/alias and if it's already uses
	// the needed methods
	if entity.IsPrimaryEntity() {
		tableNameCheck := p.reTableNameCheck.FindSubmatch(b)
		if len(tableNameCheck) > 0 {
			entity.tableName = string(tableNameCheck[1])
		}

		tableAliasCheck := p.reTableAliasCheck.FindSubmatch(b)
		if len(tableAliasCheck) > 0 {
			entity.tableAlias = string(tableAliasCheck[1])
		}
	}

	entity.hasPrivateNewMethod = bytes.Contains(b, []byte("func new"))
	entity.hasPublicNewMethod = bytes.Contains(b, []byte("func New"))

	return entity, nil
}

func getFirstLetterLowercase(s string) string {
	if s == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r))
}

// New returns a new instance of Package.
func New() *Package {
	return &Package{
		reLowerCase:                 regexp.MustCompile(`[a-z]`),
		rePropertyLinesCheck:        regexp.MustCompile(`(?m)^\s+([\w_]\w+)\s+(.{2,}?)$`),
		reSynthesizeOccurrences:     regexp.MustCompile(`\n//\s*@synthesize[\n\s]`),
		reStructBlockCheck:          regexp.MustCompile(`(?s)@synthesize\ntype ([a-zA-Z]+) struct \{\n.*?\}\n`),
		reExtraInterfaceMethods:     regexp.MustCompile(`(?s)type ([a-z]\w+)Methods interface \{\n([^\}]+)\}\n`),
		reInterfaceMethodLinesCheck: regexp.MustCompile(`(?m)^\s+(\w+)\((.*?)\)(.*?)$`),
		reTableNameCheck:            regexp.MustCompile(` TableName\(\) string \{\n\s+return "(.*?)"`),
		reTableAliasCheck:           regexp.MustCompile(` TableAlias\(\) string \{\n\s+return "(.*?)"`),

		rePublicMethodsCheck: regexp.MustCompile(
			`(?ms)^func \(\w+ \*\w+\) ([A-Z]\w+)\((.*?)\)( {$| ([^(][^\s]+) {$| \((.*?)\) {$)`),
		reServicesCheck:          regexp.MustCompile(`\s+(\w+)\s+(\w+\.\w+)`),
		reImportBlockCheck:       regexp.MustCompile(`(?s)import\ \(\n(.*?)\n\)`),
		reStoreStructBlockCheck:  regexp.MustCompile(`(?s)type ([a-zA-Z]+) struct \{\n(.*?)\n\}\n`),
		reImportStatementsCheck:  regexp.MustCompile(`\s+"(.*?)"`),
		rePackagesInMethodsCheck: regexp.MustCompile(`\w+\.\w+`),
	}
}
