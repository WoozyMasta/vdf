// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"encoding"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()

// zeroer is implemented by types that can report their own zero state.
// Marshal respects this when the omitzero tag option is set.
type zeroer interface {
	IsZero() bool
}

// vdfTag holds parsed struct tag options for a single field.
type vdfTag struct {
	name      string // VDF key name for this field
	omitempty bool   // omit if reflect.Value.IsZero()
	omitzero  bool   // omit if IsZero() bool method, else reflect.Value.IsZero()
	inline    bool   // hoist child struct fields into parent object
	repeated  bool   // []T -> multiple sibling nodes with the same key
	indexed   bool   // []T -> object with keys "0", "1", ...
}

// shouldOmit reports whether fv should be omitted during marshal.
func shouldOmit(fv reflect.Value, tag vdfTag) bool {
	switch {
	case tag.omitempty:
		return fv.IsZero()

	case tag.omitzero:
		if fv.CanInterface() {
			if z, ok := fv.Interface().(zeroer); ok {
				return z.IsZero()
			}
		}
		if fv.CanAddr() {
			if z, ok := fv.Addr().Interface().(zeroer); ok {
				return z.IsZero()
			}
		}
		return fv.IsZero()
	}

	return false
}

// fieldInfo maps a struct field index to its parsed VDF tag.
type fieldInfo struct {
	tag   vdfTag
	index int
}

// fieldCache caches field metadata per reflect.Type to avoid repeated reflection.
var fieldCache sync.Map // map[reflect.Type][]fieldInfo

// getFields returns cached field descriptors for a struct type.
func getFields(t reflect.Type) []fieldInfo {
	if cached, ok := fieldCache.Load(t); ok {
		return cached.([]fieldInfo)
	}

	fields := make([]fieldInfo, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		raw, ok := f.Tag.Lookup("vdf")
		if ok && raw == "-" {
			continue
		}

		tag := parseVDFTag(f)
		fields = append(fields, fieldInfo{index: i, tag: tag})
	}

	fieldCache.Store(t, fields)
	return fields
}

// parseVDFTag parses the vdf struct tag for one field.
func parseVDFTag(f reflect.StructField) vdfTag {
	raw, ok := f.Tag.Lookup("vdf")
	if !ok {
		return vdfTag{name: strings.ToLower(f.Name)}
	}

	parts := strings.Split(raw, ",")
	t := vdfTag{name: parts[0]}
	if t.name == "" {
		t.name = strings.ToLower(f.Name)
	}

	for _, opt := range parts[1:] {
		switch opt {
		case "omitempty":
			t.omitempty = true
		case "omitzero":
			t.omitzero = true
		case "inline":
			t.inline = true
		case "repeated":
			t.repeated = true
		case "indexed":
			t.indexed = true
		}
	}

	return t
}

// Marshal encodes a Go struct into a Document with one root object node.
// v must be a struct or a pointer to a struct.
func Marshal(root string, v any) (*Document, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, fmt.Errorf("%w: nil pointer passed to Marshal", ErrReflectUnsupportedType)
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: Marshal requires a struct, got %s", ErrReflectUnsupportedType, rv.Kind())
	}

	rootNode, err := marshalStructToNode(root, rv)
	if err != nil {
		return nil, err
	}

	doc := NewDocument()
	doc.AddRoot(rootNode)
	return doc, nil
}

// marshalStructToNode converts a struct value into a NodeObject.
func marshalStructToNode(key string, rv reflect.Value) (*Node, error) {
	obj := NewObjectNode(key)
	fields := getFields(rv.Type())

	for _, fi := range fields {
		fv := rv.Field(fi.index)
		ft := rv.Type().Field(fi.index)

		if shouldOmit(fv, fi.tag) {
			continue
		}

		if fi.tag.inline {
			tmp := fv
			skip := false
			for tmp.Kind() == reflect.Pointer {
				if tmp.IsNil() {
					skip = true
					break
				}
				tmp = tmp.Elem()
			}

			if skip {
				continue
			}

			if tmp.Kind() != reflect.Struct {
				return nil, fmt.Errorf("%w: inline field %q must be a struct", ErrReflectUnsupportedType, ft.Name)
			}

			inlineNode, err := marshalStructToNode("__inline__", tmp)
			if err != nil {
				return nil, err
			}

			obj.Children = append(obj.Children, inlineNode.Children...)
			continue
		}

		nodes, err := marshalValue(fi.tag.name, fi.tag, fv)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", ft.Name, err)
		}

		obj.Children = append(obj.Children, nodes...)
	}

	return obj, nil
}

