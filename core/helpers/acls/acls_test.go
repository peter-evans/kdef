package acls

import (
	"reflect"
	"testing"

	"github.com/peter-evans/kdef/core/model/def"
)

func TestDiffPatchIntersection(t *testing.T) {
	fooGroup := def.AclEntryGroup{
		Principals:     []string{"foo"},
		Hosts:          []string{"*"},
		Operations:     []string{"READ"},
		PermissionType: "ALLOW",
	}
	barGroup := def.AclEntryGroup{
		Principals:     []string{"foo"},
		Hosts:          []string{"*"},
		Operations:     []string{"WRITE"},
		PermissionType: "ALLOW",
	}
	bazGroup := def.AclEntryGroup{
		Principals:     []string{"bar"},
		Hosts:          []string{"*"},
		Operations:     []string{"CREATE"},
		PermissionType: "DENY",
	}

	type args struct {
		a def.AclEntryGroups
		b def.AclEntryGroups
	}
	tests := []struct {
		name             string
		args             args
		wantPatch        def.AclEntryGroups
		wantIntersection def.AclEntryGroups
	}{
		{
			name: "Tests empty acl entry groups",
			args: args{
				a: def.AclEntryGroups{},
				b: def.AclEntryGroups{},
			},
			wantPatch:        nil,
			wantIntersection: nil,
		},
		{
			name: "Tests identical acl entry groups",
			args: args{
				a: def.AclEntryGroups{fooGroup, barGroup, bazGroup},
				b: def.AclEntryGroups{fooGroup, barGroup, bazGroup},
			},
			wantPatch:        nil,
			wantIntersection: def.AclEntryGroups{fooGroup, barGroup, bazGroup},
		},
		{
			name: "Tests acl entry groups with no intersection",
			args: args{
				a: def.AclEntryGroups{fooGroup, bazGroup},
				b: def.AclEntryGroups{barGroup},
			},
			wantPatch:        def.AclEntryGroups{fooGroup, bazGroup},
			wantIntersection: nil,
		},
		{
			name: "Tests acl entry groups with patch and intersection",
			args: args{
				a: def.AclEntryGroups{fooGroup, barGroup, bazGroup},
				b: def.AclEntryGroups{fooGroup, bazGroup},
			},
			wantPatch:        def.AclEntryGroups{barGroup},
			wantIntersection: def.AclEntryGroups{fooGroup, bazGroup},
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
		groups def.AclEntryGroups
	}
	tests := []struct {
		name string
		args args
		want def.AclEntryGroups
	}{
		{
			name: "Test merging acl entry groups (2 iterations)",
			args: args{
				groups: def.AclEntryGroups{
					def.AclEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ"},
						PermissionType: "ALLOW",
					},
					def.AclEntryGroup{
						Principals:     []string{"bar"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.AclEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"WRITE"},
						PermissionType: "ALLOW",
					},
					def.AclEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.AclEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
				},
			},
			want: def.AclEntryGroups{
				def.AclEntryGroup{
					Principals:     []string{"foo"},
					Hosts:          []string{"*"},
					Operations:     []string{"READ", "WRITE"},
					PermissionType: "ALLOW",
				},
				def.AclEntryGroup{
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
				groups: def.AclEntryGroups{
					def.AclEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ", "WRITE"},
						PermissionType: "ALLOW",
					},
					def.AclEntryGroup{
						Principals:     []string{"bar", "baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"DELETE"},
						PermissionType: "DENY",
					},
					def.AclEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"READ"},
						PermissionType: "ALLOW",
					},
					def.AclEntryGroup{
						Principals:     []string{"bar"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
					def.AclEntryGroup{
						Principals:     []string{"foo"},
						Hosts:          []string{"*"},
						Operations:     []string{"WRITE"},
						PermissionType: "ALLOW",
					},
					def.AclEntryGroup{
						Principals:     []string{"baz"},
						Hosts:          []string{"*"},
						Operations:     []string{"CREATE"},
						PermissionType: "DENY",
					},
				},
			},
			want: def.AclEntryGroups{
				def.AclEntryGroup{
					Principals:     []string{"baz", "foo"},
					Hosts:          []string{"*"},
					Operations:     []string{"READ", "WRITE"},
					PermissionType: "ALLOW",
				},
				def.AclEntryGroup{
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
