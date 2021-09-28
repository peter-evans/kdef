package brokers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/service"
	"github.com/peter-evans/kdef/test/compose"
	"github.com/peter-evans/kdef/test/fixtures"
	"github.com/peter-evans/kdef/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_applier_Execute$ ./core/operators/brokers -v
func Test_applier_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	type fields struct {
		cl      *client.Client
		yamlDoc string
		flags   ApplierFlags
	}
	type testCase struct {
		name                    string
		fields                  fields
		wantDiff                string
		wantErr                 string
		wantHasUnappliedChanges bool
	}

	runTests := func(t *testing.T, tests []testCase) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := NewApplier(tt.fields.cl, tt.fields.yamlDoc, tt.fields.flags)
				got := a.Execute()

				if log.Verbose {
					// Output apply result JSON
					jsonOut, err := json.MarshalIndent(got, "", "  ")
					if err != nil {
						t.Errorf("failed to convert apply result to json: %v", err)
						t.FailNow()
					}
					fmt.Println("[test] ApplyResult JSON:")
					fmt.Println(string(jsonOut))
				}

				if got.Diff != tt.wantDiff {
					t.Errorf("applier.Execute().Diff = %v, want %v", got.Diff, tt.wantDiff)
				}
				if !tutil.ErrorContains(got.GetErr(), tt.wantErr) {
					t.Errorf("applier.Execute() error = %v, wantErr %v", got.GetErr(), tt.wantErr)
				}
				if got.HasUnappliedChanges() != tt.wantHasUnappliedChanges {
					t.Errorf("applier.Execute().HasUnappliedChanges() = %v, want %v",
						got.HasUnappliedChanges(),
						tt.wantHasUnappliedChanges,
					)
				}

				// Sleep to give Kafka time to update internally
				time.Sleep(2 * time.Second)
			})
		}
	}

	getDiffsFixture := func(t *testing.T, path string) []string {
		var diffs []string
		if err := json.Unmarshal(tutil.Fixture(t, path), &diffs); err != nil {
			t.Errorf("failed to unmarshal JSON test fixture: %v", err)
			t.FailNow()
		}
		return diffs
	}

	// Create the test cluster
	c := compose.Up(
		t,
		fixtures.BrokersApplierTest.ComposeFilePaths,
		fixtures.BrokersApplierTest.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := client.New(&client.ClientFlags{
		ConfigPath: "does-not-exist",
		FlagConfigOpts: []string{
			fmt.Sprintf("seedBrokers=localhost:%d", fixtures.BrokersApplierTest.BrokerPort),
		},
	})

	// Wait for Kafka to be ready
	if !service.IsKafkaReady(cl, fixtures.BrokersApplierTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Tests changes to configs
	fooDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/brokers/core.operators.brokers.applier.foo.yml")
	fooDiffs := getDiffsFixture(t, "../../../test/fixtures/brokers/core.operators.brokers.applier.foo.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Add configs
			name: "1: Dry-run brokers config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                fooDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add configs
			name: "2: Apply brokers config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				flags:   ApplierFlags{},
			},
			wantDiff:                fooDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Update configs
			name: "3: Dry-run brokers foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                fooDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Update configs
			name: "4: Apply brokers foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				flags:   ApplierFlags{},
			},
			wantDiff:                fooDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Delete configs
			name: "5: Dry-run brokers foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				flags: ApplierFlags{
					DryRun:               true,
					DeleteMissingConfigs: true,
				},
			},
			wantDiff:                fooDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Delete configs
			name: "6: Apply brokers foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				flags: ApplierFlags{
					DeleteMissingConfigs: true,
				},
			},
			wantDiff:                fooDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Update configs (non-incremental)
			name: "7: Dry-run brokers foo version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[3],
				flags: ApplierFlags{
					DryRun:         true,
					NonIncremental: true,
				},
			},
			wantDiff:                fooDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Update configs (non-incremental)
			name: "7: Apply brokers foo version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[3],
				flags: ApplierFlags{
					NonIncremental: true,
				},
			},
			wantDiff:                fooDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Delete configs (non-incremental)
			name: "7: Dry-run brokers foo version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[4],
				flags: ApplierFlags{
					DryRun:         true,
					NonIncremental: true,
				},
			},
			wantDiff:                fooDiffs[4],
			wantErr:                 "cannot apply delete config operations because flag --delete-missing-configs is not set",
			wantHasUnappliedChanges: true,
		},
		{
			// Delete configs (non-incremental)
			name: "8: Apply brokers foo version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[4],
				flags: ApplierFlags{
					NonIncremental:       true,
					DeleteMissingConfigs: true,
				},
			},
			wantDiff:                fooDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})
}
