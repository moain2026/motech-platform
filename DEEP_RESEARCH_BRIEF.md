# Deep Research Brief — Motech Remote Access Platform

**Repository (PUBLIC, clone & analyze it):** https://github.com/moain2026/motech-platform

You are a senior systems/security architect. Clone the repo above, read the full codebase (Go backend + Go Windows agent + dashboard + `docs/` + `planning/`), then deliver a deep, actionable research report. Be concrete, cite files/lines, and give working examples — not generic advice.

---

## 1. What the product is
A **centralized secure remote-access management platform** for Windows client machines across many branches/companies (an MSP use-case). Scaling target: from ~3 clients today to **1000+**.

Three components (all in the repo):
1. **Backend + DB** — Go (chi, sqlx) + PostgreSQL. JWT auth (admin + agent), AES-256-GCM encryption of SSH private keys at rest, NetBird Cloud integration, one-time setup tokens, activity log, key rotation state machine.
2. **Dashboard** — server-rendered (Tailwind + Alpine, RTL Arabic). Client list, online/offline, add client (generates setup token + installer link), SSH key management + rotation, disable/delete, copy connection info, activity log.
3. **Agent** — `motech-connect.exe` (Go, cross-compiled for Windows, ~10MB, Authenticode-signed by "Al-Abbasi Soft"). One-time setup via activation token → installs/joins NetBird mesh → receives a unique SSH key → configures `administrators_authorized_keys` + firewall + OpenSSH Server → registers as a Windows service/scheduled task → sends heartbeat every 20s → applies key rotation / disable commands.

Tech facts: Go 1.23 (backend) / 1.25 (agent), ~3150 lines of Go. Agent deps: `kardianos/service`, `lxn/walk` (GUI), `golang.org/x/crypto`. NetBird Cloud (free tier). Primary access = OpenSSH over NetBird mesh; NetBird's built-in SSH is a future enhancement.

The real end goal: an operator (or an AI agent) pastes "connection info" and does SSH-based work on any client machine, securely, with per-client unique keys that can be rotated/revoked centrally.

---

## 2. The concrete problems we hit (analyze each, root-cause + fix)
1. **Agent install must be 100% silent (no windows).** Every external command (PowerShell, netsh, schtasks, taskkill, netbird, icacls, the NetBird installer) historically flashed a console window during install. We added a `silentCmd` helper (`HideWindow + CREATE_NO_WINDOW`, `-WindowStyle Hidden`). Verify our approach in `agent/internal/agent/cmd_windows.go`, `sshd_windows.go`, `task_windows.go`, `netbird_install.go`. Is it complete and correct? Any missed window-spawning paths? Better patterns (e.g. building the agent as a GUI-subsystem binary, job objects, `STARTUPINFO`)?
2. **Install must be sequential and never stall.** It should run each step, report progress, and always advance to the end with a clear "installed successfully" / "completed with warnings" — never hang mid-step. We saw `netbird up` **block indefinitely** (waiting on interactive SSO when a setup-key path stalls). We added a 45s context timeout. Review `agent/internal/agent/agent.go` `JoinNetbird` + `cmd/agent/main.go` register flow. Is the timeout strategy right? What are robust patterns for "install that always finishes"?
3. **Final verification before declaring success.** We added `VerifySSHReady` (checks sshd Running + our public key present in `administrators_authorized_keys`). Review `sshd_windows.go`. Is the verification sufficient/correct? What else should be verified (firewall rule, NetBird peer connected, heartbeat acked)?
4. **The new agent .exe appeared to crash/produce no output on a real Windows 10 Pro machine** after our edits, while it built cleanly and `go vet` passed. We could NOT reliably capture stdout because we were driving it over `ssh -> sshd-on-Windows (PowerShell default shell) -> cmd /c` through a Cloudflare quick tunnel, and quoting/redirection kept breaking. **Determine:** did our edits actually break the binary, or was it purely the test harness (nested bash→ssh→PowerShell quoting + Windows OpenSSH default shell)? Give a reliable, repeatable way to run and capture an exe's stdout/stderr/exit-code over Windows OpenSSH. Note: a previous build of this same agent ran fine on this machine on 2026-06-05.
5. **`administrators_authorized_keys` ACL pitfall.** On one client, SSH negotiated fully (KEX/NEWKEYS) then `Connection reset` at port 22 before publickey auth — the classic Windows OpenSSH ACL issue (file must be owned by Administrators+SYSTEM only, inheritance disabled). Confirm our `icacls` fix in `sshd_windows.go` is correct and complete for all Windows 10/11/Server versions.

---

## 3. What we want (objectives)
- An agent install experience that is **silent, sequential, self-verifying, idempotent, and reliable across ANY Windows version (10/11/Server 2016–2025)** with zero manual steps and no visible windows except (optionally) our own UI.
- **Trust without per-machine setup**: today the agent is self-signed (needs `ca.crt` imported via GPO). We want SmartScreen/AV to never block it at 1000-machine scale.
- **Robust packaging**: evaluate MSI (WiX) vs bare exe vs bootstrapper; UAC elevation manifest; clean uninstall; GPO/SCCM/Intune mass deployment.
- A correct, production-grade **key rotation + revocation** model and **online/offline** accuracy.

---

## 4. Questions to answer with evidence
**A. Language/runtime for the agent:** We chose **Go**. Validate vs alternatives (Rust, C#/.NET, C++, PowerShell-only). For a long-running Windows service that manages SSH keys + mesh VPN + heartbeat at 1000+ scale, what is genuinely best on: silent operation, AV/EDR friendliness, service robustness, single-binary deploy, signing, footprint? Is there anything materially stronger than our current Go approach?

**B. Silent/headless Windows execution — the definitive guide:** the correct way to spawn child processes with no console window from a Go Windows binary (subsystem choice, `SysProcAttr`, `CREATE_NO_WINDOW`, job objects), and how Windows service Session 0 isolation affects stdout/UI. Confirm our pattern or give a better one.

**C. Reliable remote testing for US/me:** **How do I run a real Windows test environment in MY environment** (a Linux VM with fast internet) so I stop depending on the user's machine + a flaky tunnel? Evaluate and give step-by-step setup for the best option: (a) Windows in QEMU/KVM on Linux, (b) cloud Windows VM (Azure/AWS/GCP free or cheap), (c) Windows Sandbox, (d) Wine/`wine` limits for this agent. I need to: build the exe, run it, join NetBird, SSH in, watch logs — fast and repeatably. Give exact commands.

**D. Code analysis & strength review:** Read the actual Go code. Assess architecture quality, security (JWT handling, AES-GCM key management, token lifecycle, master-key rotation per ADR-008), the rotation state machine, error handling, the agent's service/scheduled-task survival logic under SYSTEM/Session 0. **Where is it strong, where is it weak, and what concrete changes make it stronger?** Cite files.

**E. International standards & best practices** for: secure remote access / mesh VPN (NetBird/WireGuard/Tailscale comparison), SSH key lifecycle management at scale, Windows agent/installer hardening, code-signing (OV vs EV, Azure Trusted Signing), zero-trust access, audit logging (what a real SOC expects). Map our design against them and list gaps.

---

## 5. Deliverable format
1. Executive summary (what's good, what's risky, top 5 priorities).
2. Per-problem root cause + concrete fix (with code/commands).
3. Agent language/packaging recommendation with justification.
4. **Exact, copy-paste setup for a Windows test environment on a Linux VM.**
5. Code-level findings (file:line) — strengths + weaknesses + stronger alternatives.
6. Standards/best-practices gap analysis + prioritized roadmap.
