// Package docparse implements parsers to transform input into separated documents.
package docparse

import (
	"reflect"
	"testing"
)

func Test_bytesToYAMLDocs(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Tests basic doc split",
			args: args{
				bytes: []byte("doc1\n---\ndoc2\n---\ndoc3"),
			},
			want: []string{
				"doc1",
				"doc2",
				"doc3",
			},
		},
		{
			name: "Tests doc split with leading separator",
			args: args{
				bytes: []byte("---\ndoc1\n---\ndoc2\n---\ndoc3"),
			},
			want: []string{
				"doc1",
				"doc2",
				"doc3",
			},
		},
		{
			name: "Tests doc split with unnecessary whitespace",
			args: args{
				bytes: []byte("doc1    \n\n---\n\tdoc2\n---\n\n  doc3"),
			},
			want: []string{
				"doc1",
				"doc2",
				"doc3",
			},
		},
		{
			name: "Tests doc split with YAML comments",
			args: args{
				bytes: []byte("doc1 #foo\n---\n#bar\ndoc2\n---\n#baz\n---\ndoc3"),
			},
			want: []string{
				"doc1",
				"doc2",
				"doc3",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytesToYAMLDocs(tt.args.bytes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bytesToYAMLDocs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_bytesToJSONDocs(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Tests handling a single JSON doc (non-array)",
			args: args{
				bytes: []byte("{\"name\": \"foo\"}"),
			},
			want: []string{
				"{\"name\":\"foo\"}",
			},
			wantErr: false,
		},
		{
			name: "Tests handling an array of JSON docs",
			args: args{
				bytes: []byte("[{\"name\": \"foo\"},{\"name\": \"bar\"}]"),
			},
			want: []string{
				"{\"name\":\"foo\"}",
				"{\"name\":\"bar\"}",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesToJSONDocs(tt.args.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("bytesToJSONDocs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bytesToJSONDocs() = %v, want %v", got, tt.want)
			}
		})
	}
}
