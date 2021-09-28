package topic

import (
	"context"
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
	"github.com/twmb/franz-go/pkg/kgo"
)

// VERBOSE_TESTS=1 go test -run ^Test_applier_Execute$ ./core/operators/topic -v
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
		fixtures.TopicsApplierTest.ComposeFilePaths,
		fixtures.TopicsApplierTest.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := client.New(&client.ClientFlags{
		ConfigPath: "does-not-exist",
		FlagConfigOpts: []string{
			fmt.Sprintf("seedBrokers=localhost:%d", fixtures.TopicsApplierTest.BrokerPort),
		},
	})

	// Wait for Kafka to be ready
	if !service.IsKafkaReady(cl, fixtures.TopicsApplierTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Tests changes to configs
	fooDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/topic/core.operators.topic.applier.foo.yml")
	fooDiffs := getDiffsFixture(t, "../../../test/fixtures/topic/core.operators.topic.applier.foo.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic foo version 0",
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
			// Create topic
			name: "2: Apply topic foo version 0",
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
			name: "3: Dry-run topic foo version 1",
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
			name: "4: Apply topic foo version 1",
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
			name: "5: Dry-run topic foo version 2",
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
			name: "6: Apply topic foo version 2",
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
			name: "7: Dry-run topic foo version 3",
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
			name: "7: Apply topic foo version 3",
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
			name: "7: Dry-run topic foo version 4",
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
			name: "8: Apply topic foo version 4",
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

	// Tests changes to assignments and handling of in-progress reassignments
	barDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/topic/core.operators.topic.applier.bar.yml")
	barDiffs := getDiffsFixture(t, "../../../test/fixtures/topic/core.operators.topic.applier.bar.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic bar version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[0],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                barDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Create topic
			name: "2: Apply topic bar version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[0],
				flags:   ApplierFlags{},
			},
			wantDiff:                barDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
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
		cl.Client().Produce(context.Background(), r, func(r *kgo.Record, err error) {})
	}
	if err := cl.Client().Flush(context.Background()); err != nil {
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
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                barDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Increase replication factor
			name: "4: Apply topic bar version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[1],
				flags:   ApplierFlags{},
			},
			wantDiff:                barDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions
			name: "5: Dry-run topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                barDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions
			name: "6: Apply topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				flags:   ApplierFlags{},
			},
			wantDiff:                barDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                barDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				flags:   ApplierFlags{},
			},
			wantDiff:                barDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                barDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				flags:   ApplierFlags{},
			},
			wantDiff:                barDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})

	// Tests partition and replication factor changes (without static assignments)
	bazDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/topic/core.operators.topic.applier.baz.yml")
	bazDiffs := getDiffsFixture(t, "../../../test/fixtures/topic/core.operators.topic.applier.baz.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic baz version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[0],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                bazDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Create topic
			name: "2: Apply topic baz version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[0],
				flags:   ApplierFlags{},
			},
			wantDiff:                bazDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Increase replication factor
			name: "3: Dry-run topic baz version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[1],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                bazDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Increase replication factor
			name: "4: Apply topic baz version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[1],
				flags:   ApplierFlags{},
			},
			wantDiff:                bazDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions
			name: "5: Dry-run topic baz version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[2],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                bazDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions
			name: "6: Apply topic baz version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[2],
				flags:   ApplierFlags{},
			},
			wantDiff:                bazDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic baz version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[3],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                bazDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic baz version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[3],
				flags:   ApplierFlags{},
			},
			wantDiff:                bazDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic baz version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[4],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                bazDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic baz version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: bazDocs[4],
				flags:   ApplierFlags{},
			},
			wantDiff:                bazDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})

	// Tests rack assignments
	quxDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/topic/core.operators.topic.applier.qux.yml")
	quxDiffs := getDiffsFixture(t, "../../../test/fixtures/topic/core.operators.topic.applier.qux.json")
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Create topic
			name: "1: Dry-run topic qux version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[0],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                quxDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Create topic
			name: "2: Apply topic qux version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[0],
				flags:   ApplierFlags{},
			},
			wantDiff:                quxDiffs[0],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Increase replication factor
			name: "3: Dry-run topic qux version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[1],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                quxDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Increase replication factor
			name: "4: Apply topic qux version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[1],
				flags:   ApplierFlags{},
			},
			wantDiff:                quxDiffs[1],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions
			name: "5: Dry-run topic qux version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[2],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                quxDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions
			name: "6: Apply topic qux version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[2],
				flags:   ApplierFlags{},
			},
			wantDiff:                quxDiffs[2],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions and increase replication factor
			name: "7: Dry-run topic qux version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[3],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                quxDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions and increase replication factor
			name: "8: Apply topic qux version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[3],
				flags:   ApplierFlags{},
			},
			wantDiff:                quxDiffs[3],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Decrease replication factor
			name: "9: Dry-run topic qux version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[4],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantDiff:                quxDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Decrease replication factor
			name: "10: Apply topic qux version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: quxDocs[4],
				flags:   ApplierFlags{},
			},
			wantDiff:                quxDiffs[4],
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})
}
