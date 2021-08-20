package in

import (
	"reflect"
	"testing"
)

func Test_bytesToYamlDocs(t *testing.T) {
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
			name: "Tests basic doc split",
			args: args{
				bytes: []byte("doc1\n---\ndoc2\n---\ndoc3"),
			},
			want: []string{
				"doc1",
				"doc2",
				"doc3",
			},
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
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
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bytesToYamlDocs(tt.args.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("bytesToYamlDocs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("bytesToYamlDocs() = %v, want %v", got, tt.want)
			}
		})
	}
}
