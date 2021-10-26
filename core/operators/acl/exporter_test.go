package acl

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/test/compose"
	"github.com/peter-evans/kdef/core/test/compose_fixture"
	"github.com/peter-evans/kdef/core/test/tutil"
)

// VERBOSE_TESTS=1 go test -run ^Test_exporter_Execute$ ./core/operators/acl -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create the test cluster
	c := compose.Up(
		t,
		compose_fixture.AclExporterComposeFixture.ComposeFilePaths,
		compose_fixture.AclExporterComposeFixture.Env(),
	)
	defer compose.Down(t, c)

	// Create client
	cl := tutil.CreateClient(t,
		[]string{
			fmt.Sprintf("seedBrokers=localhost:%d", compose_fixture.AclExporterComposeFixture.BrokerPort),
			"sasl.method=plain",
			"sasl.user=alice",
			"sasl.pass=alice-secret",
		},
	)

	// Wait for Kafka to be ready
	srv := kafka.NewService(cl)
	if !srv.IsKafkaReady(compose_fixture.AclExporterComposeFixture.Brokers, 90) {
		t.Errorf("kafka failed to be ready within timeout")
		t.FailNow()
	}

	// Load YAML doc test fixtures
	yamlDocs := tutil.FileToYamlDocs(t, "../../test/fixtures/acl/core.operators.acl.exporter.yml")

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
			name: "1: Test export of acl definitions for all resource types",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:        ".*",
					Exclude:      ".^",
					ResourceType: "any",
					AutoGroup:    true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.1.json")),
			wantErr:  false,
		},
		{
			name: "2: Test export of acl definitions for a specific resource type",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:        ".*",
					Exclude:      ".^",
					ResourceType: "topic",
					AutoGroup:    true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.2.json")),
			wantErr:  false,
		},
		{
			name: "3: Test export of acl definitions with match regexp",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:        "core.operators.acl.exporter.f.*",
					Exclude:      ".^",
					ResourceType: "any",
					AutoGroup:    true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.3.json")),
			wantErr:  false,
		},
		{
			name: "4: Test export of acl definitions for a specific resource type and with exclude regexp",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:        ".*",
					Exclude:      "core.operators.acl.exporter.bar",
					ResourceType: "topic",
					AutoGroup:    true,
				},
			},
			wantJson: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.4.json")),
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
