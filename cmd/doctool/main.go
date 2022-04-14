package main

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
)

type Node struct {
	Name     string
	Desc     string
	Type     reflect.Type
	Tag      []string
	Children []Node
	Pointer  uintptr
	Flag     *flag.Flag
	IsRoot   bool
}

type Apply func(tree Node) Node

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

func ParseTree(t Node, v reflect.Value) Node {
	fields := reflect.VisibleFields(v.Type())
	for _, field := range fields {
		yamlTag := field.Tag.Get("yaml")
		name, tags := parseTag(yamlTag)
		if name == "" || name == "-" {
			continue
		}
		fieldValue := v.FieldByIndex(field.Index)
		node := Node{
			Name: name,
			Tag:  tags,
			Type: field.Type,
		}
		switch field.Type.Kind() {
		case reflect.Struct:
			t.Children = append(t.Children, ParseTree(node, fieldValue))
		default:
			node.Pointer = fieldValue.Addr().Pointer()
			t.Children = append(t.Children, node)
		}
	}
	return t
}

func PrintConfigTree(t Node, i int) string {
	sb := strings.Builder{}
	sb.WriteString(t.Name)
	sb.WriteString(fmt.Sprintf(" (%v)", t.IsRoot))
	if t.Type != nil && len(t.Children) == 0 {
		sb.WriteString(": ")
		sb.WriteString(getType(t.Type))
		if t.Flag != nil {
			sb.WriteString(" -")
			sb.WriteString(t.Flag.Name)
			sb.WriteString("=")
			sb.WriteString(t.Flag.DefValue)
		}
	}
	if len(t.Children) > 0 {
		sb.WriteString(" {\n")
		for _, field := range t.Children {
			sb.WriteString(indent(i+1) + PrintConfigTree(field, i+1) + "\n")
		}
		sb.WriteString(indent(i) + "}")
	}
	return sb.String()
}

func WalkConfigTree(tree Node, fn Apply) Node {
	if len(tree.Children) > 0 {
		for i := range tree.Children {
			tree.Children[i] = WalkConfigTree(tree.Children[i], fn)
		}
	}
	return fn(tree)
}

func Tree() Node {
	return Node{Name: "root"}
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

func ApplyRootBlocks(tree Node, blocks []Block) Node {
	return WalkConfigTree(tree, func(node Node) Node {
		for _, block := range blocks {
			t := reflect.TypeOf(block.Type)
			if node.Type == t {
				node.IsRoot = true
				node.Desc = block.Desc
				fmt.Println("block:", node.Name, block.Desc, getType(t))
			}
		}
		return node
	})
}

func main() {
	root := Config()
	v := reflect.ValueOf(root)
	tree := ParseTree(Tree(), v.Elem())

	fs := flag.NewFlagSet("docs", flag.PanicOnError)
	root.RegisterFlags(fs)
	flagByPtr := parseFlags(fs)

	tree = WalkConfigTree(tree, func(t Node) Node {
		t.Flag = flagByPtr[t.Pointer]
		return t
	})

	fmt.Println(PrintConfigTree(tree, 0))

	tree = ApplyRootBlocks(tree, Blocks())
}
