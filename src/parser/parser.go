package parser

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"fmt"
	"path/filepath"
	"strings"
	"regexp"

	"../util"
)

const (
	NO_PACKAGE_ERR = "Couldn't find package of source file %s\n"
	NO_FILENAME_ERR = "Couldn't find filename of source file %s\n"
)

type Set map[string]struct{}

func (s Set) MarshalJSON() ([]byte, error) {
	a := make([]string, 0)
	for k, _ := range s {
		a = append(a, k)
	}
	return json.Marshal(a)
}

type Type struct {
	Name			string
	Type			string
	PrivateVars		map[string]string		`json:"PrivateVars,omitempty"`
	PublicVars		map[string]string		`json:"PublicVars,omitempty"`
	PrivateFuncs	Set						`json:"PrivateFuncs,omitempty"`
	PublicFuncs		Set						`json:"PublicFuncs,omitempty"`
	Relationships	Set						`json:"Relationships,omitempty"`
}

type File struct {
	Name		string
	PkgName		string
	Source		string						`json:"-"`
	Imports		map[string]string
	Types		map[string]Type
	Package		*Package					`json:"-"`
}

type Package struct {
	Name		string
	Files		map[string]File
	TypeSet		map[string]string
}

type Parse struct {
	Sources			map[string]string		`json:"-"`
	Packages		map[string]Package
	TypeMap			map[string]string
}

