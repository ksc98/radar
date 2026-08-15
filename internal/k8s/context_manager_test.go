package k8s

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConnectionProbeHTTPTimeout(t *testing.T) {
	if got := connectionProbeHTTPTimeout(context.Background()); got != ConnectionTestTimeout {
		t.Fatalf("connectionProbeHTTPTimeout(no deadline) = %v, want %v", got, ConnectionTestTimeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ConnectionTestTimeout)
	defer cancel()
	got := connectionProbeHTTPTimeout(ctx)
	if got <= 0 || got >= ConnectionTestTimeout {
		t.Fatalf("connectionProbeHTTPTimeout(standard deadline) = %v, want positive timeout shorter than %v", got, ConnectionTestTimeout)
	}
	if got < ConnectionTestTimeout-2*connectionProbeTimeoutHeadroom {
		t.Fatalf("connectionProbeHTTPTimeout(standard deadline) = %v, want close to deadline with headroom", got)
	}

	shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shortCancel()
	got = connectionProbeHTTPTimeout(shortCtx)
	if got <= 0 || got > connectionProbeTimeoutHeadroom {
		t.Fatalf("connectionProbeHTTPTimeout(short deadline) = %v, want the remaining short deadline", got)
	}
}

func TestConnectionTestOperationTimeoutUsesLongerExecAuthBudget(t *testing.T) {
	withContextExecAuth(t, false)
	if got := connectionTestOperationTimeout(); got != ConnectionTestTimeout {
		t.Fatalf("connectionTestOperationTimeout(no exec auth) = %v, want %v", got, ConnectionTestTimeout)
	}

	withContextExecAuth(t, true)
	want := execAuthConnectionProbeHTTPTimeout + connectionProbeTimeoutHeadroom
	if got := connectionTestOperationTimeout(); got != want {
		t.Fatalf("connectionTestOperationTimeout(exec auth) = %v, want %v", got, want)
	}
}

// In-cluster, a switch to a different context must be rejected in preflight —
// before any teardown. The pre-fix ordering tore down every subsystem and only
// then hit SwitchContext's in-cluster guard, leaving the app dead until a pod
// restart (reachable in production via POST /api/connection/retry and the
// runtime-auth recovery worker).
func TestPerformContextSwitchInClusterPreflight(t *testing.T) {
	ForceInCluster = true
	t.Cleanup(func() { ForceInCluster = false })

	err := PerformContextSwitch("some-other-context")
	if err == nil {
		t.Fatal("expected error switching context in-cluster")
	}
	if !errors.Is(err, ErrContextSwitchPreflight) {
		t.Fatalf("want ErrContextSwitchPreflight (nothing torn down), got: %v", err)
	}
}
