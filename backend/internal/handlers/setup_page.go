package handlers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// setupPageData feeds the install page template.
type setupPageData struct {
	Token       string
	ClientName  string
	Valid       bool
	Reason      string // why invalid (used/expired/unknown)
	DownloadURL string
}

// SetupPage serves a public, self-contained install page at /setup/{token}.
// The branch user opens this single link: it shows the client name, a download
// button for the signed installer, and clear step-by-step Arabic instructions.
// No login required — the one-time token in the URL is the only secret.
func (h *Handler) SetupPage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	d := setupPageData{
		Token:       token,
		DownloadURL: "/download/Alabbasi_soft.exe",
	}

	// Validate token without consuming it (consumption happens at register).
	var clientName string
	var usedAt *time.Time
	var expiresAt time.Time
	row := h.DB.QueryRow(`
		SELECT c.name, t.used_at, t.expires_at
		FROM setup_tokens t JOIN clients c ON c.id = t.client_id
		WHERE t.token_hash = $1`, hashToken(token))
	if err := row.Scan(&clientName, &usedAt, &expiresAt); err != nil {
		d.Valid = false
		d.Reason = "رمز التفعيل غير معروف أو تم حذف العميل."
	} else if usedAt != nil {
		d.Valid = false
		d.ClientName = clientName
		d.Reason = "تم استخدام هذا الرمز من قبل. اطلب رمزاً جديداً من المسؤول."
	} else if time.Now().After(expiresAt) {
		d.Valid = false
		d.ClientName = clientName
		d.Reason = "انتهت صلاحية هذا الرمز. اطلب رمزاً جديداً من المسؤول."
	} else {
		d.Valid = true
		d.ClientName = clientName
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = setupTmpl.Execute(w, d)
}

var setupTmpl = template.Must(template.New("setup").Parse(setupHTML))

const setupHTML = `<!DOCTYPE html>
<html lang="ar" dir="rtl">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>تثبيت Motech Connect — Al-Abbasi Soft</title>
<script src="https://cdn.tailwindcss.com"></script>
<link href="https://fonts.googleapis.com/css2?family=Tajawal:wght@400;500;700&display=swap" rel="stylesheet"/>
<style>body{font-family:'Tajawal',sans-serif}</style>
</head>
<body class="bg-slate-50 min-h-screen flex items-center justify-center p-4">
  <div class="bg-white rounded-3xl shadow-xl max-w-lg w-full p-8 space-y-6">
    <div class="text-center space-y-1">
      <div class="text-2xl font-bold text-emerald-600">Al-Abbasi Soft</div>
      <div class="text-sm text-slate-400">نظام إدارة الوصول الآمن — Motech Connect</div>
    </div>

    {{if .Valid}}
    <div class="bg-emerald-50 border border-emerald-200 rounded-2xl p-4 text-center">
      <div class="text-sm text-slate-500">تثبيت الوكيل للجهاز</div>
      <div class="text-lg font-bold text-emerald-700">{{.ClientName}}</div>
    </div>

    <a href="{{.DownloadURL}}" download="Alabbasi_soft.exe"
       class="flex items-center justify-center gap-2 w-full bg-emerald-600 hover:bg-emerald-500 text-white py-4 rounded-2xl font-bold text-lg transition">
      ⬇ تحميل برنامج التثبيت
    </a>

    <div class="space-y-3 text-sm text-slate-600">
      <div class="font-bold text-slate-800">الخطوات:</div>
      <div class="flex gap-3"><span class="shrink-0 w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 grid place-items-center font-bold">1</span><div>اضغط <b>تحميل برنامج التثبيت</b> بالأعلى.</div></div>
      <div class="flex gap-3"><span class="shrink-0 w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 grid place-items-center font-bold">2</span><div>افتح الملف <code class="bg-slate-100 px-1 rounded">Alabbasi_soft.exe</code> (لو ظهر تحذير ويندوز اضغط «مزيد من المعلومات» ← «تشغيل على أي حال»).</div></div>
      <div class="flex gap-3"><span class="shrink-0 w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 grid place-items-center font-bold">3</span><div>عند طلب رمز التفعيل، الصق هذا الرمز:</div></div>
      <div class="flex items-center gap-2">
        <code id="tok" class="flex-1 font-mono bg-slate-100 px-3 py-2 rounded-lg text-center tracking-widest text-emerald-700">{{.Token}}</code>
        <button onclick="navigator.clipboard.writeText('{{.Token}}');this.textContent='✓'" class="px-3 py-2 bg-emerald-600 text-white rounded-lg">📋</button>
      </div>
      <div class="flex gap-3"><span class="shrink-0 w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 grid place-items-center font-bold">4</span><div>انتظر حتى يكتمل التثبيت تلقائياً — سيظهر الجهاز «متصل» في لوحة التحكم خلال دقيقة.</div></div>
    </div>

    <div class="text-xs text-slate-400 text-center border-t pt-4">
      هذا الرمز يُستخدم مرة واحدة فقط. لا تشاركه مع أحد.
    </div>
    {{else}}
    <div class="bg-red-50 border border-red-200 rounded-2xl p-6 text-center space-y-2">
      <div class="text-4xl">⚠️</div>
      <div class="font-bold text-red-700">تعذّر التثبيت</div>
      <div class="text-sm text-slate-600">{{.Reason}}</div>
      {{if .ClientName}}<div class="text-xs text-slate-400">الجهاز: {{.ClientName}}</div>{{end}}
    </div>
    {{end}}
  </div>
</body>
</html>`
