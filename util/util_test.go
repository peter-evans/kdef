package util

import (
	"testing"
)

func TestDerefStr(t *testing.T) {
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
			if got := DerefStr(tt.args.s); got != tt.want {
				t.Errorf("DerefStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormStr(t *testing.T) {
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
			if got := NormStr(tt.args.s); got != tt.want {
				t.Errorf("NormStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringInList(t *testing.T) {
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
			if got := StringInList(tt.args.str, tt.args.list); got != tt.want {
				t.Errorf("StringInList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_duplicateInSlice(t *testing.T) {
	type args struct {
		s []int32
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Tests when a duplicate is contained in the slice",
			args: args{
				s: []int32{1, 2, 3, 4, 2, 5},
			},
			want: true,
		},
		{
			name: "Tests when a duplicate is not contained in the slice",
			args: args{
				s: []int32{1, 2, 3, 4, 5, 6},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DuplicateInSlice(tt.args.s); got != tt.want {
				t.Errorf("duplicateInSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
