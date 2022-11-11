// Package docparse implements parsers to transform input into separated documents.
package docparse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Format represents the format of the documents to be parsed.
type Format int8

// Supported formats.
const (
	YAML Format = 1
	JSON Format = 2
)

var (
	yamlDocSeparatorRegExp = regexp.MustCompile(`(?m)^---`)
	yamlCommentRegExp      = regexp.MustCompile(`(?m)^([^#]*)#?.*$`)
)

// FromFile parses a file to a slice of separated documents.
func FromFile(filepath string, format Format) ([]string, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	switch format {
	case YAML:
		return bytesToYAMLDocs(b), nil
	case JSON:
		docs, err := bytesToJSONDocs(b)
		if err != nil {
			return nil, err
		}
		return docs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}

// FromStdin parses stdin to a slice of separated documents.
func FromStdin(format Format) ([]string, error) {
	var b []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		b = append(b, scanner.Bytes()...)
		b = append(b, "\n"...)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	switch format {
	case YAML:
		return bytesToYAMLDocs(b), nil
	case JSON:
		docs, err := bytesToJSONDocs(b)
		if err != nil {
			return nil, err
		}
		return docs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}

func bytesToYAMLDocs(b []byte) []string {
	cleanBytes := yamlCommentRegExp.ReplaceAll(b, []byte("$1"))
	docs := yamlDocSeparatorRegExp.Split(string(cleanBytes), -1)

	var yDocs []string
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if len(doc) > 0 {
			yDocs = append(yDocs, doc)
		}
	}
	return yDocs
}

func bytesToJSONDocs(b []byte) ([]string, error) {
	var bi interface{}
	if err := json.Unmarshal(b, &bi); err != nil {
		return nil, err
	}

	var docs []interface{}
	switch v := bi.(type) {
	case []interface{}:
		docs = v
	case interface{}:
		docs = []interface{}{v}
	default:
		return nil, fmt.Errorf("json document is invalid")
	}

	jDocs := make([]string, len(docs))
	for i, doc := range docs {
		jb, err := json.Marshal(doc)
		if err != nil {
			return nil, err
		}
		jDocs[i] = string(jb)
	}

	return jDocs, nil
}
