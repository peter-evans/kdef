package jsondiff

import (
	"testing"

	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestDiff(t *testing.T) {
	type testObject struct {
		Foo      string   `json:"foo"`
		Bar      string   `json:"bar,omitempty"`
		Baz      int      `json:"baz"`
		Qux      []string `json:"qux,omitempty"`
		internal string
	}

	type args struct {
		a interface{}
		b interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "1: Test computing a JSON diff",
			args: args{
				a: &testObject{
					Foo: "abc",
					Bar: "xyz",
					Baz: 8,
					Qux: []string{
						"foo",
						"bar",
					},
					internal: "foo",
				},
				b: &testObject{
					Foo: "def",
					Bar: "xyz",
					Baz: 8,
					Qux: []string{
						"foo",
						"bar",
						"baz",
					},
				},
			},
			want:    string(tutil.Fixture(t, "../../test/fixtures/jsondiff/test1.diff")),
			wantErr: false,
		},
		{
			name: "2: Test computing a JSON diff when one side is null",
			args: args{
				a: nil,
				b: &testObject{
					Foo: "def",
					Bar: "xyz",
					Baz: 8,
					Qux: []string{
						"foo",
						"bar",
						"baz",
					},
				},
			},
			want:    string(tutil.Fixture(t, "../../test/fixtures/jsondiff/test2.diff")),
			wantErr: false,
		},
		{
			name: "2: Test returning an empty string when both sides are identical",
			args: args{
				a: &testObject{
					Foo: "abc",
					Bar: "xyz",
					Baz: 8,
					Qux: []string{
						"foo",
						"bar",
					},
					internal: "foo",
				},
				b: &testObject{
					Foo: "abc",
					Bar: "xyz",
					Baz: 8,
					Qux: []string{
						"foo",
						"bar",
					},
					internal: "foo",
				},
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Diff(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Diff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Diff() = %v, want %v", got, tt.want)
			}
		})
	}
}
