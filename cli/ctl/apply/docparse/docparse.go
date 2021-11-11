package docparse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Format int8

const (
	YAML Format = 1
	JSON Format = 2
)

var (
	yamlDocSeparatorRegExp = regexp.MustCompile(`(?m)^---`)
	yamlCommentRegExp      = regexp.MustCompile(`(?m)^([^#]*)#?.*$`)
)

// Parses a file to a slice of separated documents
func FromFile(filepath string, format Format) ([]string, error) {
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	switch format {
	case YAML:
		return bytesToYAMLDocs(fileBytes), nil
	case JSON:
		jsonDocs, err := bytesToJSONDocs(fileBytes)
		if err != nil {
			return nil, err
		}
		return jsonDocs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}

// Parses stdin to a slice of separated documents
func FromStdin(format Format) ([]string, error) {
	var stdinBytes []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stdinBytes = append(stdinBytes, scanner.Bytes()...)
		stdinBytes = append(stdinBytes, "\n"...)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	switch format {
	case YAML:
		return bytesToYAMLDocs(stdinBytes), nil
	case JSON:
		jsonDocs, err := bytesToJSONDocs(stdinBytes)
		if err != nil {
			return nil, err
		}
		return jsonDocs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}

// Converts bytes to a slice of yaml documents
func bytesToYAMLDocs(bytes []byte) []string {
	// Remove yaml comments
	cleanFileBytes := yamlCommentRegExp.ReplaceAll(bytes, []byte("$1"))

	// Separate into yaml documents
	separatedDocs := yamlDocSeparatorRegExp.Split(string(cleanFileBytes), -1)

	var yamlDocs []string
	for _, doc := range separatedDocs {
		doc = strings.TrimSpace(doc)
		if len(doc) > 0 {
			yamlDocs = append(yamlDocs, doc)
		}
	}
	return yamlDocs
}

// Converts bytes to a slice of json documents
func bytesToJSONDocs(bytes []byte) ([]string, error) {
	var iBytes interface{}
	if err := json.Unmarshal(bytes, &iBytes); err != nil {
		return nil, err
	}

	var iDocs []interface{}
	switch v := iBytes.(type) {
	case []interface{}:
		iDocs = v
	case interface{}:
		iDocs = []interface{}{v}
	default:
		return nil, fmt.Errorf("json document is invalid")
	}

	jDocs := make([]string, len(iDocs))
	for i, iDoc := range iDocs {
		jBytes, err := json.Marshal(iDoc)
		if err != nil {
			return nil, err
		}
		jDocs[i] = string(jBytes)
	}

	return jDocs, nil
}
