package database

// MONGO_DEPLOYMENT decides whether multi-document transactions are available.
// Only the explicitly-configured branches are unit testable — the "auto" probe
// needs a live server to ask, so it is exercised by the integration suite.

import "testing"

func TestResolveDeployment_HonoursAnExplicitMode(t *testing.T) {
	cases := []struct {
		configured string
		wantMode   string
		wantTx     bool
	}{
		{DeploymentStandalone, DeploymentStandalone, false},
		{DeploymentReplicaSet, DeploymentReplicaSet, true},
		{DeploymentSharded, DeploymentSharded, true},
		// Operators type what they type. A mode that fails to match must not
		// fall through to the probe with a nil client.
		{"  StandAlone  ", DeploymentStandalone, false},
		{"REPLICASET", DeploymentReplicaSet, true},
	}
	for _, c := range cases {
		// A nil client is safe precisely because an explicit mode returns before
		// the probe — the property being pinned. If someone reorders
		// resolveDeployment so the probe runs first, this panics.
		mode, tx := resolveDeployment(nil, c.configured)
		if mode != c.wantMode || tx != c.wantTx {
			t.Errorf("%q: want (%s, %v), got (%s, %v)", c.configured, c.wantMode, c.wantTx, mode, tx)
		}
	}
}

func TestTransactor_TransactionsEnabledReportsItsMode(t *testing.T) {
	// A caller branches on this to decide whether it must defend itself (e.g.
	// with a compare-and-set), so a Transactor that misreports its mode silently
	// drops that guarantee on a standalone.
	if !NewTransactorWithMode(nil, true).TransactionsEnabled() {
		t.Error("transactions enabled: want true")
	}
	if NewTransactorWithMode(nil, false).TransactionsEnabled() {
		t.Error("transactions disabled: want false")
	}
}

func TestNewTransactor_TakesTheResolvedModeOffMongoDB(t *testing.T) {
	// The resolved value lives on *MongoDB and is read from there — the
	// single-resolver shape, so a caller cannot build a Transactor that
	// disagrees with the server it is connected to.
	if NewTransactor(&MongoDB{Transactions: false}).TransactionsEnabled() {
		t.Error("standalone MongoDB: Transactor must report transactions unavailable")
	}
	if !NewTransactor(&MongoDB{Transactions: true}).TransactionsEnabled() {
		t.Error("replica-set MongoDB: Transactor must report transactions available")
	}
}
