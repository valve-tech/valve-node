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
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/valve-tech/valve-node/internal/server"
)

//go:embed all:web/dist
var embeddedUI embed.FS

func main() {
	bind := flag.String("bind", "127.0.0.1:8799", "address to bind the local server to")
	noOpen := flag.Bool("no-open", false, "do not open a browser window automatically")
	flag.Parse()

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
