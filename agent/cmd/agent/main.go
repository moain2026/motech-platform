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
	"io"
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
		log.Printf("run: starting (logToFile active)")
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
		// Sequential install: each step runs, reports, and ALWAYS continues to
		// the next — the install never stalls mid-way. Non-fatal steps log a
		// warning but do not abort, so we always reach the final verification.
		step := func(n int, name string, fn func() error) {
			fmt.Printf("[%d/5] %s ...\n", n, name)
			if err := fn(); err != nil {
				fmt.Printf("      ⚠ %s: %v (نُكمل)\n", name, err)
			} else {
				fmt.Printf("      ✓ %s\n", name)
			}
		}

		// Step 1 is the only hard requirement: without a valid token there is
		// nothing to install, so this one aborts on failure.
		fmt.Println("[1/5] التسجيل بالرمز ...")
		if err := ag.Register(*token); err != nil {
			log.Fatalf("      ✗ فشل التسجيل: %v", err)
		}
		fmt.Println("      ✓ تم التسجيل")

		step(2, "الانضمام لشبكة NetBird", ag.JoinNetbird)
		step(3, "تثبيت مفتاح SSH وتشغيل خادم SSH", ag.SetupAccess)
		step(4, "تثبيت الخدمة في الخلفية", ag.InstallService)

		// Step 5: verify everything is actually ready before declaring success.
		fmt.Println("[5/5] التحقق من الجاهزية ...")
		msg, verr := agent.VerifySSHReady(ag.InstalledPubKey())
		if verr != nil {
			fmt.Printf("      ⚠ %s\n", msg)
			fmt.Println("\n⚠ اكتمل التثبيت مع تنبيهات — راجع الرسائل أعلاه.")
		} else {
			fmt.Printf("      ✓ %s\n", msg)
			fmt.Println("\n✅ تم التثبيت بنجاح — الجهاز متصل وجاهز.")
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

// logToFile redirects the standard logger to a file next to the executable's
// stable data dir, with an absolute path (service working dir is System32).
//
// CRITICAL under Windows SYSTEM/Session 0: there is NO stdout/stderr handle, so
// the default logger (which writes to os.Stderr) can KILL the process on the
// first log call. We therefore ALWAYS redirect away from stderr: to the file if
// we can open it, otherwise to io.Discard. We never leave the logger on stderr.
func logToFile() {
	// Default to discard FIRST so any early failure can't touch stderr.
	log.SetOutput(io.Discard)

	candidates := []string{}
	if pd := os.Getenv("ProgramData"); pd != "" {
		candidates = append(candidates, filepath.Join(pd, "Motech"))
	}
	candidates = append(candidates, `C:\ProgramData\Motech`, os.TempDir())

	for _, logDir := range candidates {
		if logDir == "" {
			continue
		}
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			continue
		}
		f, err := os.OpenFile(filepath.Join(logDir, "agent.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			continue
		}
		log.SetOutput(f) // file ONLY — never os.Stderr under a service
		return
	}
	// stays on io.Discard if no writable location was found
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
