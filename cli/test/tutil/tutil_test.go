// Package tutil implements testing utility functions.
package tutil

import (
	"errors"
	"testing"
)

func TestErrorContains(t *testing.T) {
	type args struct {
		out  error
		want string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Tests nil error",
			args: args{
				out:  nil,
				want: "",
			},
			want: true,
		},
		{
			name: "Tests that error contains the wanted content",
			args: args{
				out:  errors.New("something exploded"),
				want: "exploded",
			},
			want: true,
		},
		{
			name: "Tests that error does not contain the wanted content",
			args: args{
				out:  errors.New("something died"),
				want: "exploded",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorContains(tt.args.out, tt.args.want); got != tt.want {
				t.Errorf("ErrorContains() = %v, want %v", got, tt.want)
			}
		})
	}
}
