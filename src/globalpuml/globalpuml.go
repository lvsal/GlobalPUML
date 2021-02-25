package main

import (
	"os"
	"path/filepath"
	"fmt"

	"errors"
	"../parser"
	"../puml"
	"../util"
)

func eoe(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error() + "\n")
		os.Exit(1)
	}
}

func main() {
	if len(os.Args) < 2 {
		eoe(errors.New("Usage: globalpuml (root source directory) [-d | -g]"))
	}

	if len(os.Args) == 3 {
		switch os.Args[2] {
		case "-d":
			util.Debug = true
			util.Global = true
		case "-g":
			util.Global = true
		}
	}

	sources := make([]string, 0)
	err := filepath.Walk(os.Args[1],
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".go" {
				sources = append(sources, path)
			}
			return nil
		})
	eoe(err)
	
	p, err := parser.Parser(sources)
	eoe(err)
	eoe(puml.GeneratePUML(p))
}