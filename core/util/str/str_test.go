package str

import (
	"reflect"
	"testing"
)

func TestDeref(t *testing.T) {
	str := "foo"

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
		str string
		s   []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Tests a slice containing a string",
			args: args{
				str: "bar",
				s:   []string{"foo", "bar", "baz"},
			},
			want: true,
		},
		{
			name: "Tests a slice not containing a string",
			args: args{
				str: "ba",
				s:   []string{"foo", "bar", "baz"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.str, tt.args.s); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnorderedEqual(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test slices of different length",
			args: args{
				a: []string{"foo", "bar", "baz"},
				b: []string{"foo", "bar"},
			},
			want: false,
		},
		{
			name: "Test an element in slice B not existing in A",
			args: args{
				a: []string{"foo", "bar", "baz"},
				b: []string{"qux", "foo", "bar"},
			},
			want: false,
		},
		{
			name: "Test a duplicate element in slice B",
			args: args{
				a: []string{"foo", "bar", "baz"},
				b: []string{"foo", "foo", "bar"},
			},
			want: false,
		},
		{
			name: "Test equal slices with differing element orders",
			args: args{
				a: []string{"foo", "bar", "baz"},
				b: []string{"baz", "foo", "bar"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UnorderedEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("UnorderedEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeduplicate(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test a slice with no duplicates",
			args: args{
				s: []string{"foo", "bar", "baz"},
			},
			want: []string{"foo", "bar", "baz"},
		},
		{
			name: "Test a slice with duplicates",
			args: args{
				s: []string{"foo", "foo", "bar", "baz", "bar"},
			},
			want: []string{"foo", "bar", "baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Deduplicate(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Deduplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}
