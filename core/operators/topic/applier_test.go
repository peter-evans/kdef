//go:build integration
// +build integration

package topic

import (
	"context"
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
	"github.com/twmb/franz-go/pkg/kgo"
)

// VERBOSE_TESTS=1 go test --tags=integration -run ^Test_applier_Execute$ ./core/operators/topic -v
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
		[]string{fmt.Sprintf("seedBrokers=localhost:%d", harness.TopicsApplier.BrokerPort)},
	)

	// Create client set to use non-incremental alter configs
	clNonInc := tutil.CreateClient(t,
		[]string{
			fmt.Sprintf("seedBrokers=localhost:%d", harness.TopicsApplier.BrokerPort),
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
			harness.TopicsApplier.ComposeFilePaths,
			harness.TopicsApplier.Env(),
		)
		if srv.IsKafkaReady(harness.TopicsApplier.Brokers, 90) {
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
	fooDocs := tutil.FileToYAMLDocs(t, "../../test/fixtures/topic/core.operators.topic.applier.foo.yml")
	fooDiffs := getDiffsFixture(t, "../../test/fixtures/topic/core.operators.topic.applier.foo.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Create topic
			name: "2: Apply topic foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    fooDiffs[0],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// No diff check
			name: "3: Dry-run topic foo version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    "",
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs
			name: "4: Dry-run topic foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs
			name: "5: Apply topic foo version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    fooDiffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs
			name: "6: Dry-run topic foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Delete configs
			name: "7: Apply topic foo version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    fooDiffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Update configs (non-incremental)
			name: "8: Dry-run topic foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Update configs (non-incremental)
			name: "9: Apply topic foo version 3",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    fooDiffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Delete configs (non-incremental)
			// Fail due to deletion of undefined configs being not enabled
			name: "10: Dry-run topic foo version 4",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    fooDiffs[4],
			wantErr:     "cannot apply configs because deletion of undefined configs is not enabled",
			wantApplied: false,
		},
		{
			// Delete configs (non-incremental)
			name: "11: Apply topic foo version 5",
			fields: fields{
				cl:      clNonInc,
				yamlDoc: fooDocs[5],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    fooDiffs[5],
			wantErr:     "",
			wantApplied: true,
		},
	})

	// Tests changes to assignments and handling of in-progress reassignments
	barDocs := tutil.FileToYAMLDocs(t, "../../test/fixtures/topic/core.operators.topic.applier.bar.yml")
	barDiffs := getDiffsFixture(t, "../../test/fixtures/topic/core.operators.topic.applier.bar.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic bar version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[0],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					DryRun:            true,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Create topic
			name: "2: Apply topic bar version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[0],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[0],
			wantErr:     "",
			wantApplied: true,
		},
	})

	// Produce records into topic before proceeding with remaining test cases
	topic := "core.operators.topic.applier.bar"
	t.Logf("Producing records into topic %q before proceeding...", topic)
	val, _ := tutil.RandomBytes(6000)
	for i := 0; i < 1500000; i++ {
		key, _ := tutil.RandomBytes(16)
		r := &kgo.Record{
			Topic: topic,
			Key:   key,
			Value: val,
		}
		cl.Client.Produce(context.Background(), r, func(r *kgo.Record, err error) {})
	}
	if err := cl.Client.Flush(context.Background()); err != nil {
		t.Errorf("failed to produce records: %v", err)
		t.FailNow()
	}

	// Tests changes to assignments and handling of in-progress reassignments (continued)
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Increase replication factor
			name: "3: Dry-run topic bar version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[1],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					DryRun:            true,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Increase replication factor
			name: "4: Apply topic bar version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[1],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions
			name: "5: Dry-run topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					DryRun:            true,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions
			name: "6: Apply topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					DryRun:            true,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					DryRun:            true,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[4],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				opts: ApplierOptions{
					DefinitionFormat:  opt.YAMLFormat,
					ReassAwaitTimeout: 30,
				},
			},
			wantDiff:    barDiffs[4],
			wantErr:     "",
			wantApplied: true,
		},
	})

	// Tests partition and replication factor changes (without static assignments)
	bazDocs := tutil.FileToYAMLDocs(t, "../../test/fixtures/topic/core.operators.topic.applier.baz.yml")
	bazDiffs := getDiffsFixture(t, "../../test/fixtures/topic/core.operators.topic.applier.baz.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic baz version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    bazDiffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Create topic
			name: "2: Apply topic baz version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    bazDiffs[0],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Increase replication factor
			name: "3: Dry-run topic baz version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    bazDiffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Increase replication factor
			name: "4: Apply topic baz version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    bazDiffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions
			name: "5: Dry-run topic baz version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    bazDiffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions
			name: "6: Apply topic baz version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    bazDiffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic baz version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    bazDiffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic baz version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    bazDiffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic baz version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    bazDiffs[4],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic baz version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    bazDiffs[4],
			wantErr:     "",
			wantApplied: true,
		},
	})

	// Tests rack assignments
	quxDocs := tutil.FileToYAMLDocs(t, "../../test/fixtures/topic/core.operators.topic.applier.qux.yml")
	quxDiffs := getDiffsFixture(t, "../../test/fixtures/topic/core.operators.topic.applier.qux.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic qux version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    quxDiffs[0],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Create topic
			name: "2: Apply topic qux version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[0],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    quxDiffs[0],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Increase replication factor
			name: "3: Dry-run topic qux version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    quxDiffs[1],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Increase replication factor
			name: "4: Apply topic qux version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[1],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    quxDiffs[1],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions
			name: "5: Dry-run topic qux version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    quxDiffs[2],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions
			name: "6: Apply topic qux version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[2],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    quxDiffs[2],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic qux version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    quxDiffs[3],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic qux version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[3],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    quxDiffs[3],
			wantErr:     "",
			wantApplied: true,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic qux version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
					DryRun:           true,
				},
			},
			wantDiff:    quxDiffs[4],
			wantErr:     "",
			wantApplied: false,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic qux version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[4],
				opts: ApplierOptions{
					DefinitionFormat: opt.YAMLFormat,
				},
			},
			wantDiff:    quxDiffs[4],
			wantErr:     "",
			wantApplied: true,
		},
	})
}
