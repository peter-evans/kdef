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
			name: "Derefs a string",
			args: args{
				s: &str,
			},
			want: "foo",
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
			name: "Normalises a string",
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
			name: "List contains string",
			args: args{
				str:  "bar",
				list: []string{"foo", "bar", "baz"},
			},
			want: true,
		},
		{
			name: "List does not contain string",
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
