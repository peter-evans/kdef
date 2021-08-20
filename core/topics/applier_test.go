package topics

import (
	"fmt"
	"testing"
	"time"

	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/req"
	"github.com/peter-evans/kdef/test/compose"
	"github.com/peter-evans/kdef/test/fixtures"
	"github.com/peter-evans/kdef/test/tutil"
)

func Test_applier_Execute(t *testing.T) {
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
	if !req.IsKafkaReady(cl, fixtures.TopicsApplierTest.Brokers, 60) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/topics/test.core.topics.applier.foo.yml")

	type fields struct {
		cl      *client.Client
		yamlDoc string
		flags   ApplierFlags
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr string
	}{
		// NOTE: Execution of tests is ordered
		{
			name: "1: Dry-run topic version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[0],
				flags: ApplierFlags{
					DryRun:   true,
					ExitCode: true,
				},
			},
			wantErr: "unapplied changes exist for topic",
		},
		{
			name: "2: Apply topic version 0",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[0],
				flags:   ApplierFlags{},
			},
			wantErr: "",
		},
		{
			// Update configs
			name: "3: Dry-run topic version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[1],
				flags: ApplierFlags{
					DryRun:   true,
					ExitCode: true,
				},
			},
			wantErr: "unapplied changes exist for topic",
		},
		{
			// Update configs
			name: "4: Apply topic version 1",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[1],
				flags:   ApplierFlags{},
			},
			wantErr: "",
		},
		{
			// Delete configs
			name: "5: Dry-run topic version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[2],
				flags: ApplierFlags{
					DryRun:               true,
					ExitCode:             true,
					DeleteMissingConfigs: true,
				},
			},
			wantErr: "unapplied changes exist for topic",
		},
		{
			// Delete configs
			name: "6: Apply topic version 2",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[2],
				flags: ApplierFlags{
					DeleteMissingConfigs: true,
				},
			},
			wantErr: "",
		},
		{
			// Update configs (non-incremental)
			name: "7: Dry-run topic version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[3],
				flags: ApplierFlags{
					DryRun:         true,
					ExitCode:       true,
					NonIncremental: true,
				},
			},
			wantErr: "unapplied changes exist for topic",
		},
		{
			// Update configs (non-incremental)
			name: "7: Apply topic version 3",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[3],
				flags: ApplierFlags{
					NonIncremental: true,
				},
			},
			wantErr: "",
		},
		{
			// Delete configs (non-incremental)
			name: "7: Dry-run topic version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[4],
				flags: ApplierFlags{
					DryRun:         true,
					ExitCode:       true,
					NonIncremental: true,
				},
			},
			wantErr: "cannot apply operations containing deletions",
		},
		{
			// Delete configs (non-incremental)
			name: "8: Apply topic version 4",
			fields: fields{
				cl:      cl,
				yamlDoc: yamlDocs[4],
				flags: ApplierFlags{
					NonIncremental:       true,
					DeleteMissingConfigs: true,
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewApplier(tt.fields.cl, tt.fields.yamlDoc, tt.fields.flags)
			if err := a.Execute(); !tutil.ErrorContains(err, tt.wantErr) {
				t.Errorf("applier.Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Sleep to give Kafka time to update internally
			time.Sleep(2 * time.Second)
		})
	}
}
