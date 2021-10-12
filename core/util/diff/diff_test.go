package diff

import (
	"testing"

	"github.com/peter-evans/kdef/core/test/tutil"
)

func TestLineOriented(t *testing.T) {
	type args struct {
		src string
		dst string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "1: Test diff of simple changes",
			args: args{
				src: string(tutil.Fixture(t, "../../test/fixtures/diff/test1-src.json")),
				dst: string(tutil.Fixture(t, "../../test/fixtures/diff/test1-dst.json")),
			},
			want: string(tutil.Fixture(t, "../../test/fixtures/diff/test1.diff")),
		},
		{
			name: "2: Test diff of nested array changes",
			args: args{
				src: string(tutil.Fixture(t, "../../test/fixtures/diff/test2-src.json")),
				dst: string(tutil.Fixture(t, "../../test/fixtures/diff/test2-dst.json")),
			},
			want: string(tutil.Fixture(t, "../../test/fixtures/diff/test2.diff")),
		},
		{
			// Fails when some DiffCleanup options are enabled
			name: "3: Test diff of misc changes",
			args: args{
				src: string(tutil.Fixture(t, "../../test/fixtures/diff/test3-src.json")),
				dst: string(tutil.Fixture(t, "../../test/fixtures/diff/test3-dst.json")),
			},
			want: string(tutil.Fixture(t, "../../test/fixtures/diff/test3.diff")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LineOriented(tt.args.src, tt.args.dst); got != tt.want {
				t.Errorf("LineOriented() = %v, want %v", got, tt.want)
			}
		})
	}
}
