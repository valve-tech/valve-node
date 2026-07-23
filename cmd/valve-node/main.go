// Command valve-node sets up and monitors an Ethereum / PulseChain /
// PulseChain-v4 node: one binary, guided setup, sync monitoring, and AI log
// explanations, fronted by a local token-gated web UI.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/valve-tech/valve-node/internal/config"
	"github.com/valve-tech/valve-node/internal/server"
)

//go:embed all:web/dist
var embeddedUI embed.FS

// bindFlagUsage is --bind's help text. Called out here (rather than inline
// in flag.String) so it's independently testable.
const bindFlagUsage = "address to bind the local server to. " +
	"WARNING: binding beyond 127.0.0.1 exposes full control of your servers over plain HTTP"

func main() {
	bind := flag.String("bind", "127.0.0.1:8799", bindFlagUsage)
	noOpen := flag.Bool("no-open", false, "do not open a browser window automatically")
	flag.Parse()

	if warning := bindWarningLine(*bind); warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// Load (or lazily create on first Save) valve-node's local state —
	// known targets, AI provider settings — from ~/.valve-node/config.json.
	// The server re-reads it per-request rather than holding this value, so
	// it's only loaded here to fail fast on a corrupt file before the
	// server starts serving.
	if _, err := config.Load(); err != nil {
		log.Fatalf("valve-node: load config: %v", err)
	}

	uiFS, err := fs.Sub(embeddedUI, "web/dist")
	if err != nil {
		log.Fatalf("valve-node: embedded UI: %v", err)
	}

	token := server.NewSessionToken()
	s := server.New(server.Config{
		Bind:  *bind,
		Token: token,
		UI:    uiFS,
	})

	url := fmt.Sprintf("http://%s/?token=%s", *bind, token)
	fmt.Println(url)

	if !*noOpen {
		openBrowser(url)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := s.ListenAndServe(ctx); err != nil {
		log.Fatalf("valve-node: server: %v", err)
	}
}

// bindWarningLine returns a loud warning line when bind's host is not
// loopback (127.0.0.1 or localhost — the server's safe default), or "" if
// it is. valve-node's local server is token-gated but plain HTTP: binding
// it beyond loopback puts full control of every configured target (setup,
// shell-equivalent install/build commands, log access) on the network
// reachable at that address, over an unencrypted channel a network
// observer can read the session token off of.
func bindWarningLine(bind string) string {
	host, _, err := net.SplitHostPort(bind)
	if err != nil {
		// No port (or an unparsable address) — treat the whole string as
		// the host, e.g. a bare "127.0.0.1" or "0.0.0.0".
		host = bind
	}
	if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
		return ""
	}
	return fmt.Sprintf(
		"WARNING: binding to %s exposes full control of your servers over plain HTTP — "+
			"anyone who can reach %s can drive setup, run install/build commands, and read logs. "+
			"Only bind beyond 127.0.0.1 on a trusted network (e.g. behind an SSH tunnel), never on the open internet.",
		bind, bind,
	)
}

// openBrowser opens url in the user's default browser. Best-effort: errors
// are ignored since this is a convenience, not a requirement.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
