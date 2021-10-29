// +build integration

package topic

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/test/compose"
	"github.com/peter-evans/kdef/core/test/compose_fixture"
	"github.com/peter-evans/kdef/core/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_exporter_Execute$ ./core/operators/topic -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create client
	cl := tutil.CreateClient(t,
		[]string{fmt.Sprintf("seedBrokers=localhost:%d", compose_fixture.TopicsExporterComposeFixture.BrokerPort)},
	)

	// Create the test cluster
	srv := kafka.NewService(cl)
	maxTries := 3
	try := 1
	for {
		start := time.Now()
		c := compose.Up(
			t,
			compose_fixture.TopicsExporterComposeFixture.ComposeFilePaths,
			compose_fixture.TopicsExporterComposeFixture.Env(),
		)
		if srv.IsKafkaReady(compose_fixture.TopicsExporterComposeFixture.Brokers, 90) {
			duration := time.Since(start)
			log.Info("kafka cluster ready in %v", duration)
			defer compose.Down(t, c)
			break
		} else {
			log.Warn("kafka failed to be ready within timeout")
			compose.Down(t, c)
			try++
		}
		if try > maxTries {
			t.Errorf("kafka failed to be ready within timeout after %d tries", maxTries)
			t.FailNow()
		}
		time.Sleep(2 * time.Second)
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/topic/core.operators.topic.exporter.yml")

	// Apply the fixtures
	for _, yamlDoc := range yamlDocs {
		applier := NewApplier(cl, yamlDoc, ApplierOptions{})
		res := applier.Execute()
		if err := res.GetErr(); err != nil {
			t.Errorf("failed to apply fixture: %v", err)
			t.FailNow()
		}
	}

	// Sleep to give Kafka time to update internally
	time.Sleep(2 * time.Second)

	type fields struct {
		cl   *client.Client
		opts ExporterOptions
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
				opts: ExporterOptions{
					Match:   ".*",
					Exclude: ".^",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.1.json")),
			wantErr:  false,
		},
		{
			name: "2: Test export of all topics including internal",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:           ".*",
					Exclude:         ".^",
					IncludeInternal: true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.2.json")),
			wantErr:  false,
		},
		{
			name: "3: Test export of topics with match regexp",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:   "core.operators.topic.exporter.foo.*",
					Exclude: ".^",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.3.json")),
			wantErr:  false,
		},
		{
			name: "4: Test export of topics with exclude regexp",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:   ".*",
					Exclude: "core.operators.topic.exporter.bar.*",
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.4.json")),
			wantErr:  false,
		},
		{
			name: "5: Test export of topics including broker assignments",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: opt.BrokerAssignments,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.5.json")),
			wantErr:  false,
		},
		{
			name: "6: Test export of topics including rack assignments",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:       "core.operators.topic.exporter.foo1",
					Exclude:     ".^",
					Assignments: opt.RackAssignments,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/topic/core.operators.topic.exporter.6.json")),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExporter(tt.fields.cl, tt.fields.opts)
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
