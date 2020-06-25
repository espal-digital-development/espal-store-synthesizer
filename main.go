package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/espal-digital-development/espal-store-synthesizer/packages"
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

	// First collect all the information from the stores' files
	pkgs := []*packages.Package{}
	entries, err := zglob.Glob(storesPath + "/**/*")
	if err != nil {
		log.Fatal(errors.ErrorStack(err))
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

	// Build the output
	for _, pkg := range pkgs {
		entityData, err := pkg.MainEntity().BuildFileOutput()
		if err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
		if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(pkg.MainEntity().Name())+
			"_synthesized.go", entityData, 0644); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}

		entityTestData, err := pkg.MainEntity().BuildTestFileOutput()
		if err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
		if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(pkg.MainEntity().Name())+
			"_synthesized_test.go", entityTestData, 0644); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}

		for _, entity := range pkg.Entities() {
			if entity.IsPrimaryEntity() {
				log.Fatal(errors.Errorf("expected a non-primary entity for `%s` at `%s`", entity.Name(), pkg.Path()))
			}
			entityData, err := entity.BuildFileOutput()
			if err != nil {
				log.Fatal(errors.ErrorStack(err))
			}
			if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(entity.Name())+
				"_synthesized.go", entityData, 0644); err != nil {
				log.Fatal(errors.ErrorStack(err))
			}

			entityTestData, err := entity.BuildTestFileOutput()
			if err != nil {
				log.Fatal(errors.ErrorStack(err))
			}
			if err := ioutil.WriteFile(pkg.Path()+"/"+strings.ToLower(entity.Name())+
				"_synthesized_test.go", entityTestData, 0644); err != nil {
				log.Fatal(errors.ErrorStack(err))
			}
		}

		storeData, err := pkg.Store().BuildFileOutput()
		if err != nil {
			log.Fatal(errors.ErrorStack(err))
		}
		if err := ioutil.WriteFile(pkg.Path()+"/store_synthesized.go", storeData, 0644); err != nil {
			log.Fatal(errors.ErrorStack(err))
		}

		// TODO :: 777777 Build this too
		// var storeTestFile []byte
	}

	// Format all files one more times to get clean and valid inspector checks
	if out, err := exec.Command("go", "fmt", storesPath+"/...").Output(); err != nil {
		fmt.Println(string(out))
		log.Fatal(errors.ErrorStack(err))
	}
}
