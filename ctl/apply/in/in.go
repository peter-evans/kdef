package in

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/peter-evans/kdef/core/model/opt"
)

var (
	yamlDocSeparatorRegExp = regexp.MustCompile(`(?m)^---`)
	yamlCommentRegExp      = regexp.MustCompile(`(?m)^([^#]*)#?.*$`)
)

// Converts bytes to an array of yaml documents
func bytesToYamlDocs(bytes []byte) ([]string, error) {
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
	return yamlDocs, nil
}

// Converts bytes to an array of json documents
func bytesToJsonDocs(bytes []byte) ([]string, error) {
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

// Reads a file to an array of separated documents
func FileToSeparatedDocs(filepath string, format opt.DefinitionFormat) ([]string, error) {
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	switch format {
	case opt.YamlFormat:
		yamlDocs, err := bytesToYamlDocs(fileBytes)
		if err != nil {
			return nil, err
		}
		return yamlDocs, nil
	case opt.JsonFormat:
		jsonDocs, err := bytesToJsonDocs(fileBytes)
		if err != nil {
			return nil, err
		}
		return jsonDocs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}

// Reads stdin to an array of separated documents
func StdinToSeparatedDocs(format opt.DefinitionFormat) ([]string, error) {
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
	case opt.YamlFormat:
		yamlDocs, err := bytesToYamlDocs(stdinBytes)
		if err != nil {
			return nil, err
		}
		return yamlDocs, nil
	case opt.JsonFormat:
		jsonDocs, err := bytesToJsonDocs(stdinBytes)
		if err != nil {
			return nil, err
		}
		return jsonDocs, nil
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}
