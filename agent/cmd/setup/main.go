// Command setup is a console-based interactive installer (motech-setup.exe).
// It is the reliable fallback to the GUI: a console window prompts for the
// activation code, then auto-installs everything. Always works (no GUI deps).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"motech-platform/agent/internal/agent"
)

const defaultServer = "https://qfetmfdn.gensparkclaw.com"

func main() {
	fmt.Println("============================================")
	fmt.Println("   Al-Abbasi Soft - Secure Access Installer")
	fmt.Println("   نظام إدارة الوصول الآمن")
	fmt.Println("============================================")
	fmt.Println()

	server := defaultServer
	if v := os.Getenv("MOTECH_SERVER"); v != "" {
		server = v
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your license / activation code (رمز التفعيل): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token == "" {
		fmt.Println("No code entered. Press Enter to exit.")
		_, _ = reader.ReadString('\n')
		return
	}

	ag := agent.New(server)
	step := func(name string, fn func() error) {
		fmt.Printf("- %s ... ", name)
		if err := fn(); err != nil {
			fmt.Println("WARN:", err)
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Println("\nStarting setup...")
	fmt.Print("- Verifying license ... ")
	if err := ag.Register(token); err != nil {
		fmt.Println("FAILED:", err)
		fmt.Println("\nتأكد من الرمز وحاول مرة أخرى. اضغط Enter للخروج.")
		_, _ = reader.ReadString('\n')
		return
	}
	fmt.Println("OK")

	step("Joining NetBird mesh", ag.JoinNetbird)
	step("Configuring SSH access", ag.SetupAccess)
	step("Installing Windows service", ag.InstallService)
	step("Syncing with dashboard", func() error { _, err := ag.Heartbeat(); return err })

	fmt.Println("\n🎉 Setup complete! هذا الجهاز الآن متصل بلوحة التحكم.")
	fmt.Println("يمكنك إغلاق هذه النافذة. Press Enter to exit.")
	_, _ = reader.ReadString('\n')
}
