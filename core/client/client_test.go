package client

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestBuildSASLOpt_AWSMSKIam(t *testing.T) {
	// Test that AWS MSK IAM SASL option can be built without errors
	// This test verifies that the new AWS SDK v2 code path compiles and runs
	// without making actual AWS API calls

	config := &Config{
		SASL: &saslConfig{
			Method: "awsmskiam",
		},
	}

	client := &Client{
		cc:      config,
		kgoOpts: []kgo.Opt{},
	}

	// This should not panic or return an error during normal operation
	// If AWS credentials are not available, it should fail only when actually
	// attempting to retrieve credentials, not during setup
	err := client.buildSASLOpt()
	// The function should not return an error during setup phase
	// Error will only occur during actual credential retrieval at runtime
	if err != nil {
		t.Errorf("buildSASLOpt() returned error during setup: %v", err)
	}
}

func TestBuildSASLOpt_InvalidMethod(t *testing.T) {
	config := &Config{
		SASL: &saslConfig{
			Method: "invalid-method",
		},
	}

	client := &Client{
		cc:      config,
		kgoOpts: []kgo.Opt{},
	}

	err := client.buildSASLOpt()
	if err == nil {
		t.Error("buildSASLOpt() should return error for invalid SASL method")
	}
}
