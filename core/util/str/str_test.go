package str

import (
	"testing"
)

func TestDeref(t *testing.T) {
	var str = "foo"

	type args struct {
		s *string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Tests dereferencing a string pointer",
			args: args{
				s: &str,
			},
			want: "foo",
		},
		{
			name: "Tests dereferencing a nil pointer",
			args: args{
				s: nil,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Deref(tt.args.s); got != tt.want {
				t.Errorf("Deref() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNorm(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Tests normalising a string",
			args: args{
				s: " FOO-bar__Baz ",
			},
			want: "foobarbaz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Norm(tt.args.s); got != tt.want {
				t.Errorf("Norm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	type args struct {
		str  string
		list []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Tests a list containing a string",
			args: args{
				str:  "bar",
				list: []string{"foo", "bar", "baz"},
			},
			want: true,
		},
		{
			name: "Tests a list not containing a string",
			args: args{
				str:  "ba",
				list: []string{"foo", "bar", "baz"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.str, tt.args.list); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
