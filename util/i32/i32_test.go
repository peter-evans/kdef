package i32

import (
	"reflect"
	"testing"
)

func Test_ContainsDuplicate(t *testing.T) {
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
			if got := ContainsDuplicate(tt.args.s); got != tt.want {
				t.Errorf("ContainsDuplicate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDiff(t *testing.T) {
	type args struct {
		a []int32
		b []int32
	}
	tests := []struct {
		name string
		args args
		want []int32
	}{
		{
			name: "Tests a diff between two slices",
			args: args{
				a: []int32{1, 2, 3, 4, 5, 6},
				b: []int32{1, 3, 6},
			},
			want: []int32{2, 4, 5},
		},
		{
			name: "Tests no diff",
			args: args{
				a: []int32{1, 2, 3, 4, 5, 6},
				b: []int32{1, 2, 3, 4, 5, 6},
			},
			want: []int32{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Diff(tt.args.a, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Diff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMax(t *testing.T) {
	type args struct {
		s []int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "Test returning the max",
			args: args{
				s: []int32{3, 2, 8, 4, 1},
			},
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Max(tt.args.s); got != tt.want {
				t.Errorf("Max() = %v, want %v", got, tt.want)
			}
		})
	}
}
