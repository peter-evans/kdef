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

func Test_exporter_Execute(t *testing.T) {
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
	if !req.IsKafkaReady(cl, fixtures.TopicsExporterTest.Brokers, 60) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/topics/test.core.topics.exporter.yml")

	// Apply the topic fixtures
	for _, yamlDoc := range yamlDocs {
		applier := NewApplier(cl, yamlDoc, ApplierFlags{})
		if err := applier.Execute(); err != nil {
			t.Errorf("failed to apply topic fixture: %v", err)
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
		want    int
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
			want:    4,
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
			want:    5,
			wantErr: false,
		},
		{
			name: "Test export of topics with match regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   "test.core.topics.exporter.foo.*",
					Exclude: ".^",
				},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "Test export of topics with exclude regexp",
			fields: fields{
				cl: cl,
				flags: ExporterFlags{
					Match:   ".*",
					Exclude: "test.core.topics.exporter.bar.*",
				},
			},
			want:    2,
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
			if got != tt.want {
				t.Errorf("exporter.Execute() = %v, want %v", got, tt.want)
			}
		})
	}
}