func Parser(sources []string) (*Parse, error) {
	p := new(Parse)
	p.Sources = make(map[string]string)
	p.Packages = make(map[string]Package)
	p.TypeMap = make(map[string]string)
	for _, source := range sources {
		data, err := ioutil.ReadFile(source)
		if err != nil {
			return nil, err
		}

		p.Sources[source] = util.StripComment(string(data))
	}

	if err := p.GetPackages(); err != nil {
		return nil, err
	}

	p.PackageStructs()

	if util.Debug {
		if err := p.Dump(); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func InitFile() File {
	var f File
	f.Imports = make(map[string]string)
	f.Types = make(map[string]Type)
	return f
}

func InitPackage() Package {
	var pkg Package
	pkg.Files = make(map[string]File)
	pkg.TypeSet = make(map[string]string)
	return pkg
}

func (p *Parse) GetPackages() error {
	for file, source := range p.Sources {
		directory := filepath.Dir(file)
		split := strings.SplitAfterN(directory, "src/", 2)
		if len(split) != 2 {
			return fmt.Errorf(NO_PACKAGE_ERR, file)
		}
		packageName := split[1]

		split = strings.SplitAfterN(file, "src/", 2)
		if len(split) != 2 {
			return fmt.Errorf(NO_FILENAME_ERR, file)
		}
		filename := split[1]

		f := InitFile()
		f.Source = source
		f.PkgName = packageName
		f.Name = filename
		

		pkg, exists := p.Packages[packageName]
		if !exists {
			pkg = InitPackage()
			pkg.Name = packageName
		}

		f.Package = &pkg
		pkg.Files[filename] = f
		p.Packages[packageName] = pkg
	}
	
	for _, pkg := range p.Packages {
		for _, f := range pkg.Files {
			if err := f.GetImports(); err != nil {
				return err
			}
			
			if err := f.GetStructs(); err != nil {
				return err
			}
			
		}		
	}

	for _, pkg := range p.Packages {
		for _, f := range pkg.Files {
			if err := f.GetFunctions(); err != nil {
				return err
			}
		}	
	}
	
	return nil
}

const (
	SINGLE_LINE_IMPORT_REGEX = "import(\\s)*\".*\"\n"
	MULTI_LIINE_IMPORT_REGEX = "import.*?\\((.|\\s)*?\\)"
	IMPORT_REGEX = "\".*?\""
	RENAME_REGEX = "\\S*\\s\".*?\""
)

func (f *File) GetImports() error {
	re := regexp.MustCompile(SINGLE_LINE_IMPORT_REGEX)
	sImports := re.FindAllString(f.Source, -1)

	re = regexp.MustCompile(MULTI_LIINE_IMPORT_REGEX)
	bImports := re.FindAllString(f.Source, -1)


	re = regexp.MustCompile(IMPORT_REGEX)
	for _, line := range sImports {
		ip := re.FindString(line)
		ip = util.ReplaceAll(ip, "\"", "")
		ip = util.ReplaceAll(ip, "../", "")
		f.Imports[filepath.Base(ip)] = ip
	}

	re = regexp.MustCompile(RENAME_REGEX)
	re2 := regexp.MustCompile("\\s")
	for _, lines := range bImports {
		ips := re.FindAllString(lines, -1)
		for _, line := range ips {
			split := re2.Split(line, 2)

			if len(split) != 2 {
				return fmt.Errorf("Bad import parsing: %s", lines)
			}
			ip := util.ReplaceAll(split[1], "\"", "")
			ip = util.ReplaceAll(ip, "../", "")
			if split[0] == "" {
				f.Imports[filepath.Base(ip)] = ip
			} else {
				f.Imports[split[0]] = ip
			}
		}
	}
	return nil
}

const (
	STRUCT_REGEX = "type\\s.*?\\sstruct\\s?{"
	TYPE_REGEX = "\\s?type\\s"
)

func InitType() Type {
	var t Type
	t.PrivateVars = make(map[string]string)
	t.PublicVars = make(map[string]string)
	t.PrivateFuncs = make(Set)
	t.PublicFuncs = make(Set)
	t.Relationships = make(Set)
	return t
}

func (f *File) GetStructs() error {
	source := util.ReplaceAll(f.Source, "\n\n", "\n")

	lines := strings.Split(source, "\n")

	re := regexp.MustCompile(TYPE_REGEX)
	re2 := regexp.MustCompile("\\s+")

	for i, line := range lines {
		if !re.Match([]byte(line)) {
			continue
		}
		
		nLine := re.ReplaceAllString(line, "")
		split := re2.Split(nLine, 2)
		if len(split) != 2 {
			return fmt.Errorf("Type not formatted correctly. Format: \"type NAME DEFINITION\"): %s\n", nLine)
		} 

		t := InitType()
		t.Name = split[0]
		f.Package.TypeSet[t.Name] = t.Name
		
		if strings.HasPrefix(split[1], "struct") {
			t.Type = "struct"

			if i+1 >= len(lines) {
				return errors.New("Struct definition has no content")
			}
			for _, field := range lines[i+1:] {
				if field == "}" {
					break
				}
				definition := re2.Split(field, 3)
				if len(definition) < 2 {
					return fmt.Errorf("Definition not formatted correctly. Format: \"NAME DEFINITION <optional tag>\"): %s\n", field)
				}
				if definition[0] == "" {
					definition = definition[1:]
				}
				name := re2.ReplaceAllString(definition[0], "")
				typ := re2.ReplaceAllString(definition[1], "")
				if len(name) < 1 {
					return errors.New("Field name empty")
				}
				if strings.ToUpper(string(name[0])) == string(name[0]) {
					t.PublicVars[name] = typ
				} else {
					t.PrivateVars[name] = typ
				}
			}
			
		} else {
			t.Type = split[1]
		}
		f.Types[t.Name] = t
	}
	return nil
}

const (
	FUNC_REGEX = "func\\s.*?{\n"
	FUNC_END_REGEX = "\\s{"
	FUNC_STRUCT_REGEX = "func\\s\\(.*?\\)\\s"
	FUNC_START_REGEX = "func\\s"
	STRUCT_REGEX2 = "(.*?\\s\\*?(.*?))\\s"
)

func (f *File) GetFunctions() error {
	source := util.ReplaceAll(f.Source, "\n\n", "\n")
	lines := strings.Split(source, "\n")

	re := regexp.MustCompile(FUNC_REGEX)
	re2 := regexp.MustCompile(FUNC_END_REGEX)
	re3 := regexp.MustCompile(FUNC_STRUCT_REGEX)
	re4 := regexp.MustCompile(FUNC_START_REGEX)
	re5 := regexp.MustCompile(STRUCT_REGEX2)

	global := make([]string, 0)
	functionLines := make(map[string][]string)
	
	// Get struct functions
	for i := 0; i < len(lines); {
		line := lines[i]
		if !re.Match([]byte(line + "\n")) {
			global = append(global, line)
			i++
			continue
		}
		funcStr := re2.ReplaceAllString(line, "")

		if !re3.Match([]byte(funcStr)) {
			global = append(global, line)
			i++
			continue
		}

		noFuncStr := re4.ReplaceAllString(funcStr, "")
		matches := re5.FindStringSubmatch(noFuncStr)
		if len(matches) < 3 {
			return fmt.Errorf("Failed to get struct for function: %s", line)
		}
		typ := matches[2][:len(matches[2])-1]
		if _, exists := f.Types[typ]; !exists {
			newType := InitType()
			newType.Name = typ
			f.Types[typ] = newType
		}
		
		funcDef := re3.ReplaceAllString(funcStr, "")
		if len(funcDef) < 1 {
			return fmt.Errorf("No function name: %s")
		}

		if strings.ToUpper(string(funcDef[0])) == string(funcDef[0]) {
			f.Types[typ].PublicFuncs[funcDef] = struct{}{}
		} else {
			f.Types[typ].PrivateFuncs[funcDef] = struct{}{}
		}

		j := i
		for _, cnt := range lines[j:] {
			i++
			if cnt == "}" {
				break
			}
			functionLines[typ] = append(functionLines[typ], cnt)
		}
	}

	if err := f.Global(global); err != nil {
		return err
	}

	for typ, lines := range functionLines {
		for _, line := range lines {
			uses := f.Uses(line)
			for _, use := range uses {
				f.Types[typ].Relationships[use] = struct{}{}
			}
		}
	}
	return nil
}

const (
	CONST_VAR_MULTI_REGEX = "(var|const)\\s?\\("
	CONST_VAR_SINGLE_REGEX = "(var|const)\\s?.*?"
	CONST_VAR_REGEX = "(var|const)\\s?"
	VAR_STRUCT_REGEX = "var\\s+.*?struct"
)

func (f *File) Global(lines []string) error {
	typ := strings.Title(f.PkgName) + "Global" 
	re := regexp.MustCompile(FUNC_REGEX)
	re2 := regexp.MustCompile(FUNC_END_REGEX)
	re4 := regexp.MustCompile(FUNC_START_REGEX)

	if _, exists := f.Types[typ]; !exists {
		t := InitType()
		t.Name = typ
		t.Type = "global"
		f.Types[typ] = t
	}

	variables := make([]string, 0)
	functionLines := make([]string, 0)
	for i := 0; i < len(lines); {
		line := lines[i]

		if !re.Match([]byte(line + "\n")) {
			variables = append(variables, line)
			i++
			continue
		}

		funcStr := re2.ReplaceAllString(line, "")
		noFuncStr := re4.ReplaceAllString(funcStr, "")

		if strings.ToUpper(string(noFuncStr[0])) == string(noFuncStr[0]) {
			f.Types[typ].PublicFuncs[noFuncStr] = struct{}{}
		} else {
			f.Types[typ].PrivateFuncs[noFuncStr] = struct{}{}
		}

		j := i
		for _, cnt := range lines[j:] {
			i++
			if cnt == "}" {
				break
			}
			functionLines = append(functionLines, cnt)	
		}
	}

	noMulti := make([]string, 0)
	re = regexp.MustCompile(CONST_VAR_MULTI_REGEX)
	re2 = regexp.MustCompile("\\s")
	for i := 0; i < len(variables); {

		line := variables[i]
		if !re.Match([]byte(line + "\n")) {
			noMulti = append(noMulti, line)
			i++
			continue
		}
		j := i + 1
		if j >= len(variables) {
			return fmt.Errorf("Couldn't parse const in %s", f.PkgName)
		}
		for _, cnst := range variables[j:] {
			i++
			if cnst == ")" {
				break
			}
			split := re2.Split(cnst, 4)
			if len(split) < 2 {
				return fmt.Errorf("Failed to get const variable: %s", line)
			}
			constant := split[0]
			if constant == "" {
				constant = split[1]
			}
			cType := split[2]
			if cType == "=" {
				cType = ""
			}

			f.Package.TypeSet[constant] = typ
			if strings.ToUpper(string(constant[0])) == string(constant[0]) {
				f.Types[typ].PublicVars[constant] = cType
			} else {
				f.Types[typ].PrivateVars[constant] = cType
			}
		}
	}

	
	re = regexp.MustCompile(CONST_VAR_SINGLE_REGEX)
	re2 = regexp.MustCompile(CONST_VAR_REGEX)
	re4 = regexp.MustCompile(VAR_STRUCT_REGEX)

	for _, line := range noMulti {
		if !re.Match([]byte(line)) {
			continue
		}

		variables := make(map[string]string)
		nLine := re2.ReplaceAllString(line, "")
		if strings.Contains(nLine, ",") {
			
			re3 := regexp.MustCompile("(\\s|,\\s?)")
			replaced := re3.ReplaceAllString(nLine, ",")
			split := strings.Split(replaced, ",")

			if split == nil {
				return fmt.Errorf("Failed to parse variable: %s", line)
			}

			nSplit := make([]string, 0)
			re3 = regexp.MustCompile("\\s+")
			for _, s := range split {
				if re3.Match([]byte(s)) || s == "" {
					continue
				}
				nSplit = append(nSplit, s)
			}


			if strings.Contains(nLine, "=") {
				var t string
				vars := make([]string, 0)
				for _, variable := range nSplit {
					if re3.Match([]byte(variable)) {
						split := re3.Split(variable, 2)
						vars = append(vars, split[0])
						t = split[1]
						break
					} else if variable == "=" {
						t = ""
						break
					} else {
						vars = append(vars, variable)
					}
				}

				for _, v := range vars {
					variables[v] = t
				}
			} else {
				for i, variable := range nSplit {
					if i >= len(nSplit) - 1 {
						break
					}
					variables[variable] = split[len(nSplit)-1]
				}
			}
			
		} /*else {
			re5 := regexp.MustCompile("\\s")
			split := re5.Split(nLine, 2)
			if len(split) < 2 {
				return fmt.Errorf("Failed to parse variable: %s", line)
			}

			variables[split[0]] = ""
		} 
		*/
		/*else if strings.Contains(nLine, "=") {
			//fmt.Println("=:", nLine)
			re3 := regexp.MustCompile("\\s?=\\s?")
			split := re3.Split(nLine, 3)
			if len(split) < 2 {
				return fmt.Errorf("Failed to parse variable: %s", line)
			}
			cType := split[1]
			if split[1] == "=" {
				cType = ""
			}
			variables[split[0]] = cType
		} else if re4.Match([]byte(line)) {
			//fmt.Println("re4:", line)
			re3 := regexp.MustCompile("\\s+")
			split := re3.Split(nLine, 2)
			if len(split) < 2 {
				return fmt.Errorf("Failed to parse variable: %s", line)
			}
			variables[split[0]] = "struct{}"
		} else {
			//fmt.Println("else:", line)
		}*/
		//fmt.Println(variables)

		for k, v := range variables {
			if len(k) < 1 {
				return errors.New("Variable name empty. Bad parsing.")
			}
			f.Package.TypeSet[k] = typ
			if strings.ToUpper(string(k[0])) == string(k[0]) {
				f.Types[typ].PublicVars[k] = v
			} else {
				f.Types[typ].PrivateVars[k] = v
			}
		}
	}

	for _, line := range functionLines {
		uses := f.Uses(line)
		for _, use := range uses {
			f.Types[typ].Relationships[use] = struct{}{}
		}
	}
	return nil
}

func (f *File) Uses(line string) []string {
	uses := make([]string, 0)
	for k, v := range f.Imports {
		re := regexp.MustCompile("(" + k + "\\..*?)(\\s|,|\\(|\\)|{|}|\\.)")
		matches := re.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		uses = append(uses, strings.Replace(matches[1], k, filepath.Base(v), 1))
	}

	for k, v := range f.Package.TypeSet {
		re := regexp.MustCompile("(\\s|\\(|,)" + k + "(\\s|\\(|,|)")
		if re.Match([]byte(line)) && v != f.Name {
			uses = append(uses, f.Package.Name + "." + v)
		}
	}
	return uses
}


func (p *Parse) PackageStructs() {
	for name, pkg := range p.Packages {
		for _, f := range pkg.Files {
			for typ, t := range f.Types {
				p.TypeMap[filepath.Base(name) + "." + typ] = typ
				if t.Type == "global" {
					
				}
			}
		}
	}
}

func (p *Parse) Dump() error {
	data, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}