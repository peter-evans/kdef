//go:build integration
// +build integration

// Package broker implements operators for broker definition operations.
package broker

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
	"github.com/peter-evans/kdef/core/test/harness"
	"github.com/peter-evans/kdef/core/test/tutil"
)

// VERBOSE_TESTS=1 go test --tags=integration -run ^Test_applier_Execute$ ./core/operators/broker -v
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
	cl := tutil.CreateClient(t,
		[]string{fmt.Sprintf("seedBrokers=localhost:%d", harness.BrokerApplier.BrokerPort)},
	)

	// Create client set to use non-incremental alter configs
	clNonInc := tutil.CreateClient(t,
		[]string{
			fmt.Sprintf("seedBrokers=localhost:%d", harness.BrokerApplier.BrokerPort),
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
			harness.BrokerApplier.ComposeFilePaths,
			harness.BrokerApplier.Env(),
		)
		if srv.IsKafkaReady(harness.BrokerApplier.Brokers, 90) {
			duration := time.Since(start)
			log.Infof("kafka cluster ready in %v", duration)
			defer compose.Down(t, c)
			break
		} else {
			log.Warnf("kafka failed to be ready within timeout")
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
	broker1Docs := tutil.FileToYAMLDocs(t, "../../test/fixtures/broker/core.operators.broker.applier.1.yml")
	broker1Diffs := getDiffsFixture(t, "../../test/fixtures/broker/core.operators.broker.applier.1.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Add configs
			name: "1: Dry-run broker config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    broker1Diffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add configs
			name: "2: Apply broker config foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    broker1Diffs[0],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Update configs
			name: "3: Dry-run broker foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    broker1Diffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs
			name: "4: Apply broker foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    broker1Diffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs
			name: "5: Dry-run broker foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    broker1Diffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Delete configs
			name: "6: Apply broker foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: broker1Docs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    broker1Diffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Update configs (non-incremental)
			name: "7: Dry-run broker foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: broker1Docs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    broker1Diffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs (non-incremental)
			name: "8: Apply broker foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: broker1Docs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    broker1Diffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs (non-incremental)
			// Fail due to deletion of undefined configs being not enabled
			name: "9: Dry-run broker foo version 4",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: broker1Docs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    broker1Diffs[4],
			wantErr:     "cannot apply configs because deletion of undefined configs is not enabled",
			wantApplied: false,
		},
		{
			// Delete configs (non-incremental)
			name: "10: Apply broker foo version 5",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: broker1Docs[5],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    broker1Diffs[5],
			wantErr:     "",
			wantApplied: true,
		},
	})
}
