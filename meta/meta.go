package meta

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/espal-digital-development/espal-store-synthesizer/packages"
	"github.com/juju/errors"
)

// Meta package object.
type Meta struct {
	storesMetaPath string
}

// Build builds or refreshes the storesmeta package.
func (m *Meta) Build(packages []*packages.Package) error {
	_, err := os.Stat(m.storesMetaPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Trace(err)
	}
	createDir := err == nil || os.IsNotExist(err)
	if err == nil {
		if err := os.RemoveAll(m.storesMetaPath); err != nil {
			return errors.Trace(err)
		}
	}
	if createDir {
		if err := os.Mkdir(m.storesMetaPath, 0700); err != nil {
			return errors.Trace(err)
		}
	}
	if err := m.createFile(); err != nil {
		return errors.Trace(err)
	}
	if err := m.createTestFile(); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (m *Meta) createFile() error {
	output := &strings.Builder{}
	output.WriteString("package storesmeta\n\n")

	output.WriteString("// StoresMeta object.\n")
	output.WriteString("type StoresMeta struct{}\n\n")

	// for _, pkg := range packages {
	// 	entities := pkg.Entities()
	// 	for _, entity := range entities {

	// 	}
	// }

	output.WriteString("// New returns a new instance of StoresMeta.\n")
	output.WriteString("func New() (*StoresMeta, error) {\n")
	output.WriteString("\treturn &StoresMeta{}, nil\n")
	output.WriteString("}\n")

	return errors.Trace(ioutil.WriteFile(m.storesMetaPath+"/storesmeta.go", []byte(output.String()), 0600))
}

func (m *Meta) createTestFile() error {
	output := &strings.Builder{}
	output.WriteString("package storesmeta_test\n")
	return errors.Trace(ioutil.WriteFile(m.storesMetaPath+"/storesmeta_test.go", []byte(output.String()), 0600))
}

// New returns a new instance of Meta.
func New() (*Meta, error) {
	storesMetaPath, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
	storesMetaPath += "/storesmeta"
	m := &Meta{
		storesMetaPath: storesMetaPath,
	}
	return m, nil
}
