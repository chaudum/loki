package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Node is the datastructure of the parsed configuration
type Node struct {
	Name string
	Desc string
	Type reflect.Type

	Tag      []string
	Children []Node

	Pointer uintptr
	Flag    *flag.Flag
}

type ApplyFunc func(tree Node) Node
type TransformFunc func(tree Node) *ConfigBlock

// ConfigBlock is the datastructure of the analised configuration
type ConfigBlock struct {
	Name  string      `yaml:"name"`
	Desc  string      `yaml:"description"`
	Type  string      `yaml:"type"`
	Value interface{} `yaml:"value"`

	FlagName   string `yaml:"flag"`
	FlagPrefix string

	Fields []*ConfigBlock `yaml:"fields"`

	IsRoot bool `yaml:"root"`
}

func indent(i int) string {
	sb := strings.Builder{}
	for x := 0; x < i; x++ {
		sb.WriteString("  ")
	}
	return sb.String()
}

func getType(t reflect.Type) string {
	if t == nil {
		return "-"
	}
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
		return t.PkgPath() + "." + t.Name()
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

func WalkConfigTree(tree Node, fn ApplyFunc) Node {
	if len(tree.Children) > 0 {
		for i := range tree.Children {
			tree.Children[i] = WalkConfigTree(tree.Children[i], fn)
		}
	}
	return fn(tree)
}

func AnalyzeConfigTree(tree Node, fn TransformFunc) *ConfigBlock {
	b := fn(tree)
	for i := range tree.Children {
		child := AnalyzeConfigTree(tree.Children[i], fn)
		if child != nil {
			b.Fields = append(b.Fields, child)
		}
	}
	return b
}

func Tree(cfg interface{}) Node {
	return Node{Name: "root", Desc: "", Type: reflect.TypeOf(cfg)}
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

func blockForNode(n Node, blocks []Block) (*Block, bool) {
	for _, bb := range blocks {
		if n.Type == bb.Type {
			return &bb, true
		}
	}
	return nil, false
}

// Apply analyzes the parsed config
func Apply(tree Node, blocks []Block, flagMap map[uintptr]*flag.Flag) *ConfigBlock {
	return AnalyzeConfigTree(tree, func(node Node) *ConfigBlock {
		b := &ConfigBlock{
			Name: node.Name,
			Desc: node.Desc,
			Type: getType(node.Type),
		}
		rootBlock, ok := blockForNode(node, blocks)
		if ok {
			b.IsRoot = true
			b.FlagPrefix = append(b.FlagPrefix, getFlagPrefix())
			b.Desc = rootBlock.Desc
		}
		if flag, ok := flagMap[node.Pointer]; ok {
			b.FlagName = flag.Name
			b.Desc = flag.Usage
		}
		return b
	})
}

func main() {
	root := Config()
	v := reflect.ValueOf(root)
	tree := ParseTree(Tree(root), v.Elem())

	fs := flag.NewFlagSet("docs", flag.PanicOnError)
	root.RegisterFlags(fs)
	fmt.Println(PrintConfigTree(tree, 0))

	out := Apply(tree, Blocks(), parseFlags(fs))
	yaml.NewEncoder(os.Stderr).Encode(out)
}
