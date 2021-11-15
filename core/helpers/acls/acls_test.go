// Package acls implements helper functions for handling ACLEntryGroups.
package acls

import (
	"reflect"
	"testing"

	"github.com/peter-evans/kdef/core/model/def"
)

func TestDiffPatchIntersection(t *testing.T) {
	fooGroup := def.ACLEntryGroup{
		Principals:     []string{"foo"},
		Hosts:          []string{"*"},
		Operations:     []string{"READ"},
		PermissionType: "ALLOW",
	}
	barGroup := def.ACLEntryGroup{
		Principals:     []string{"foo"},
		Hosts:          []string{"*"},
		Operations:     []string{"WRITE"},
		PermissionType: "ALLOW",
	}
	bazGroup := def.ACLEntryGroup{
		Principals:     []string{"bar"},
		Hosts:          []string{"*"},
		Operations:     []string{"CREATE"},
		PermissionType: "DENY",
	}

	type args struct {
		a def.ACLEntryGroups
		b def.ACLEntryGroups
	}
	tests := []struct {
		name             string
		args             args
		wantPatch        def.ACLEntryGroups
		wantIntersection def.ACLEntryGroups
	}{
		{
			name: "Tests empty acl entry groups",
			args: args{
				a: def.ACLEntryGroups{},
				b: def.ACLEntryGroups{},
			},
			wantPatch:        nil,
			wantIntersection: nil,
		},
		{
			name: "Tests identical acl entry groups",
			args: args{
				a: def.ACLEntryGroups{fooGroup, barGroup, bazGroup},
				b: def.ACLEntryGroups{fooGroup, barGroup, bazGroup},
			},
			wantPatch:        nil,
			wantIntersection: def.ACLEntryGroups{fooGroup, barGroup, bazGroup},
		},
		{
			name: "Tests acl entry groups with no intersection",
			args: args{
				a: def.ACLEntryGroups{fooGroup, bazGroup},
				b: def.ACLEntryGroups{barGroup},
			},
			wantPatch:        def.ACLEntryGroups{fooGroup, bazGroup},
			wantIntersection: nil,
		},
		{
			name: "Tests acl entry groups with patch and intersection",
			args: args{
				a: def.ACLEntryGroups{fooGroup, barGroup, bazGroup},
				b: def.ACLEntryGroups{fooGroup, bazGroup},
			},
			wantPatch:        def.ACLEntryGroups{barGroup},
			wantIntersection: def.ACLEntryGroups{fooGroup, bazGroup},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, intersection := DiffPatchIntersection(tt.args.a, tt.args.b)
			if !reflect.DeepEqual(patch, tt.wantPatch) {
				t.Errorf("DiffPatchIntersection() patch = %v, want %v", patch, tt.wantPatch)
			}
			if !reflect.DeepEqual(intersection, tt.wantIntersection) {
				t.Errorf("DiffPatchIntersection() intersection = %v, want %v", intersection, tt.wantIntersection)
			}
		})
	}
}

func TestMergeGroups(t *testing.T) {
	type args struct {
		groups def.ACLEntryGroups
	}
	tests := []struct {
		name string
		args args
		want def.ACLEntryGroups
	}{
		{
			name: "Test merging acl entry groups (2 iterations)",
			args: args{
				groups: def.ACLEntryGroups{
					def.ACLEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ"},
						PermissionType: "ALLOW",
					},
					def.ACLEntryGroup{
						Principals:     []string{"bar"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.ACLEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"WRITE"},
						PermissionType: "ALLOW",
					},
					def.ACLEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.ACLEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
				},
			},
			want: def.ACLEntryGroups{
				def.ACLEntryGroup{
					Principals:     []string{"foo"},
					Hosts:          []string{"*"},
					Operations:     []string{"READ", "WRITE"},
					PermissionType: "ALLOW",
				},
				def.ACLEntryGroup{
					Principals:     []string{"bar", "baz", "foo"},
					Hosts:          []string{"*"},
					Operations:     []string{"CREATE"},
					PermissionType: "DENY",
				},
			},
		},
		{
			name: "Test merging acl entry groups (3 iterations)",
			args: args{
				groups: def.ACLEntryGroups{
					def.ACLEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ", "WRITE"},
						PermissionType: "ALLOW",
					},
					def.ACLEntryGroup{
						Principals:     []string{"bar", "baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"DELETE"},
						PermissionType: "DENY",
					},
					def.ACLEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ"},
						PermissionType: "ALLOW",
					},
					def.ACLEntryGroup{
						Principals:     []string{"bar"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.ACLEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"WRITE"},
						PermissionType: "ALLOW",
					},
					def.ACLEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
				},
			},
			want: def.ACLEntryGroups{
				def.ACLEntryGroup{
					Principals:     []string{"baz", "foo"},
					Hosts:          []string{"*"},
					Operations:     []string{"READ", "WRITE"},
					PermissionType: "ALLOW",
				},
				def.ACLEntryGroup{
					Principals:     []string{"bar", "baz"},
					Hosts:          []string{"*"},
					Operations:     []string{"DELETE", "CREATE"},
					PermissionType: "DENY",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeGroups(tt.args.groups); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeGroups() = %v, want %v", got, tt.want)
			}
		})
	}
}
