package topic

import (
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
		name     string
		fields   fields
		wantJson string
		wantErr  bool
	}{
		{
			name: "1: Test export of all topics",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   ".*",
					Exclude: ".^",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.1.json")),
			wantErr:  false,
		},
		{
			name: "2: Test export of all topics including internal",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:           ".*",
					Exclude:         ".^",
					IncludeInternal: true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.2.json")),
			wantErr:  false,
		},
		{
			name: "3: Test export of topics with match regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   "core.operators.topic.exporter.foo.*",
					Exclude: ".^",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.3.json")),
			wantErr:  false,
		},
		{
			name: "4: Test export of topics with exclude regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   ".*",
					Exclude: "core.operators.topic.exporter.bar.*",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.4.json")),
			wantErr:  false,
		},
		{
			name: "5: Test export of topics including broker assignments",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: "broker",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.5.json")),
			wantErr:  false,
		},
		{
			name: "6: Test export of topics including rack assignments",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: "rack",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/topic/core.operators.topic.exporter.6.json")),
			wantErr:  false,
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
			j, err := got.JSON()
			if err != nil {
				t.Errorf("failed to convert export result to json: %v", err)
				t.FailNow()
			}
			if !tutil.EqualJSON(t, j, tt.wantJson) {
				t.Errorf("exporter.Execute().JSON() = %v, want %v", j, tt.wantJson)
			}

			if log.Verbose {
				fmt.Println("[test] ExportResults JSON:")
				fmt.Println(j)
			}
		})
	}
}
