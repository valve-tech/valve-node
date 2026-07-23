package monitor

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node/internal/catalog"
	"github.com/valve-tech/valve-node/internal/executor"
)

// fakeExecutor mirrors internal/setup's test double: a scripted
// map[string]executor.Result keyed by command substring (longest match
// wins), with every Run call recorded in order.
type fakeExecutor struct {
	mu      sync.Mutex
	scripts map[string]executor.Result
	errs    map[string]error
	calls   []string
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{scripts: map[string]executor.Result{}, errs: map[string]error{}}
}

func (f *fakeExecutor) script(substr string, res executor.Result) *fakeExecutor {
	f.scripts[substr] = res
	return f
}

func (f *fakeExecutor) errOn(substr string, err error) *fakeExecutor {
	f.errs[substr] = err
	return f
}

func (f *fakeExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.mu.Unlock()

	if key, err := f.matchErr(cmd); key != "" {
		return executor.Result{}, err
	}
	if key, res := f.matchScript(cmd); key != "" {
		return res, nil
	}
	return executor.Result{ExitCode: 0}, nil
}

func (f *fakeExecutor) matchScript(cmd string) (string, executor.Result) {
	keys := make([]string, 0, len(f.scripts))
	for k := range f.scripts {
		if strings.Contains(cmd, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return "", executor.Result{}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return keys[0], f.scripts[keys[0]]
}

func (f *fakeExecutor) matchErr(cmd string) (string, error) {
	keys := make([]string, 0, len(f.errs))
	for k := range f.errs {
		if strings.Contains(cmd, k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return "", nil
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return keys[0], f.errs[keys[0]]
}

func (f *fakeExecutor) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	return nil
}
func (f *fakeExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) { return nil, nil }
func (f *fakeExecutor) Close() error                                              { return nil }

// ---------------------------------------------------------------------
// poll: field parsing
// ---------------------------------------------------------------------

func TestPollParsesSnapshotFields(t *testing.T) {
	fe := newFakeExecutor().
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("eth_blockNumber", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x1234"}`}).
		script("net_peerCount", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0xa"}`}).
		script("/eth/v1/node/syncing", executor.Result{Stdout: `{"data":{"head_slot":"1000","sync_distance":"5","is_syncing":true,"is_optimistic":false,"el_offline":false}}`}).
		script("/eth/v1/node/peer_count", executor.Result{Stdout: `{"data":{"disconnected":"1","connecting":"0","connected":"7","disconnecting":"0"}}`}).
		script("df --output=pcent", executor.Result{Stdout: "Use%\n 42%\n"}).
		script("systemctl is-active", executor.Result{Stdout: "active\nactive\n"})

	m := New(Config{Exec: fe, Wire: catalog.WireConfig{ChainID: 369, DataDir: "/var/lib/valve-node/369"}})
	snap := m.poll(context.Background())

	if snap.ExecSyncing {
		t.Errorf("ExecSyncing = true, want false")
	}
	if snap.ExecHead != 0x1234 {
		t.Errorf("ExecHead = %#x, want 0x1234", snap.ExecHead)
	}
	if snap.ExecPeers != 10 {
		t.Errorf("ExecPeers = %d, want 10", snap.ExecPeers)
	}
	if snap.BeaconSlot != 1000 {
		t.Errorf("BeaconSlot = %d, want 1000", snap.BeaconSlot)
	}
	if snap.BeaconDistance != 5 {
		t.Errorf("BeaconDistance = %d, want 5", snap.BeaconDistance)
	}
	if snap.BeaconPeers != 7 {
		t.Errorf("BeaconPeers = %d, want 7", snap.BeaconPeers)
	}
	if snap.DiskUsedPct != 42 {
		t.Errorf("DiskUsedPct = %v, want 42", snap.DiskUsedPct)
	}
	if !snap.ExecActive || !snap.BeaconActive {
		t.Errorf("ExecActive=%v BeaconActive=%v, want true/true", snap.ExecActive, snap.BeaconActive)
	}
	if snap.At.IsZero() {
		t.Errorf("At is zero, want set")
	}
}

// TestPollUsesWireConfigCustomPorts locks in that every exec/beacon probe
// resolves its target address via WireConfig.ExecHTTP()/BeaconHTTP()
// (default 8545/5052 on the zero value) rather than a hardcoded constant —
// a target configured with custom ports must be probed on those exact
// ports, and the old default ports must never be touched.
func TestPollUsesWireConfigCustomPorts(t *testing.T) {
	fe := newFakeExecutor().
		script("http://127.0.0.1:9545", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x2a"}`}).
		script("http://127.0.0.1:6052/eth/v1/node/syncing", executor.Result{
			Stdout: `{"data":{"head_slot":"7","sync_distance":"0","is_syncing":false,"is_optimistic":false,"el_offline":false}}`,
		})

	m := New(Config{Exec: fe, Wire: catalog.WireConfig{ExecHTTPPort: 9545, BeaconHTTPPort: 6052}})
	snap := m.poll(context.Background())

	if snap.ExecHead != 0x2a {
		t.Errorf("ExecHead = %#x, want 0x2a (probe should have hit the configured custom exec port)", snap.ExecHead)
	}
	if snap.BeaconSlot != 7 {
		t.Errorf("BeaconSlot = %d, want 7 (probe should have hit the configured custom beacon port)", snap.BeaconSlot)
	}
	for _, c := range fe.calls {
		if strings.Contains(c, "127.0.0.1:8545") || strings.Contains(c, "127.0.0.1:5052") {
			t.Fatalf("probe used a default port instead of the configured custom port: %q", c)
		}
	}
}

func TestPollExecSyncingObjectMeansSyncingTrue(t *testing.T) {
	fe := newFakeExecutor().
		script("eth_syncing", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":{"startingBlock":"0x0","currentBlock":"0x64","highestBlock":"0xc8"}}`})
	m := New(Config{Exec: fe})
	snap := m.poll(context.Background())
	if !snap.ExecSyncing {
		t.Errorf("ExecSyncing = false, want true")
	}
}

func TestPollServiceInactiveReportsFalse(t *testing.T) {
	fe := newFakeExecutor().
		script("systemctl is-active", executor.Result{ExitCode: 3, Stdout: "active\ninactive\n"})
	m := New(Config{Exec: fe})
	snap := m.poll(context.Background())
	if !snap.ExecActive {
		t.Errorf("ExecActive = false, want true")
	}
	if snap.BeaconActive {
		t.Errorf("BeaconActive = true, want false")
	}
}

// ---------------------------------------------------------------------
// poll: failure semantics — a failed probe never errors the whole poll
// ---------------------------------------------------------------------

func TestPollToleratesTransportAndParseFailures(t *testing.T) {
	fe := newFakeExecutor().
		errOn("eth_syncing", fmt.Errorf("boom: ssh transport dropped")).
		script("eth_blockNumber", executor.Result{Stdout: "not json"}).
		script("net_peerCount", executor.Result{ExitCode: 1, Stdout: `{"jsonrpc":"2.0","id":1,"result":"0xa"}`}).
		script("/eth/v1/node/syncing", executor.Result{Stdout: "not json either"}).
		script("df --output=pcent", executor.Result{ExitCode: 1, Stdout: ""})

	m := New(Config{Exec: fe})

	var snap Snapshot
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("poll panicked on a failed probe: %v", r)
			}
		}()
		snap = m.poll(context.Background())
	}()

	if snap.ExecSyncing {
		t.Errorf("ExecSyncing = true, want false (zero value) after transport error")
	}
	if snap.ExecHead != 0 {
		t.Errorf("ExecHead = %d, want 0 after unparseable JSON", snap.ExecHead)
	}
	if snap.ExecPeers != 0 {
		t.Errorf("ExecPeers = %d, want 0 after non-zero exit", snap.ExecPeers)
	}
	if snap.BeaconSlot != 0 || snap.BeaconDistance != 0 {
		t.Errorf("BeaconSlot=%d BeaconDistance=%d, want 0/0 after unparseable JSON", snap.BeaconSlot, snap.BeaconDistance)
	}
	if snap.DiskUsedPct != 0 {
		t.Errorf("DiskUsedPct = %v, want 0 after non-zero exit", snap.DiskUsedPct)
	}
}

// ---------------------------------------------------------------------
// poll: RefRPC
// ---------------------------------------------------------------------

func TestPollRefHeadFromRefRPC(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x5678"}`))
	}))
	defer ts.Close()

	m := New(Config{Exec: newFakeExecutor(), RefRPC: ts.URL})
	snap := m.poll(context.Background())
	if snap.RefHead != 0x5678 {
		t.Errorf("RefHead = %#x, want 0x5678", snap.RefHead)
	}
}

func TestPollRefHeadZeroWhenRefRPCDown(t *testing.T) {
	// Port 1 is privileged and refuses connections on a normal test box.
	m := New(Config{Exec: newFakeExecutor(), RefRPC: "http://127.0.0.1:1"})
	snap := m.poll(context.Background())
	if snap.RefHead != 0 {
		t.Errorf("RefHead = %d, want 0 when RefRPC is unreachable", snap.RefHead)
	}
}

func TestPollRefHeadZeroWhenRefRPCEmpty(t *testing.T) {
	m := New(Config{Exec: newFakeExecutor(), RefRPC: ""})
	snap := m.poll(context.Background())
	if snap.RefHead != 0 {
		t.Errorf("RefHead = %d, want 0 when RefRPC is unset", snap.RefHead)
	}
}

func TestPollRefHeadZeroOnNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	m := New(Config{Exec: newFakeExecutor(), RefRPC: ts.URL})
	snap := m.poll(context.Background())
	if snap.RefHead != 0 {
		t.Errorf("RefHead = %d, want 0 on a 500 response", snap.RefHead)
	}
}

