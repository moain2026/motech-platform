package agent

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// netbirdWindowsInstaller is the official NetBird Windows MSI installer URL.
// (latest release; silent install supported via msiexec /qn)
const netbirdWindowsInstaller = "https://pkgs.netbird.io/windows/x64"

// netbirdInstalledPath returns the netbird executable path if installed, else "".
func netbirdInstalledPath() string {
	if p, err := exec.LookPath("netbird"); err == nil {
		return p
	}
	// common install location on Windows
	if runtime.GOOS == "windows" {
		candidates := []string{
			`C:\Program Files\NetBird\netbird.exe`,
			filepath.Join(os.Getenv("ProgramFiles"), "NetBird", "netbird.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	return ""
}

// EnsureNetbirdInstalled downloads and silently installs NetBird on Windows if
// it is not already present. Returns the path to the netbird executable.
func (a *Agent) EnsureNetbirdInstalled() (string, error) {
	if p := netbirdInstalledPath(); p != "" {
		return p, nil
	}
	if runtime.GOOS != "windows" {
		// dev/test environment: cannot install; report clearly.
		return "", fmt.Errorf("netbird not installed (auto-install only on windows)")
	}

	tmp := filepath.Join(os.TempDir(), "netbird_installer.exe")
	if err := download(netbirdWindowsInstaller, tmp); err != nil {
		return "", fmt.Errorf("download netbird: %w", err)
	}
	defer os.Remove(tmp)

	// NetBird ships an NSIS installer; /S = silent install.
	cmd := exec.Command(tmp, "/S")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("install netbird: %w (%s)", err, string(out))
	}
	// Installer returns before files settle; poll for the executable up to 60s.
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		if p := netbirdInstalledPath(); p != "" {
			time.Sleep(3 * time.Second) // let the netbird service come up
			return p, nil
		}
	}
	return "", fmt.Errorf("netbird installed but executable not found after wait (قد يحتاج إعادة تشغيل)")
}

// download fetches url into dst.
func download(url, dst string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
