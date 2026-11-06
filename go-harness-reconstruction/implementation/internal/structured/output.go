// Package structured validates bounded child results without schemas fetching
// external files/URLs or implicitly rerunning a child to repair its output.
package structured

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const MaxSchema = 16 << 10
const MaxOutput = 24 << 10

type denyLoader struct{}

func (denyLoader) Load(string) (any, error) { return nil, errors.New("external schemas are disabled") }

// decode preserves JSON integers, rejects duplicate object keys and bounds shape.
func decode(raw []byte, max int) (any, error) {
	if len(raw) > max || !utf8.Valid(raw) {
		return nil, errors.New("JSON exceeds byte/encoding limits")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	nodes := 0
	var value func(int) (any, error)
	value = func(depth int) (any, error) {
		nodes++
		if depth > 32 || nodes > 4096 {
			return nil, errors.New("JSON exceeds shape limits")
		}
		token, err := d.Token()
		if err != nil {
			return nil, errors.New("invalid JSON")
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return token, nil
		}
		switch delim {
		case '{':
			object := map[string]any{}
			for d.More() {
				key, e := d.Token()
				if e != nil {
					return nil, errors.New("invalid JSON key")
				}
				name, ok := key.(string)
				if !ok {
					return nil, errors.New("invalid JSON key")
				}
				if _, ok = object[name]; ok {
					return nil, errors.New("duplicate JSON key")
				}
				child, e := value(depth + 1)
				if e != nil {
					return nil, e
				}
				object[name] = child
			}
			if end, e := d.Token(); e != nil || end != json.Delim('}') {
				return nil, errors.New("invalid JSON object")
			}
			return object, nil
		case '[':
			array := []any{}
			for d.More() {
				child, e := value(depth + 1)
				if e != nil {
					return nil, e
				}
				array = append(array, child)
			}
			if end, e := d.Token(); e != nil || end != json.Delim(']') {
				return nil, errors.New("invalid JSON array")
			}
			return array, nil
		}
		return nil, errors.New("unexpected JSON delimiter")
	}
	out, err := value(0)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON content")
	}
	return out, nil
}
func Compile(raw json.RawMessage) (*jsonschema.Schema, error) {
	value, err := decode(raw, MaxSchema)
	if err != nil {
		return nil, err
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, errors.New("output_schema must be a JSON Schema object")
	}
	nodes := 0
	var inspect func(any, int) error
	inspect = func(v any, depth int) error {
		nodes++
		if depth > 16 || nodes > 512 {
			return errors.New("output schema exceeds shape limits")
		}
		switch x := v.(type) {
		case map[string]any:
			for k, item := range x {
				switch k {
				case "$ref", "$dynamicRef", "$recursiveRef", "$id":
					return errors.New("output schema references/identifiers are unsupported")
				}
				if err := inspect(item, depth+1); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range x {
				if err := inspect(item, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err = inspect(value, 0); err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(denyLoader{})
	const location = "https://harness.invalid/child-output"
	if err = compiler.AddResource(location, value); err != nil {
		return nil, errors.New("invalid output_schema")
	}
	schema, err := compiler.Compile(location)
	if err != nil {
		return nil, errors.New("invalid or unsupported output_schema")
	}
	return schema, nil
}
func Validate(schema *jsonschema.Schema, raw string) (any, error) {
	value, err := decode([]byte(raw), MaxOutput)
	if err != nil {
		return nil, err
	}
	canonical, e := json.Marshal(value)
	if e != nil || len(canonical) > MaxOutput {
		return nil, errors.New("encoded child output exceeds 24KiB")
	}
	if err = schema.Validate(value); err != nil {
		return nil, errors.New("child final response does not match output_schema")
	}
	return value, nil
}
