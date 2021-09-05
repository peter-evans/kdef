package topics

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/req"
	"github.com/peter-evans/kdef/test/compose"
	"github.com/peter-evans/kdef/test/fixtures"
	"github.com/peter-evans/kdef/test/tutil"
	"github.com/twmb/franz-go/pkg/kgo"
)

// go test -run ^Test_applier_Execute$ ./core/topics -v
func Test_applier_Execute(t *testing.T) {
	// log.Verbose = true

	type fields struct {
		cl      *client.Client
		yamlDoc string
		flags   ApplierFlags
	}
	type testCase struct {
		name                    string
		fields                  fields
		wantErr                 string
		wantHasUnappliedChanges bool
	}

	runTests := func(t *testing.T, tests []testCase) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				a := NewApplier(tt.fields.cl, tt.fields.yamlDoc, tt.fields.flags)
				result := a.Execute()

				if log.Verbose {
					// Output apply result JSON
					jsonOut, err := json.MarshalIndent(result, "", "  ")
					if err != nil {
						t.Errorf("failed to convert apply result to json: %v", err)
						t.FailNow()
					}
					fmt.Println(string(jsonOut))
				}

				if !tutil.ErrorContains(result.GetErr(), tt.wantErr) {
					t.Errorf("applier.Execute() error = %v, wantErr %v", result.GetErr(), tt.wantErr)
				}
				if result.HasUnappliedChanges() != tt.wantHasUnappliedChanges {
					t.Errorf("exporter.Execute().HasUnappliedChanges() = %v, want %v",
						result.HasUnappliedChanges(),
						tt.wantHasUnappliedChanges,
					)
				}

				// Sleep to give Kafka time to update internally
				time.Sleep(2 * time.Second)
			})
		}
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
	if !req.IsKafkaReady(cl, fixtures.TopicsApplierTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	fooDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/topics/test.core.topics.applier.foo.yml")
	barDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/topics/test.core.topics.applier.bar.yml")

	// Tests configs and addition of partitions
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
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions
			name: "9: Dry-run topic foo version 5",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[5],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions
			name: "10: Apply topic foo version 5",
			fields: fields{
				cl:      cl,
				yamlDoc: fooDocs[5],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})

	// Tests assignments and reassignment cases
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
			wantErr:                 "invalid broker id",
			wantHasUnappliedChanges: false,
		},
		{
			// Create topic
			name: "2: Dry-run topic bar version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[1],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Create topic
			name: "3: Apply topic bar version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[1],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})

	// Produce records into topic before proceeding with remaining test cases
	topic := "test.core.topics.applier.bar"
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

	// Tests assignments and reassignment cases (continued)
	runTests(t, []testCase{
		// NOTE: Execution of tests is ordered
		{
			// Increase replication factor
			name: "4: Dry-run topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Increase replication factor
			name: "5: Apply topic bar version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[2],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions
			name: "7: Dry-run topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions
			name: "8: Apply topic bar version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[3],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Add partitions and increase replication factor
			name: "9: Dry-run topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Add partitions and increase replication factor
			name: "10: Apply topic bar version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[4],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
		{
			// Decrease replication factor
			name: "11: Dry-run topic bar version 5",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[5],
				flags: ApplierFlags{
					DryRun: true,
				},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: true,
		},
		{
			// Decrease replication factor
			name: "12: Apply topic bar version 5",
			fields: fields{
				cl:      cl,
				yamlDoc: barDocs[5],
				flags:   ApplierFlags{},
			},
			wantErr:                 "",
			wantHasUnappliedChanges: false,
		},
	})
}
