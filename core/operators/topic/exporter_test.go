package topic

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/client"
	"github.com/peter-evans/kdef/core/service"
	"github.com/peter-evans/kdef/test/compose"
	"github.com/peter-evans/kdef/test/fixtures"
	"github.com/peter-evans/kdef/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_exporter_Execute$ ./core/operators/topic -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create the test cluster
	c := compose.Up(
		t,
		fixtures.TopicsExporterTest.ComposeFilePaths,
		fixtures.TopicsExporterTest.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := client.New(&client.ClientFlags{
		ConfigPath: "does-not-exist",
		FlagConfigOpts: []string{
			fmt.Sprintf("seedBrokers=localhost:%d", fixtures.TopicsExporterTest.BrokerPort),
		},
	})

	// Wait for Kafka to be ready
	if !service.IsKafkaReady(cl, fixtures.TopicsExporterTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/topic/core.operators.topic.exporter.yml")

	// Apply the fixtures
	for _, yamlDoc := range yamlDocs {
		applier := NewApplier(cl, yamlDoc, ApplierFlags{})
		res := applier.Execute()
		if err := res.GetErr(); err != nil {
			t.Errorf("failed to apply fixture: %v", err)
			t.FailNow()
		}
	}

	// Sleep to give Kafka time to update internally
	time.Sleep(2 * time.Second)

	type fields struct {
		cl    *client.Client
		flags ExporterFlags
	}
	tests := []struct {
		name    string
		fields  fields
		wantIds []string
		wantErr bool
	}{
		{
			name: "Test export of all topics",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   ".*",
					Exclude: ".^",
				},
			},
			wantIds: []string{
				"core.operators.topic.exporter.bar1",
				"core.operators.topic.exporter.bar2",
				"core.operators.topic.exporter.foo1",
				"core.operators.topic.exporter.foo2",
			},
			wantErr: false,
		},
		{
			name: "Test export of all topics including internal",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:           ".*",
					Exclude:         ".^",
					IncludeInternal: true,
				},
			},
			wantIds: []string{
				"__core.operators.topic.exporter.baz1",
				"core.operators.topic.exporter.bar1",
				"core.operators.topic.exporter.bar2",
				"core.operators.topic.exporter.foo1",
				"core.operators.topic.exporter.foo2",
			},
			wantErr: false,
		},
		{
			name: "Test export of topics with match regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   "core.operators.topic.exporter.foo.*",
					Exclude: ".^",
				},
			},
			wantIds: []string{
				"core.operators.topic.exporter.foo1",
				"core.operators.topic.exporter.foo2",
			},
			wantErr: false,
		},
		{
			name: "Test export of topics with exclude regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   ".*",
					Exclude: "core.operators.topic.exporter.bar.*",
				},
			},
			wantIds: []string{
				"core.operators.topic.exporter.foo1",
				"core.operators.topic.exporter.foo2",
			},
			wantErr: false,
		},
		{
			name: "Test export of topics including broker assignments",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: "broker",
				},
			},
			wantIds: []string{
				"core.operators.topic.exporter.foo1",
			},
			wantErr: false,
		},
		{
			name: "Test export of topics including rack assignments",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: "rack",
				},
			},
			wantIds: []string{
				"core.operators.topic.exporter.foo1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExporter(tt.fields.cl, tt.fields.flags)
			got, err := e.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("exporter.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.Ids(), tt.wantIds) {
				t.Errorf("exporter.Execute().Ids() = %v, want %v", got.Ids(), tt.wantIds)
			}

			if log.Verbose {
				j, err := got.JSON()
				if err != nil {
					t.Errorf("failed to convert export result to json: %v", err)
					t.FailNow()
				}
				fmt.Println("[test] ExportResults JSON:")
				fmt.Println(j)
			}
		})
	}
}
