// +build integration

package brokers

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/test/compose"
	"github.com/peter-evans/kdef/core/test/compose_fixture"
	"github.com/peter-evans/kdef/core/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_applier_Execute$ ./core/operators/brokers -v
func Test_applier_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	type fields struct {
		cl      *client.Client
		yamlDoc string
		opts    ApplierOptions
	}
	type testCase struct {
		name        string
		fields      fields
		wantDiff    string
		wantErr     string
		wantApplied bool
	}

	runTests := func(t *testing.T, tests []testCase) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := NewApplier(tt.fields.cl, tt.fields.yamlDoc, tt.fields.opts)
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
				if got.Applied != tt.wantApplied {
					t.Errorf("applier.Execute().Applied = %v, want %v", got.Applied, tt.wantApplied)
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

	// Create client
	cl := tutil.CreateClient(
		t,
		[]string{fmt.Sprintf("seedBrokers=localhost:%d", compose_fixture.BrokersApplierComposeFixture.BrokerPort)},
	)

	// Create client set to use non-incremental alter configs
	clNonInc := tutil.CreateClient(t,
		[]string{
			fmt.Sprintf("seedBrokers=localhost:%d", compose_fixture.BrokersApplierComposeFixture.BrokerPort),
			"alterConfigsMethod=non-incremental",
		},
	)

	// Create the test cluster
	srv := kafka.NewService(cl)
	maxTries := 3
	try := 1
	for {
		start := time.Now()
		c := compose.Up(
			t,
			compose_fixture.BrokersApplierComposeFixture.ComposeFilePaths,
			compose_fixture.BrokersApplierComposeFixture.Env(),
		)
		if srv.IsKafkaReady(compose_fixture.BrokersApplierComposeFixture.Brokers, 90) {
			duration := time.Since(start)
			log.Info("kafka cluster ready in %v", duration)
			defer compose.Down(t, c)
			break
		} else {
			log.Warn("kafka failed to be ready within timeout")
			compose.Down(t, c)
			try++
		}
		if try > maxTries {
			t.Errorf("kafka failed to be ready within timeout after %d tries", maxTries)
			t.FailNow()
		}
		time.Sleep(2 * time.Second)
	}

	// Tests changes to configs
	fooDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/brokers/core.operators.brokers.applier.foo.yml")
	fooDiffs := getDiffsFixture(t, "../../test/fixtures/brokers/core.operators.brokers.applier.foo.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Add configs
			name: "1: Dry-run brokers config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add configs
			name: "2: Apply brokers config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
				},
			},
			wantDiff:    fooDiffs[0],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// No diff check
			name: "3: Dry-run brokers config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    "",
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs
			name: "4: Dry-run brokers foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs
			name: "5: Apply brokers foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
				},
			},
			wantDiff:    fooDiffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs
			name: "6: Dry-run brokers foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Delete configs
			name: "7: Apply brokers foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
				},
			},
			wantDiff:    fooDiffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Update configs (non-incremental)
			name: "8: Dry-run brokers foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs (non-incremental)
			name: "9: Apply brokers foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
				},
			},
			wantDiff:    fooDiffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs (non-incremental)
			// Fail due to deletion of undefined configs being not enabled
			name: "10: Dry-run brokers foo version 4",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[4],
			wantErr:     "cannot apply configs because deletion of undefined configs is not enabled",
			wantApplied: false,
		},
		{
			// Delete configs (non-incremental)
			name: "11: Apply brokers foo version 5",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[5],
				opts: ApplierOptions{
					DefinitionFormat: opt.YamlFormat,
				},
			},
			wantDiff:    fooDiffs[5],
			wantErr:     "",
			wantApplied: true,
		},
	})
}
