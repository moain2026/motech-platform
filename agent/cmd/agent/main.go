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

	"motech-platform/agent/internal/agent"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	token := fs.String("token", "", "activation (setup) token")
	server := fs.String("server", envOr("MOTECH_SERVER", "http://127.0.0.1:8080"), "backend base URL")
	_ = fs.Parse(os.Args[2:])

	ag := agent.New(*server)

	switch cmd {
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
	case "run":
		if err := ag.RunService(); err != nil {
			log.Fatalf("run: %v", err)
		}
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

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
