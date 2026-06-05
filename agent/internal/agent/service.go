package agent

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kardianos/service"
)

// svcProgram adapts the Agent to the kardianos service interface.
type svcProgram struct {
	a    *Agent
	stop chan struct{}
}

// Start is called by the service manager; it launches the heartbeat loop.
func (p *svcProgram) Start(s service.Service) error {
	p.stop = make(chan struct{})
	go p.a.loop(p.stop)
	return nil
}

// Stop is called by the service manager on shutdown.
func (p *svcProgram) Stop(s service.Service) error {
	close(p.stop)
	return nil
}

// stableExePath is where the agent installs itself so the service never points
// at a temporary Downloads path.
func stableExePath() string {
	if runtime.GOOS == "windows" {
		pf := os.Getenv("ProgramFiles")
		if pf == "" {
			pf = `C:\Program Files`
		}
		return filepath.Join(pf, "Motech", "motech-connect.exe")
	}
	return ""
}

// installSelfToStableLocation copies the running executable to the stable
// location and returns that path. Returns "" on non-Windows.
func installSelfToStableLocation() (string, error) {
	dst := stableExePath()
	if dst == "" {
		return "", nil
	}
	src, err := os.Executable()
	if err != nil {
		return "", err
	}
	if absSrc, _ := filepath.Abs(src); absSrc == dst {
		return dst, nil // already running from the stable location
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dst, nil
}

// serviceConfig defines how the agent registers itself as a service. When
// exePath is non-empty the service is pinned to that executable (stable path).
func serviceConfig(exePath string) *service.Config {
	cfg := &service.Config{
		Name:        "MotechConnect",
		DisplayName: "Motech Connect Agent",
		Description: "Secure remote access agent (Motech / Al-Abbasi Soft).",
		Arguments:   []string{"run"},
	}
	if exePath != "" {
		cfg.Executable = exePath
	}
	return cfg
}

func (a *Agent) newServiceAt(exePath string) (service.Service, error) {
	return service.New(&svcProgram{a: a}, serviceConfig(exePath))
}

// RunService runs the agent's heartbeat loop directly and blocks forever.
// Used by the Scheduled Task (and foreground). We intentionally do NOT use the
// kardianos SCM runner here because the agent runs under Task Scheduler.
//
// A machine-wide single-instance mutex (Global\MotechConnectAgent) guards
// against duplicate run loops (e.g. a leftover user process + the SYSTEM task
// firing together), which previously caused double heartbeats.
func (a *Agent) RunService() error {
	release, ok := acquireSingleInstance()
	if !ok {
		log.Printf("another motech-connect run loop is already active; exiting this duplicate")
		return nil
	}
	defer release()

	log.Printf("run loop start (single instance acquired)")
	stop := make(chan struct{})
	a.loop(stop) // blocks forever
	return nil
}

// InstallService copies the binary to a stable location and registers it to
// run at boot. On Windows it uses a Scheduled Task (SYSTEM, at startup) which
// is far more reliable than a Go Windows-service for a long-running poll loop.
// It also installs the kardianos service as a best-effort fallback.
func (a *Agent) InstallService() error {
	// Stop any prior task/process so the stable exe isn't locked during copy.
	if runtime.GOOS == "windows" {
		_ = uninstallScheduledTask()
		stopRunningAgent()
	}
	exePath, err := installSelfToStableLocation()
	if err != nil || exePath == "" {
		log.Printf("warn: copy to stable location failed: %v", err)
		if ex, e2 := os.Executable(); e2 == nil {
			exePath = ex
		}
	}
	if runtime.GOOS == "windows" {
		if err := installScheduledTask(exePath); err != nil {
			log.Printf("scheduled task install failed: %v", err)
		} else {
			log.Println("scheduled task installed and started")
			return nil
		}
	}
	// Fallback: kardianos service.
	s, err := a.newServiceAt(exePath)
	if err != nil {
		return err
	}
	_ = s.Stop()
	_ = s.Uninstall()
	if err := s.Install(); err != nil {
		return fmt.Errorf("install service: %w", err)
	}
	log.Println("service installed at", exePath)
	return s.Start()
}

// UninstallService stops and removes the OS service and scheduled task.
func (a *Agent) UninstallService() error {
	_ = uninstallScheduledTask()
	s, err := a.newServiceAt("")
	if err != nil {
		return err
	}
	_ = s.Stop()
	return s.Uninstall()
}
