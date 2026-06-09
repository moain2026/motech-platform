//go:build windows

// Command gui is the graphical Motech Connect installer (motech-connect-gui.exe).
// The end user just: opens it → pastes the license/activation code → clicks
// "تفعيل". The app then registers, joins NetBird, configures SSH, installs the
// Windows service, and starts syncing with the dashboard — all automatically.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"motech-platform/agent/internal/agent"
)

// defaultServer is the dashboard backend the agent talks to.
const defaultServer = "https://qfetmfdn.gensparkclaw.com"

func main() {
	// HEADLESS MODE: the scheduled task / service runs this same exe with the
	// `run` argument to keep the heartbeat loop alive in the background (no GUI
	// window). Without this, the task would try to open a GUI in Session 0 and
	// the heartbeat would never run — making the dashboard show the client as
	// offline even though NetBird SSH works. Handle run/register before any GUI.
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "run":
			ag := agent.New(envOr("MOTECH_SERVER", defaultServer))
			defer func() { _ = recover() }()
			if err := ag.RunService(); err != nil {
				log.Printf("run: %v", err)
			}
			return
		case "register":
			tok := ""
			if len(os.Args) >= 3 {
				tok = os.Args[2]
			}
			ag := agent.New(envOr("MOTECH_SERVER", defaultServer))
			if err := ag.Register(tok); err == nil {
				_ = ag.JoinNetbird()
				_ = ag.SetupAccess()
				_ = ag.InstallService()
				_, _ = ag.Heartbeat()
			}
			return
		}
	}
	var mw *walk.MainWindow
	var tokenEdit *walk.LineEdit
	var serverEdit *walk.LineEdit
	var logBox *walk.TextEdit
	var startBtn *walk.PushButton
	var statusLabel *walk.Label

	appendLog := func(s string) {
		mw.Synchronize(func() { logBox.AppendText(s + "\r\n") })
	}
	setStatus := func(s string, ok bool) {
		mw.Synchronize(func() { statusLabel.SetText(s) })
	}

	run := func() {
		token := strings.TrimSpace(tokenEdit.Text())
		server := strings.TrimSpace(serverEdit.Text())
		if token == "" {
			walk.MsgBox(mw, "تنبيه", "الرجاء إدخال مفتاح الترخيص (رمز التفعيل).", walk.MsgBoxIconWarning)
			return
		}
		if server == "" {
			server = defaultServer
		}
		startBtn.SetEnabled(false)
		setStatus("جارٍ التفعيل...", false)

		go func() {
			defer mw.Synchronize(func() { startBtn.SetEnabled(true) })
			ag := agent.New(server)

			appendLog("• التحقق من مفتاح الترخيص...")
			if err := ag.Register(token); err != nil {
				appendLog("✗ فشل التفعيل: " + err.Error())
				setStatus("فشل التفعيل", false)
				return
			}
			appendLog("✓ تم التحقق من الترخيص بنجاح")

			appendLog("• الانضمام إلى الشبكة الآمنة (NetBird)...")
			if err := ag.JoinNetbird(); err != nil {
				appendLog("⚠ NetBird: " + err.Error())
			} else {
				appendLog("✓ تم الانضمام للشبكة")
			}

			appendLog("• إعداد الوصول الآمن (SSH) + تدوير المفتاح...")
			if err := ag.SetupAccess(); err != nil {
				appendLog("⚠ SSH: " + err.Error())
			} else {
				appendLog("✓ تم إعداد الوصول الآمن")
			}

			appendLog("• تثبيت الخدمة (تعمل تلقائياً عند الإقلاع)...")
			if err := ag.InstallService(); err != nil {
				appendLog("⚠ الخدمة: " + err.Error())
			} else {
				appendLog("✓ تم تثبيت الخدمة")
			}

			appendLog("• إرسال أول مزامنة للوحة التحكم...")
			if _, err := ag.Heartbeat(); err != nil {
				appendLog("⚠ المزامنة: " + err.Error())
			} else {
				appendLog("✓ تمت المزامنة — الجهاز الآن متصل")
			}

			appendLog("")
			appendLog("🎉 اكتمل التثبيت بنجاح. يمكنك إغلاق هذه النافذة.")
			setStatus("✓ متصل بنجاح", true)
		}()
	}

	if _, err := (MainWindow{
		AssignTo: &mw,
		Title:    "Al-Abbasi Soft — تفعيل الاتصال الآمن",
		MinSize:  Size{Width: 480, Height: 520},
		Size:     Size{Width: 480, Height: 560},
		Layout:   VBox{Margins: Margins{Left: 20, Top: 20, Right: 20, Bottom: 20}, Spacing: 12},
		Children: []Widget{
			Label{Text: "Al-Abbasi Soft", Font: Font{Family: "Tahoma", PointSize: 18, Bold: true}},
			Label{Text: "نظام إدارة الوصول الآمن عن بُعد", TextColor: walk.RGB(100, 116, 139)},
			VSpacer{Size: 6},
			Label{Text: "مفتاح الترخيص (رمز التفعيل):", Font: Font{Family: "Tahoma", PointSize: 10, Bold: true}},
			LineEdit{AssignTo: &tokenEdit, CueBanner: "مثال: ABCD-1234-XYZ", Font: Font{Family: "Consolas", PointSize: 12}},
			Label{Text: "عنوان الخادم:", TextColor: walk.RGB(100, 116, 139)},
			LineEdit{AssignTo: &serverEdit, Text: defaultServer},
			PushButton{
				AssignTo:  &startBtn,
				Text:      "🔒 ابدأ التفعيل",
				Font:      Font{Family: "Tahoma", PointSize: 12, Bold: true},
				MinSize:   Size{Height: 44},
				OnClicked: run,
			},
			Label{AssignTo: &statusLabel, Text: "بانتظار إدخال المفتاح...", Font: Font{Family: "Tahoma", PointSize: 10, Bold: true}},
			Label{Text: "سجل العملية:", TextColor: walk.RGB(100, 116, 139)},
			TextEdit{AssignTo: &logBox, ReadOnly: true, VScroll: true, MinSize: Size{Height: 180}, Font: Font{Family: "Consolas", PointSize: 9}},
			Label{Text: "© Al-Abbasi Soft", TextColor: walk.RGB(148, 163, 184)},
		},
	}).Run(); err != nil {
		fmt.Println("gui error:", err)
	}
}

// envOr returns the env var value or a fallback default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
