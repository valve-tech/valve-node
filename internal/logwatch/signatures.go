package logwatch

import "regexp"

// signature is one recognized error/failure pattern: a regex over a raw
// journald line, the severity it implies, a canned plain-English
// explanation for an operator, and (where a page exists) a learn.valve.city
// deep link for further reading.
type signature struct {
	name     string
	pattern  *regexp.Regexp
	severity string
	explain  string
	learnURL string
}

const learnRPCBase = "https://learn.valve.city/rpc"

// signatures is checked in order, first match wins. Order matters only
// where patterns could otherwise overlap; today's patterns are already
// disjoint but new entries should be added with that in mind.
var signatures = []signature{
	{
		name:     "beacon-stalled",
		pattern:  regexp.MustCompile(`(?i)\bstalled\b`),
		severity: "critical",
		explain:  "The beacon client's sync state machine has stopped making progress. Restart the beacon client to force it to recover.",
		learnURL: learnRPCBase + "#syncing",
	},
	{
		name:     "engine-auth",
		pattern:  regexp.MustCompile(`(?i)jwt|401|unauthorized`),
		severity: "critical",
		explain:  "The execution and beacon clients can't authenticate to each other over the engine API. This is almost always a mismatched or missing JWT secret file — check both clients point at the same jwt.hex.",
		learnURL: learnRPCBase + "#jwt-secret",
	},
	{
		name:     "checkpoint-sync-failed",
		pattern:  regexp.MustCompile(`(?i)checkpoint sync`),
		severity: "error",
		explain:  "The beacon client failed to bootstrap from a checkpoint sync provider. Check the checkpoint-sync URL is reachable and serving a recent state, or fall back to genesis sync.",
		learnURL: learnRPCBase + "#syncing",
	},
	{
		name:     "low-peer-count",
		pattern:  regexp.MustCompile(`(?i)low peer count|0 peers`),
		severity: "warn",
		explain:  "The client has too few (or zero) peers to sync or serve requests reliably. Check outbound P2P ports are open and reachable from the internet.",
		learnURL: learnRPCBase + "#ports",
	},
	{
		name:     "disk-full",
		pattern:  regexp.MustCompile(`(?i)no space left on device`),
		severity: "critical",
		explain:  "The data disk is full. The client will stall or crash-loop until space is freed — prune, expand the volume, or move the data directory to a larger disk.",
	},
	{
		name:     "database-corrupt",
		pattern:  regexp.MustCompile(`(?i)database.*corrupt`),
		severity: "critical",
		explain:  "The client's local database is corrupt, usually from an unclean shutdown or a disk fault. Restore from a snapshot or resync from scratch; do not keep restarting against a corrupt database.",
	},
	{
		name:     "oom-killed",
		pattern:  regexp.MustCompile(`(?i)killed process|\boom\b`),
		severity: "critical",
		explain:  "The kernel OOM-killer terminated the process because the box ran out of memory. Reduce cache sizes, add swap, or move to a box with more RAM.",
	},
	{
		name:     "port-in-use",
		pattern:  regexp.MustCompile(`(?i)bind: address already in use`),
		severity: "critical",
		explain:  "The client failed to bind a listening port because something else already holds it — often a previous instance of the same process that didn't exit cleanly. Check for a stray process and stop it before restarting.",
		learnURL: learnRPCBase + "#ports",
	},
}

// levelSeverity maps an unclassified line's log-level word to a Hit
// severity, checked in this priority order since a line could in principle
// contain more than one level word.
var (
	levelCriticalRe = regexp.MustCompile(`(?i)crit|fatal`)
	// "erro" (not "error") so lighthouse-pulse's abbreviated ERRO level tag
	// matches too — "error" still matches since it contains "erro".
	levelErrorRe = regexp.MustCompile(`(?i)erro`)
	levelWarnRe  = regexp.MustCompile(`(?i)warn`)
)

func levelSeverity(line string) (string, bool) {
	switch {
	case levelCriticalRe.MatchString(line):
		return "critical", true
	case levelErrorRe.MatchString(line):
		return "error", true
	case levelWarnRe.MatchString(line):
		return "warn", true
	}
	return "", false
}
