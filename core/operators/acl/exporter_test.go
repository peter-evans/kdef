//go:build integration
// +build integration

// Package acl implements operators for acl definition operations.
package acl

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/peter-evans/kdef/cli/log"
	"github.com/peter-evans/kdef/core/client"
	"github.com/peter-evans/kdef/core/kafka"
	"github.com/peter-evans/kdef/core/model/opt"
	"github.com/peter-evans/kdef/core/test/compose"
	"github.com/peter-evans/kdef/core/test/harness"
	"github.com/peter-evans/kdef/core/test/tutil"
)

// VERBOSE_TESTS=1 go test --tags=integration -run ^Test_exporter_Execute$ ./core/operators/acl -v
func Test_exporter_Execute(t *testing.T) {
	_, log.Verbose = os.LookupEnv("VERBOSE_TESTS")

	// Create client
	cl := tutil.CreateClient(t,
		[]string{
			fmt.Sprintf("seedBrokers=localhost:%d", harness.ACLExporter.BrokerPort),
			"sasl.method=plain",
			"sasl.user=alice",
			"sasl.pass=alice-secret",
		},
	)

	ctx := context.Background()

	// Create the test cluster
	srv := kafka.NewService(cl)
	maxTries := 3
	try := 1
	for {
		start := time.Now()
		c := compose.Up(
			t,
			harness.ACLExporter.ComposeFilePaths,
			harness.ACLExporter.Env(),
		)
		if srv.IsKafkaReady(ctx, harness.ACLExporter.Brokers, 90) {
			duration := time.Since(start)
			log.Infof("kafka cluster ready in %v", duration)
			defer compose.Down(t, c)
			break
		} else {
			log.Warnf("kafka failed to be ready within timeout")
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
	yamlDocs := tutil.FileToYAMLDocs(t, "../../test/fixtures/acl/core.operators.acl.exporter.yml")

	// Apply the fixtures
	for _, yamlDoc := range yamlDocs {
		applier := NewApplier(cl, yamlDoc, ApplierOptions{
			DefinitionFormat: opt.YAMLFormat,
		})
		res := applier.Execute(ctx)
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
		wantJSON string
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
			wantJSON: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.1.json")),
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
			wantJSON: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.2.json")),
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
			wantJSON: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.3.json")),
			wantErr:  false,
		},
		{
			name: "4: Test export of acl definitions for a specific resource type and with exclude regexp",
			fields: fields{
				cl: cl,
				opts: ExporterOptions{
					Match:        ".*",
					Exclude:      "core.operators.acl.exporter.ba*",
					ResourceType: "topic",
					AutoGroup:    true,
				},
			},
			wantJSON: string(tutil.Fixture(t, "../../test/fixtures/acl/core.operators.acl.exporter.4.json")),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewExporter(tt.fields.cl, tt.fields.opts)
			got, err := e.Execute(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("exporter.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			j, err := got.JSON()
			if err != nil {
				t.Errorf("failed to convert export result to json: %v", err)
				t.FailNow()
			}
			if !tutil.EqualJSON(t, j, tt.wantJSON) {
				t.Errorf("exporter.Execute().JSON() = %v, want %v", j, tt.wantJSON)
			}

			if log.Verbose {
				fmt.Println("[test] ExportResults JSON:")
				fmt.Println(j)
			}
		})
	}
}
