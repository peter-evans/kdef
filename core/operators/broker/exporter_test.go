package broker

import (
	"fmt"
	"os"
	"testing"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/test/compose"
	"github.com/peter-evans/kdef/test/fixtures"
	"github.com/peter-evans/kdef/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_exporter_Execute$ ./core/operators/broker -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create the test cluster
	c := compose.Up(
		t,
		fixtures.BrokerExporterTest.ComposeFilePaths,
		fixtures.BrokerExporterTest.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := tutil.CreateClient(t,
		[]string{fmt.Sprintf("seedBrokers=localhost:%d", fixtures.BrokerExporterTest.BrokerPort)},
	)

	// Wait for Kafka to be ready
	srv := kafka.NewService(cl)
	if !srv.IsKafkaReady(fixtures.BrokerExporterTest.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	type fields struct {
		cl *client.Client
	}
	tests := []struct {
		name     string
		fields   fields
		wantJson string
		wantErr  bool
	}{
		{
			name: "1: Test export of broker definitions",
			fields: fields{
				cl: cl,
			},
			wantJson: string(tutil.Fixture(t, "../../../test/fixtures/broker/core.operators.broker.exporter.1.json")),
			wantErr:  false,
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
