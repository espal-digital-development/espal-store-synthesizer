package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/espal-digital-development/espal-store-synthesizer/meta"
	"github.com/espal-digital-development/espal-store-synthesizer/packages"
	"github.com/espal-digital-development/system/permissions"
	"github.com/juju/errors"
	"github.com/mattn/go-zglob"
)

func main() {
	storesPath, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
	if !strings.HasSuffix(storesPath, "/stores") {
		storesPath += "/stores"
	}

	packages, err := collectPackages(storesPath)
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
	for _, pkg := range packages {
		if err := buildOutputForPackage(pkg); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
	}
	meta, err := meta.New()
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
	}
	if err := meta.Build(packages); err != nil {
		log.Fatal(errors.ErrorStack(err))
	}

	if out, err := exec.Command("go", "fmt", storesPath+"/...").Output(); err != nil {
		fmt.Println(string(out))
		log.Fatal(errors.ErrorStack(err))
	}
}

func collectPackages(path string) ([]*packages.Package, error) {
	pkgs := []*packages.Package{}
	entries, err := zglob.Glob(path + "/**/*")
	if err != nil {
		return nil, errors.Trace(err)
	}
	for _, entry := range entries {
		stat, err := os.Stat(entry)
		// Need to skip non-existing files because the old files get
		// deleted whilst they were still in the initial glob scan results.
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
		if !stat.IsDir() {
			continue
		}

		// TODO :: This works, but can use some more strictness
		if strings.HasSuffix(entry, "mock") {
			continue
		}

		pkg := packages.New()
		if err := pkg.BuildMetaData(entry); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
}

func buildOutputForPackage(pkg *packages.Package) error {
	entityData, err := pkg.MainEntity().BuildFileOutput()
	if err != nil {
		return errors.Trace(err)
	}
	if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(pkg.MainEntity().Name())+
		"_synthesized.go", entityData, permissions.UserReadWrite); err != nil {
		return errors.Trace(err)
	}

	entityTestData, err := pkg.MainEntity().BuildTestFileOutput()
	if err != nil {
		return errors.Trace(err)
	}
	if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(pkg.MainEntity().Name())+
		"_synthesized_test.go", entityTestData, permissions.UserReadWrite); err != nil {
		return errors.Trace(err)
	}

	for _, entity := range pkg.Entities() {
		if entity.IsPrimaryEntity() {
			return errors.Errorf("expected a non-primary entity for `%s` at `%s`", entity.Name(), pkg.Path())
		}
		entityData, err := entity.BuildFileOutput()
		if err != nil {
			return errors.Trace(err)
		}
		if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(entity.Name())+
			"_synthesized.go", entityData, permissions.UserReadWrite); err != nil {
			return errors.Trace(err)
		}

		entityTestData, err := entity.BuildTestFileOutput()
		if err != nil {
			return errors.Trace(err)
		}
		if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(entity.Name())+
			"_synthesized_test.go", entityTestData, permissions.UserReadWrite); err != nil {
			return errors.Trace(err)
		}
	}

	storeData, err := pkg.Store().BuildFileOutput()
	if err != nil {
		return errors.Trace(err)
	}
	if err := ioutil.WriteFile(pkg.Path()+"/store_synthesized.go", storeData,
		permissions.UserReadWrite); err != nil {
		return errors.Trace(err)
	}

	// TODO :: 777777 Build this too
	// var storeTestFile []byte
	return nil
}