// marshalValue converts a single reflect.Value to one or more VDF nodes.
// Repeated fields return multiple nodes with the same key.
func marshalValue(key string, tag vdfTag, rv reflect.Value) ([]*Node, error) {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	// TextMarshaler via value receiver.
	if rv.CanInterface() && rv.Type().Implements(textMarshalerType) {
		text, err := rv.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, fmt.Errorf("%w: TextMarshaler key %q: %v", ErrReflectUnsupportedType, key, err)
		}
		return []*Node{NewStringNode(key, string(text))}, nil
	}

	// TextMarshaler via pointer receiver.
	if rv.CanAddr() && rv.Addr().Type().Implements(textMarshalerType) {
		text, err := rv.Addr().Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return nil, fmt.Errorf("%w: TextMarshaler key %q: %v", ErrReflectUnsupportedType, key, err)
		}
		return []*Node{NewStringNode(key, string(text))}, nil
	}

	switch rv.Kind() {
	case reflect.String:
		return []*Node{NewStringNode(key, rv.String())}, nil

	case reflect.Uint32:
		return []*Node{NewUint32Node(key, uint32(rv.Uint()))}, nil //nolint:gosec // reflect.Uint32 guarantees value fits

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint64:
		v := rv.Uint()
		if v > math.MaxUint32 {
			return nil, fmt.Errorf("%w: key %q value %d overflows uint32", ErrIntOutOfRange, key, v)
		}
		return []*Node{NewUint32Node(key, uint32(v))}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v := rv.Int()
		if v < 0 || v > math.MaxUint32 {
			return nil, fmt.Errorf("%w: key %q value %d out of uint32 range", ErrIntOutOfRange, key, v)
		}
		return []*Node{NewUint32Node(key, uint32(v))}, nil

	case reflect.Bool:
		if rv.Bool() {
			return []*Node{NewStringNode(key, "1")}, nil
		}
		return []*Node{NewStringNode(key, "0")}, nil

	case reflect.Struct:
		node, err := marshalStructToNode(key, rv)
		if err != nil {
			return nil, err
		}
		return []*Node{node}, nil

	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("%w: map key must be string for key %q", ErrReflectUnsupportedType, key)
		}

		obj := NewObjectNode(key)
		for _, mk := range rv.MapKeys() {
			childNodes, err := marshalValue(mk.String(), vdfTag{name: mk.String()}, rv.MapIndex(mk))
			if err != nil {
				return nil, err
			}
			obj.Children = append(obj.Children, childNodes...)
		}

		return []*Node{obj}, nil

	case reflect.Slice:
		if tag.indexed {
			obj := NewObjectNode(key)
			for i := 0; i < rv.Len(); i++ {
				idxKey := strconv.Itoa(i)
				elemNodes, err := marshalValue(idxKey, vdfTag{name: idxKey}, rv.Index(i))
				if err != nil {
					return nil, err
				}

				obj.Children = append(obj.Children, elemNodes...)
			}

			return []*Node{obj}, nil
		}

		if tag.repeated {
			nodes := make([]*Node, 0, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				elemNodes, err := marshalValue(key, vdfTag{name: key}, rv.Index(i))
				if err != nil {
					return nil, err
				}

				nodes = append(nodes, elemNodes...)
			}

			return nodes, nil
		}

		return nil, fmt.Errorf("%w: slice field %q requires 'indexed' or 'repeated' tag", ErrReflectUnsupportedType, key)

	default:
		return nil, fmt.Errorf("%w: unsupported kind %s for key %q", ErrReflectUnsupportedType, rv.Kind(), key)
	}
}

// Unmarshal decodes the named root object from doc into the struct pointed to by out.
// out must be a non-nil pointer to a struct.
func Unmarshal(doc *Document, root string, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("%w: Unmarshal requires a non-nil pointer", ErrReflectUnsupportedType)
	}
	rv = rv.Elem()

	var rootNode *Node
	for _, n := range doc.Roots {
		if n != nil && n.Key == root {
			rootNode = n
			break
		}
	}

	if rootNode == nil {
		return fmt.Errorf("%w: root key %q not found in document", ErrReflectFieldMismatch, root)
	}

	return unmarshalNode(rootNode, rv)
}

// unmarshalNode populates rv from a VDF node.
func unmarshalNode(node *Node, rv reflect.Value) error {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		rv = rv.Elem()
	}

	// TextUnmarshaler via pointer receiver.
	if rv.CanAddr() {
		if tu, ok := rv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			if node.Kind != NodeString {
				return fmt.Errorf("%w: TextUnmarshaler key %q expects string node", ErrReflectFieldMismatch, node.Key)
			}
			return tu.UnmarshalText([]byte(*node.StringValue))
		}
	}

	switch rv.Kind() {
	case reflect.String:
		if node.Kind != NodeString {
			return fmt.Errorf("%w: key %q expects string node", ErrReflectFieldMismatch, node.Key)
		}
		rv.SetString(*node.StringValue)

	case reflect.Uint32:
		return unmarshalUint(node, rv, 32)

	case reflect.Uint, reflect.Uint16:
		return unmarshalUint(node, rv, 64)

	case reflect.Uint8:
		return unmarshalUint(node, rv, 8)

	case reflect.Uint64:
		return unmarshalUint(node, rv, 64)

	case reflect.Int, reflect.Int32, reflect.Int64:
		return unmarshalInt(node, rv, 64)

	case reflect.Int8:
		return unmarshalInt(node, rv, 8)

	case reflect.Int16:
		return unmarshalInt(node, rv, 16)

	case reflect.Bool:
		if node.Kind != NodeString {
			return fmt.Errorf("%w: key %q expects string node for bool", ErrReflectFieldMismatch, node.Key)
		}
		rv.SetBool(*node.StringValue == "1" || strings.EqualFold(*node.StringValue, "true"))

	case reflect.Struct:
		if node.Kind != NodeObject {
			return fmt.Errorf("%w: key %q expects object node for struct", ErrReflectFieldMismatch, node.Key)
		}
		return unmarshalStruct(node, rv)

	default:
		return fmt.Errorf("%w: unsupported kind %s for key %q", ErrReflectUnsupportedType, rv.Kind(), node.Key)
	}

	return nil
}

