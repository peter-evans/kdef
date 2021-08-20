package in

import (
	"bufio"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
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

// Reads a file to an array of yaml documents
func FileToYamlDocs(filepath string) ([]string, error) {
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	yamlDocs, err := bytesToYamlDocs(fileBytes)
	if err != nil {
		return nil, err
	}

	return yamlDocs, nil
}

// Reads stdin to an array of yaml documents
func StdinToYamlDocs() ([]string, error) {
	var stdinBytes []byte
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stdinBytes = append(stdinBytes, scanner.Bytes()...)
		stdinBytes = append(stdinBytes, "\n"...)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	yamlDocs, err := bytesToYamlDocs(stdinBytes)
	if err != nil {
		return nil, err
	}

	return yamlDocs, nil
}
