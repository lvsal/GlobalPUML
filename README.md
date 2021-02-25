# GlobalPUML

This is a PUML generator for Go, and is being used for a class project. This will most likely not be maintained. This treats package global as an object to allow an object oriented representation of Go code. For example, if you have a package named "mypackage" that contains non-struct variables and functions, these will be placed into a "MypackageGlobal" object.

Namespace idea and function types were taken from https://github.com/jfeliu007/goplantuml.

Usage
-------
- Run `go build` in `src/globalpuml`
- Run `./globalpuml <directory> [-d | -g]`. The `-g` arg is for including relationships between <package>Global object and structures within the same package. I included this as an option as it's implied that package global functions/variables use package structs and vice-versa. It keeps the UML diagram clean. The `-d` arg is for debugging. It will dump the JSON data collected and relationships.

Caveats
-------
- All source code needs to be under 1 directory. Nested directories are fine.
- Go code should be properly formatted using `gofmt -w <.go file>`. The behaviour is unknown if this is not the case.
- Constants and variables with no explicit type declaration (eg. `const T = "a string"`) will not have a type. These will need to be put in manually. This also means that constants and variables using a type/struct would need to have their relationship manually put in.
- There are only 2 relationships that are being used. It's either an association when two objects are using each other, and dependency relationship when it's one-way.
