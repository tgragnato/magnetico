package opflags

import (
	"testing"
)

func TestCheckAddrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		indexerAddrs []string
		expectError  bool
	}{
		{
			name:         "EmptyIndexerAddrs",
			indexerAddrs: []string{},
			expectError:  true,
		},
		{
			name:         "SingleEmptyIndexerAddr",
			indexerAddrs: []string{""},
			expectError:  true,
		},
		{
			name:         "ValidIndexerAddrs",
			indexerAddrs: []string{"127.0.0.1:6881", "192.168.1.1:6881"},
			expectError:  false,
		},
		{
			name:         "InvalidIndexerAddr",
			indexerAddrs: []string{"invalid-addr"},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opFlags := &OpFlags{
				IndexerAddrs: tt.indexerAddrs,
			}
			err := opFlags.checkAddrs()
			if (err != nil) != tt.expectError {
				t.Errorf("checkAddrs() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		opFlags           OpFlags
		expectError       bool
		expectPrintOutput bool
	}{
		{
			name: "RunWebWithValidCred",
			opFlags: OpFlags{
				RunWeb: true,
			},
			expectError: false,
		},
		{
			name: "RunWebWithInvalidCred",
			opFlags: OpFlags{
				RunWeb: true,
				Cred:   "invalid-cred",
			},
			expectError: true,
		},
		{
			name: "RunDaemonWithValidAddrs",
			opFlags: OpFlags{
				RunDaemon:    true,
				IndexerAddrs: []string{"127.0.0.1:6881"},
			},
			expectError: false,
		},
		{
			name: "RunDaemonWithInvalidAddrs",
			opFlags: OpFlags{
				RunDaemon:    true,
				IndexerAddrs: []string{"invalid-addr"},
			},
			expectError: true,
		},
		{
			name: "RunDaemonWithInvalidCIDR",
			opFlags: OpFlags{
				RunDaemon:        true,
				FilterNodesCIDRs: []string{"invalid-cidr"},
				IndexerAddrs:     []string{"0.0.0.0:0"},
			},
			expectError: true,
		},
		{
			name: "RunDaemonWithValidCIDR",
			opFlags: OpFlags{
				IndexerAddrs:     []string{"0.0.0.0:0"},
				RunDaemon:        true,
				FilterNodesCIDRs: []string{"192.168.1.0/24"},
			},
			expectError: false,
		},
		{
			name: "RunDaemonWithDefaultBootstrappingNodesInFilterMode",
			opFlags: OpFlags{
				RunDaemon:          true,
				FilterNodesCIDRs:   []string{"192.168.1.0/24"},
				BootstrappingNodes: []string{"dht.tgragnato.it:80", "dht.tgragnato.it:443", "dht.tgragnato.it:1337", "dht.tgragnato.it:6969", "dht.tgragnato.it:6881", "dht.tgragnato.it:25401"},
				IndexerAddrs:       []string{"0.0.0.0:0"},
			},
			expectError: true,
		},
		{
			name: "RunDaemonWithCustomBootstrappingNodesInFilterMode",
			opFlags: OpFlags{
				RunDaemon:          true,
				FilterNodesCIDRs:   []string{"", "192.168.1.0/24"},
				BootstrappingNodes: []string{"", "www.tgragnato.it:443"},
				IndexerAddrs:       []string{"0.0.0.0:0"},
			},
			expectError: false,
		},
		{
			name: "RunWithBothDaemonAndWebStopped",
			opFlags: OpFlags{
				RunDaemon:    false,
				RunWeb:       false,
				IndexerAddrs: []string{"0.0.0.0:0"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opFlags.check()
			if (err != nil) != tt.expectError {
				t.Errorf("check() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
