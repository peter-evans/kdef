package brokers

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

// VERBOSE_TESTS=1 go test -run ^Test_exporter_Execute$ ./core/operators/brokers -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create the test cluster
	c := compose.Up(
		t,
		fixtures.BrokersExporterTest.ComposeFilePaths,
		fixtures.BrokersExporterTest.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := client.New(&client.ClientFlags{
		ConfigPath: "does-not-exist",
		FlagConfigOpts: []string{
			fmt.Sprintf("seedBrokers=localhost:%d", fixtures.BrokersExporterTest.BrokerPort),
		},
	})

	// Wait for Kafka to be ready
	if !service.IsKafkaReady(cl, fixtures.BrokersExporterTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../../test/fixtures/brokers/core.operators.brokers.exporter.yml")

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
		cl *client.Client
	}
	tests := []struct {
		name      string
		fields    fields
		wantCount int
		wantErr   bool
	}{
		{
			name: "Test export of brokers definition",
			fields: fields{
				cl: cl,
			},
			wantCount: 1,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExporter(tt.fields.cl)
			got, err := e.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("exporter.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("len(exporter.Execute()) = %v, want %v", len(got), tt.wantCount)
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
