package ops

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/valve-tech/valve-node/internal/executor"
)

// fakeExecutor mirrors internal/setup's and internal/monitor's test double:
// a scripted map[string]executor.Result keyed by command substring (longest
// match wins), with every Run call recorded in order for assertions.
type fakeExecutor struct {
	mu sync.Mutex

	scripts map[string]executor.Result
	errs    map[string]error

	calls []string
	files map[string][]byte
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{
		scripts: map[string]executor.Result{},
		errs:    map[string]error{},
		files:   map[string][]byte{},
	}
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
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, fmt.Sprintf("WriteFile %s", path))
	f.files[path] = content
	return nil
}

func (f *fakeExecutor) ReadFile(ctx context.Context, path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("fakeExecutor: no such file %q", path)
	}
	return b, nil
}

func (f *fakeExecutor) Close() error { return nil }

func (f *fakeExecutor) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}