func unmarshalUint(node *Node, rv reflect.Value, bits int) error {
	var v uint64
	switch node.Kind {
	case NodeUint32:
		v = uint64(*node.Uint32Value)

	case NodeString:
		var err error
		v, err = strconv.ParseUint(*node.StringValue, 10, bits)
		if err != nil {
			return fmt.Errorf("%w: key %q cannot parse uint: %v", ErrReflectFieldMismatch, node.Key, err)
		}

	default:
		return fmt.Errorf("%w: key %q expects uint32 or string node", ErrReflectFieldMismatch, node.Key)
	}

	rv.SetUint(v)
	return nil
}

func unmarshalInt(node *Node, rv reflect.Value, bits int) error {
	var v int64
	switch node.Kind {
	case NodeUint32:
		v = int64(*node.Uint32Value)

	case NodeString:
		var err error
		v, err = strconv.ParseInt(*node.StringValue, 10, bits)
		if err != nil {
			return fmt.Errorf("%w: key %q cannot parse int: %v", ErrReflectFieldMismatch, node.Key, err)
		}

	default:
		return fmt.Errorf("%w: key %q expects uint32 or string node", ErrReflectFieldMismatch, node.Key)
	}

	rv.SetInt(v)
	return nil
}

// unmarshalStruct populates a struct from a NodeObject's children.
func unmarshalStruct(node *Node, rv reflect.Value) error {
	fields := getFields(rv.Type())

	for _, fi := range fields {
		fv := rv.Field(fi.index)
		ft := rv.Type().Field(fi.index)

		if fi.tag.inline {
			tmp := fv
			for tmp.Kind() == reflect.Pointer {
				if tmp.IsNil() {
					tmp.Set(reflect.New(tmp.Type().Elem()))
				}
				tmp = tmp.Elem()
			}

			if tmp.Kind() != reflect.Struct {
				return fmt.Errorf("%w: inline field %q must be a struct", ErrReflectUnsupportedType, ft.Name)
			}

			if err := unmarshalStruct(node, tmp); err != nil {
				return fmt.Errorf("inline field %q: %w", ft.Name, err)
			}
			continue
		}

		if fi.tag.repeated {
			children := node.All(fi.tag.name)
			if len(children) == 0 {
				continue
			}

			sliceType := fv.Type()
			if sliceType.Kind() != reflect.Slice {
				return fmt.Errorf("%w: repeated field %q must be a slice", ErrReflectUnsupportedType, ft.Name)
			}

			elemType := sliceType.Elem()
			slice := reflect.MakeSlice(sliceType, 0, len(children))
			for i, child := range children {
				elem := reflect.New(elemType).Elem()
				if err := unmarshalNode(child, elem); err != nil {
					return fmt.Errorf("repeated field %q[%d]: %w", ft.Name, i, err)
				}
				slice = reflect.Append(slice, elem)
			}
			fv.Set(slice)
			continue
		}

		if fi.tag.indexed {
			child := node.First(fi.tag.name)
			if child == nil {
				continue
			}

			if child.Kind != NodeObject {
				return fmt.Errorf("%w: indexed field %q expects object node", ErrReflectFieldMismatch, ft.Name)
			}

			sliceType := fv.Type()
			if sliceType.Kind() != reflect.Slice {
				return fmt.Errorf("%w: indexed field %q must be a slice", ErrReflectUnsupportedType, ft.Name)
			}

			elemType := sliceType.Elem()
			slice := reflect.MakeSlice(sliceType, 0, len(child.Children))
			for i, c := range child.Children {
				elem := reflect.New(elemType).Elem()
				if err := unmarshalNode(c, elem); err != nil {
					return fmt.Errorf("indexed field %q[%d]: %w", ft.Name, i, err)
				}
				slice = reflect.Append(slice, elem)
			}
			fv.Set(slice)
			continue
		}

		child := node.First(fi.tag.name)
		if child == nil {
			continue
		}

		if err := unmarshalNode(child, fv); err != nil {
			return fmt.Errorf("field %q: %w", ft.Name, err)
		}
	}

	return nil
}
