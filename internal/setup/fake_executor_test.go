package setup

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"github.com/valve-tech/valve-node/internal/executor"
)

// fakeExecutor is the shared test double for this package: a scripted
// map[string]executor.Result keyed by command substring (first match wins,
// longest substring key checked first so more specific scripts beat generic
// ones), with every Run call recorded in order for assertions.
type fakeExecutor struct {
	mu sync.Mutex

	// scripts maps a command substring to the Result returned when a Run
	// command contains that substring.
	scripts map[string]executor.Result
	// errs maps a command substring to a transport error returned instead
	// of a Result.
	errs map[string]error

	calls []string // every command passed to Run, in call order
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
		if opts != nil && opts.Stream != nil {
			for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
				if line != "" {
					opts.Stream(line)
				}
			}
		}
		return res, nil
	}
	// Default: success, no output.
	return executor.Result{ExitCode: 0}, nil
}

// matchScript returns the longest scripted substring key contained in cmd,
// so a more specific script (e.g. "test -x /usr/local/bin/reth") wins over
// a broader one (e.g. "test -x").
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
	f.calls = append(f.calls, fmt.Sprintf("ReadFile %s", path))
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
