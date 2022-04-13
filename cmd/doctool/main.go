package main

import (
	"flag"
	"fmt"
	"reflect"
	"strings"

	"github.com/grafana/loki/pkg/loki"
)

type ConfigTree struct {
	Name    string
	Tag     []string
	Type    reflect.Type
	Fields  []ConfigTree
	Pointer uintptr
	Flag    *flag.Flag
}

func indent(i int) string {
	sb := strings.Builder{}
	for x := 0; x < i; x++ {
		sb.WriteString("  ")
	}
	return sb.String()
}

func getType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return "int"
	case reflect.Float32,
		reflect.Float64:
		return "float"
	case reflect.Slice:
		return "list[" + getType(t.Elem()) + "]"
	case reflect.Struct:
		return t.Name()
	case reflect.Ptr:
		return "*" + getType(t.Elem())
	default:
		return t.Kind().String()
	}
}

func parseTag(tag string) (string, []string) {
	parts := strings.Split(tag, ",")
	switch len(parts) {
	case 0:
		return "", []string{}
	case 1:
		return parts[0], []string{}
	default:
		return parts[0], parts[1:]
	}
}

func ParseTree(t ConfigTree, v reflect.Value) ConfigTree {
	fields := reflect.VisibleFields(v.Type())
	for _, field := range fields {
		yamlTag := field.Tag.Get("yaml")
		name, tags := parseTag(yamlTag)
		if name == "" || name == "-" {
			continue
		}
		fieldValue := v.FieldByIndex(field.Index)
		leave := ConfigTree{
			Name: name,
			Tag:  tags,
			Type: field.Type,
		}
		switch field.Type.Kind() {
		case reflect.Struct:
			t.Fields = append(t.Fields, ParseTree(leave, fieldValue))
		default:
			leave.Pointer = fieldValue.Addr().Pointer()
			t.Fields = append(t.Fields, leave)
		}
	}
	return t
}

func PrintConfigTree(t ConfigTree, i int) string {
	sb := strings.Builder{}
	sb.WriteString(t.Name)
	if t.Type != nil && len(t.Fields) == 0 {
		sb.WriteString(": ")
		sb.WriteString(getType(t.Type))
		if t.Flag != nil {
			sb.WriteString(" -")
			sb.WriteString(t.Flag.Name)
			sb.WriteString("=")
			sb.WriteString(t.Flag.DefValue)
		}
	}
	if len(t.Fields) > 0 {
		sb.WriteString(" {\n")
		for _, field := range t.Fields {
			sb.WriteString(indent(i+1) + PrintConfigTree(field, i+1) + "\n")
		}
		sb.WriteString(indent(i) + "}")
	}
	return sb.String()
}

func WalkConfigTree(t ConfigTree, fn func(f ConfigTree) ConfigTree) ConfigTree {
	if len(t.Fields) > 0 {
		for i, field := range t.Fields {
			t.Fields[i] = WalkConfigTree(field, fn)
		}
	}
	return fn(t)
}

func Root() ConfigTree {
	return ConfigTree{Name: "root"}
}

func parseFlags(fs *flag.FlagSet) map[uintptr]*flag.Flag {
	m := make(map[uintptr]*flag.Flag)
	fs.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "deprecated" {
			return
		}
		val := reflect.ValueOf(f.Value)
		m[val.Pointer()] = f
	})
	return m
}

func main() {
	root := &loki.Config{}
	v := reflect.ValueOf(root)
	tree := ParseTree(Root(), v.Elem())

	fs := flag.NewFlagSet("docs", flag.PanicOnError)
	root.RegisterFlags(fs)
	flagByPtr := parseFlags(fs)

	tree = WalkConfigTree(tree, func(t ConfigTree) ConfigTree {
		t.Flag = flagByPtr[t.Pointer]
		return t
	})

	fmt.Println(PrintConfigTree(tree, 0))
}
