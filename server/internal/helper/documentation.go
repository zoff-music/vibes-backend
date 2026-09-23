package helper

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"
)

// PublicDocumentation removes admin operations and schemas unreachable from public operations.
// Raw JSON preserves Swagger fields without imposing a second specification model.
func PublicDocumentation(document string) ([]byte, error) {
	var spec map[string]jsontext.Value
	err := json.Unmarshal([]byte(document), &spec)
	if err != nil {
		return nil, fmt.Errorf("error decoding documentation: %w", err)
	}

	var paths map[string]map[string]jsontext.Value
	err = json.Unmarshal(spec["paths"], &paths)
	if err != nil {
		return nil, fmt.Errorf("error decoding documentation paths: %w", err)
	}

	for path, operations := range paths {
		if strings.Contains(path+"/", "/admin/") {
			delete(paths, path)
			continue
		}

		for method, raw := range operations {
			if method == "parameters" || strings.HasPrefix(method, "x-") {
				continue
			}

			var operation map[string]jsontext.Value
			err = json.Unmarshal(raw, &operation)
			if err != nil {
				return nil, fmt.Errorf("error decoding documentation operation: %w", err)
			}

			if tags, ok := operation["tags"]; ok {
				var names []string
				err = json.Unmarshal(tags, &names)
				if err != nil {
					return nil, fmt.Errorf("error decoding documentation tags: %w", err)
				}

				for _, name := range names {
					if name == "admin" {
						delete(operations, method)
						break
					}
				}
			}
		}

		hasOperation := false
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch"} {
			if _, ok := operations[method]; ok {
				hasOperation = true
			}
		}

		if !hasOperation {
			delete(paths, path)
		}
	}

	spec["paths"], err = json.Marshal(paths, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("error encoding public documentation paths: %w", err)
	}

	// Tags are optional grouping metadata; let Swagger derive them from public operations.
	delete(spec, "tags")

	var definitions map[string]jsontext.Value
	if raw, ok := spec["definitions"]; ok {
		err = json.Unmarshal(raw, &definitions)
		if err != nil {
			return nil, fmt.Errorf("error decoding documentation definitions: %w", err)
		}
	}

	delete(spec, "definitions")

	root, err := json.Marshal(spec, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("error encoding documentation root: %w", err)
	}

	references, err := documentationReferences(root)
	if err != nil {
		return nil, fmt.Errorf("error finding documentation references: %w", err)
	}

	reachable := make(map[string]jsontext.Value)
	for len(references) > 0 {
		reference := references[0]
		references = references[1:]
		if !strings.HasPrefix(reference, "#/definitions/") {
			continue
		}

		name := strings.TrimPrefix(reference, "#/definitions/")
		name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
		if _, ok := reachable[name]; ok {
			continue
		}

		definition, ok := definitions[name]
		if !ok {
			return nil, fmt.Errorf("error missing documentation definition %q", name)
		}

		reachable[name] = definition

		nested, err := documentationReferences(definition)
		if err != nil {
			return nil, fmt.Errorf("error finding nested documentation references: %w", err)
		}

		references = append(references, nested...)
	}

	spec["definitions"], err = json.Marshal(reachable, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("error encoding public documentation definitions: %w", err)
	}

	body, err := json.Marshal(spec, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("error encoding public documentation: %w", err)
	}

	return body, nil
}

func documentationReferences(raw jsontext.Value) ([]string, error) {
	var children []jsontext.Value
	references := []string{}
	switch raw.Kind() {
	case '{':
		var fields map[string]jsontext.Value
		err := json.Unmarshal(raw, &fields)
		if err != nil {
			return nil, fmt.Errorf("error decoding documentation object: %w", err)
		}

		for key, value := range fields {
			if key == "$ref" {
				var reference string
				err = json.Unmarshal(value, &reference)
				if err != nil {
					return nil, fmt.Errorf("error decoding documentation reference: %w", err)
				}

				references = append(references, reference)
				continue
			}

			children = append(children, value)
		}

	case '[':
		err := json.Unmarshal(raw, &children)
		if err != nil {
			return nil, fmt.Errorf("error decoding documentation array: %w", err)
		}
	}

	for _, child := range children {

		nested, err := documentationReferences(child)
		if err != nil {
			return nil, fmt.Errorf("error finding child documentation references: %w", err)
		}

		references = append(references, nested...)
	}

	return references, nil
}