// ---------------------------------------------------------------------
// Subscribe / Start / Latest
// ---------------------------------------------------------------------

func TestSubscribeDeliversEachTick(t *testing.T) {
	m := New(Config{Exec: newFakeExecutor(), Interval: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsub := m.Subscribe()
	defer unsub()

	m.Start(ctx)

	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("tick %d: timed out waiting for a snapshot", i)
		}
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	m := New(Config{Exec: newFakeExecutor(), Interval: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsub := m.Subscribe()
	m.Start(ctx)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first snapshot")
	}

	unsub()

	// Drain anything already in flight at the moment unsub() ran.
	select {
	case <-ch:
	case <-time.After(50 * time.Millisecond):
	}

	select {
	case v, ok := <-ch:
		if ok {
			t.Fatalf("received a snapshot after unsubscribe: %+v", v)
		}
	case <-time.After(200 * time.Millisecond):
		// No further delivery — correct.
	}
}

func TestLatestReflectsMostRecentPoll(t *testing.T) {
	m := New(Config{Exec: newFakeExecutor(), Interval: 10 * time.Millisecond})
	if !m.Latest().At.IsZero() {
		t.Fatalf("Latest() before Start: want zero value, got %+v", m.Latest())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsub := m.Subscribe()
	defer unsub()
	m.Start(ctx)

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first snapshot")
	}

	if m.Latest().At.IsZero() {
		t.Errorf("Latest() after a poll: want non-zero At, got zero value")
	}
}
