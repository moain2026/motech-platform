// Command agent is the Motech client agent (motech-connect.exe).
//
// Usage:
//
//	motech-connect register --token ABCD-1234-XYZ [--server https://...]
//	motech-connect run            # run heartbeat loop (used by the service)
//	motech-connect install        # install as a Windows service
//	motech-connect uninstall      # remove the Windows service
//
// The typical end-user flow is the GUI/one-liner: enter the activation token,
// the agent registers, joins the NetBird mesh, installs itself as a service,
// and starts sending heartbeats.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"motech-platform/agent/internal/agent"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]

	// CRITICAL (Windows service/SYSTEM): a Windows service runs in Session 0
	// with NO stdout/stderr handles. Writing to them (incl. fmt.Print) can kill
	// the process instantly. For the long-running `run` command we redirect all
	// logging to a file up front and never touch stdout, and we recover panics.
	if cmd == "run" {
		logToFile()
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in run: %v", r)
			}
		}()
		ag := agent.New(envOr("MOTECH_SERVER", "http://127.0.0.1:8080"))
		if err := ag.RunService(); err != nil {
			log.Printf("run: %v", err)
		}
		return
	}
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	token := fs.String("token", "", "activation (setup) token")
	server := fs.String("server", envOr("MOTECH_SERVER", "http://127.0.0.1:8080"), "backend base URL")
	_ = fs.Parse(os.Args[2:])

	ag := agent.New(*server)

	switch cmd {
	case "run":
		// handled above
	case "register":
		if *token == "" {
			log.Fatal("--token is required for register")
		}
		if err := ag.Register(*token); err != nil {
			log.Fatalf("register failed: %v", err)
		}
		fmt.Println("✓ registered. joining NetBird and installing service...")
		if err := ag.JoinNetbird(); err != nil {
			log.Printf("warning: netbird join: %v", err)
		}
		if err := ag.InstallService(); err != nil {
			log.Printf("warning: service install: %v", err)
		}
		fmt.Println("✓ done. agent is now connected.")
	case "install":
		if err := ag.InstallService(); err != nil {
			log.Fatalf("install: %v", err)
		}
	case "uninstall":
		if err := ag.UninstallService(); err != nil {
			log.Fatalf("uninstall: %v", err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`motech-connect — Motech client agent
  register --token <TOKEN> [--server <URL>]   register this machine
  run                                         run heartbeat loop
  install / uninstall                         manage the Windows service`)
}

// logToFile redirects the standard logger to a file next to the executable's
// stable data dir, with an absolute path (service working dir is System32).
func logToFile() {
	dir := os.Getenv("ProgramData")
	if dir == "" {
		dir = `C:\ProgramData`
	}
	logDir := filepath.Join(dir, "Motech")
	_ = os.MkdirAll(logDir, 0o755)
	f, err := os.OpenFile(filepath.Join(logDir, "agent.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f) // file ONLY — never os.Stdout under a service
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
