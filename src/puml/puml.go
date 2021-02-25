package puml

import (
	"../parser"
	"fmt"
	"strings"
	"regexp"

	"../util"
)

var Relationships map[string]parser.Set = make(map[string]parser.Set)

type Class struct {
	Name			string
	Type			string
	PrivateVars		map[string]string
	PublicVars		map[string]string
	PrivateFuncs	parser.Set
	PublicFuncs		parser.Set
	Relationships	parser.Set
}

type Namespace struct {
	Name			string
	Classes			map[string]Class
}

type PlantUML struct {
	Namespaces		map[string]Namespace
}

func InitClass() Class {
	var c Class
	c.PrivateVars = make(map[string]string)
	c.PublicVars = make(map[string]string)
	c.PrivateFuncs = make(parser.Set)
	c.PublicFuncs = make(parser.Set)
	c.Relationships = make(parser.Set)
	return c
}

func InitNamespace() Namespace {
	var ns Namespace
	ns.Classes = make(map[string]Class)
	return ns
}


func InitPlantUML() PlantUML {
	var uml PlantUML
	uml.Namespaces = make(map[string]Namespace)
	return uml
}

func RelationshipsSet() parser.Set {
	s := make(parser.Set)
	for class, rs := range Relationships {
		for r, _ := range rs {
			found := false
			var arrow string
			if k, exists := Relationships[r]; exists {
				if _, exists2 := k[class]; exists2 {
					delete(k, class)
					arrow = " --- "
					found = true
				}
			}
			if !found {
				arrow = " --> "
				
			}
			s[class + arrow + r] = struct{}{}
		}
	}

	return s
}

func parseToPUML(parse *parser.Parse) (*PlantUML, error) {
	uml := InitPlantUML()
	for _, pkg := range parse.Packages {
		ns := InitNamespace()
		ns.Name = pkg.Name

		for _, f := range pkg.Files {
			for _, t := range f.Types {
				c, exists := ns.Classes[t.Name]
				if !exists {
					c = InitClass()
					c.Name = t.Name
					c.Type = t.Type
				} 

				for k, v := range t.PrivateVars {
					c.PrivateVars[k] = v
				}

				for k, v := range t.PublicVars {
					c.PublicVars[k] = v
				}

				for k, _ := range t.PrivateFuncs {
					c.PrivateFuncs[k] = struct{}{}
				}

				for k, _ := range t.PublicFuncs {
					c.PublicFuncs[k] = struct{}{}
				}

				for k, _ := range t.Relationships {
					split := strings.Split(k, ".")
					if len(split) != 2 {
						return nil, fmt.Errorf("Failed to get package for %s", k)
					}

					if !util.Global {
						if pkg.Name == split[0] && (strings.Contains(k, "Global") || strings.Contains(c.Name, "Global")) {
							continue
						}
					}
					
					_, exists2 := parse.TypeMap[k]
					if exists2 {
						if k != pkg.Name + "." + t.Name {
							c.Relationships[k] = struct{}{}
						}
					} else {						
						if _, exists := parse.Packages[split[0]]; exists {
							if util.Global && split[0] != pkg.Name {
								c.Relationships[split[0] + "." + strings.Title(split[0]) + "Global"] = struct{}{}
							}
						}
					}
				}
				ns.Classes[t.Name] = c
			}
		}
		uml.Namespaces[ns.Name] = ns
	}
	return &uml, nil
}

const (
	GLOBAL = "<< (G,Green) >>"
	STRUCT = "<< (S,Aquamarine) >>"
	TYPE = "<< (T, #FF7700) >>"
	COLOR_REPLACE = "<font color=%s>%s</font>"
	FUNC = "\"%s \" as %s.%s"
)

func colorText(text, replace, color string) string {
	colored := fmt.Sprintf(COLOR_REPLACE, color, replace)
	n := strings.Count(text, replace)
	return strings.Replace(text, replace, colored, n)
}

func blueType(text string) string {
	t := text
	if strings.Contains(t, "map") {
		t = colorText(t, "map", "blue")
	}

	if strings.Contains(t, "struct") {
		t = colorText(t, "struct", "blue")
	}

	if strings.Contains(t, "chan") {
		t = colorText(t, "chan", "blue")
	}

	if strings.Contains(t, "func") {
		t = colorText(t, "func", "blue")
	}
	return t
}

func (c * Class) PUMLString(namespace string) string {
	var puml, symbol string

	switch c.Type {
	case "global":
		symbol = GLOBAL
	case "struct":
		symbol = STRUCT
	default:
		symbol = TYPE
	}

	puml += fmt.Sprintf("class %s.%s %s {\n", namespace, c.Name, symbol)

	for k, v := range c.PrivateVars {
		puml += fmt.Sprintf("\t- %s %s\n", k, v)
	}

	if len(c.PrivateVars) != 0 {
		puml += "\n"
	}
	
	for k, v := range c.PublicVars {
		puml += fmt.Sprintf("\t+ %s %s\n", k, v)
	}

	if len(c.PrivateFuncs) != 0 {
		puml += "\n"
	}

	for k, _ := range c.PrivateFuncs {
		puml += fmt.Sprintf("\t- %s\n", k)
	}

	if len(c.PublicFuncs) != 0 {
		puml += "\n"
	}

	for k, _ := range c.PublicFuncs {
		puml += fmt.Sprintf("\t+ %s\n", k)
	}

	puml += "}\n"

	class := namespace + "." + c.Name
	ps, exists := Relationships[class]
	if !exists {
		ps = make(parser.Set)
	}

	for k, _ := range c.Relationships {
		ps[k] = struct{}{}
	}

	Relationships[class] = ps

	if !strings.HasPrefix(c.Type, "func") {
		return puml
	}

	blueT := blueType(c.Type)
	re := regexp.MustCompile("[^a-zA-Z\\d\\s:]")
	clean := re.ReplaceAllString(c.Type, "")
	n := strings.Count(clean, " ")
	clean = strings.Replace(clean, " ", "", n)

	fullText := fmt.Sprintf(FUNC, blueT, namespace, clean)
	puml += fmt.Sprintf("class %s {\n}\n", fullText)
	puml += fmt.Sprintf("\"%s.%s\" #.. \"%s\"\n", namespace, clean, c.Name)

	return puml
}

func (ns *Namespace) PUMLString() string {
	var puml string
	puml += fmt.Sprintf("namespace %s {\n\t", ns.Name)

	for _, c := range ns.Classes {
		cStr := c.PUMLString(ns.Name)
		n := strings.Count(cStr, "\n")
		cStr = strings.Replace(cStr, "\n", "\n\t", n)
		puml += cStr
	}

	puml += "\n}\n"
	return puml
}

func GeneratePUML(parse *parser.Parse) error {
	uml, err := parseToPUML(parse)
	if err != nil {
		return err
	}

	var puml string
	puml += "@startuml\n"
	i := 0
	for _, ns := range uml.Namespaces {
		puml += ns.PUMLString()
		if i < len(uml.Namespaces) - 1 {
			puml += "\n"
		}
		i++
	}

	s := RelationshipsSet()
	for r, _ := range s {
		puml += r + "\n"
	}
	puml += "@enduml"

	if !util.Debug {
		fmt.Println(puml)
	} else {
		data, err := util.Dump(s)
		if err != nil {
			return err
		}
		fmt.Println("Relationships:", string(data))
	}
	return nil
}