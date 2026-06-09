# Deep Research Brief #2 — How can a Linux cloud agent build, run, and TEST a Windows .exe agent, fast, WITHOUT nested virtualization?

You are a senior DevOps/virtualization/cloud architect. I need you to research and return **concrete, working methods** (with exact commands) for a very specific constrained environment. Think beyond the obvious — I want clever approaches I haven't considered. Be specific, cite tools/docs, and give copy-paste commands.

---

## WHO I AM (the constrained machine doing the testing)
I am "Genspark Claw", an autonomous AI assistant that lives **inside a Linux cloud VM** and operates it via shell, files, and a browser. My full environment:

| Property | Value |
|---|---|
| Cloud | Microsoft Azure |
| VM size | **Standard_B4as_v2** (B-series, burstable) |
| Region | SwedenCentral |
| OS | Ubuntu 24.04.4 LTS, kernel 6.17 (azure) |
| Arch | x86_64 (AMD EPYC 7763) |
| CPU | 4 vCPU |
| RAM | 15 GiB (~10 GiB free) |
| Disk | ~93 GB free |
| **`/dev/kvm`** | **MISSING** — Azure B-series does NOT support nested virtualization |
| Hypervisor | Microsoft (I am already a guest VM) |
| Internet | very fast (cloud backbone) |
| Installed | go, python3, node, cloudflared, az (Azure CLI), gh, ffmpeg, caddy, **netbird (joined to a mesh)**, ssh, git |
| NOT installed | docker, qemu, wine |
| Browser/VNC | I have a headless Chromium + Xvfb + x11vnc + noVNC desktop (Linux only) |
| Azure access | I can `az login` to the USER's Azure subscription (same subscription this VM runs in: `openclaw-rg`) |

**How I'm connected to docs & how I analyze:** I read files directly, run shell commands, fetch URLs, and can clone public Git repos. I reason over code/logs myself. I can spawn sub-agents and run long background jobs.

---

## WHAT I NEED TO TEST
A **Windows** client agent: `motech-connect.exe` (Go, cross-compiled `GOOS=windows`, ~10 MB, Authenticode-signed). On a real Windows host it must: register via token to a backend, **install/join NetBird mesh VPN (TUN driver)**, configure **Windows OpenSSH Server** + `administrators_authorized_keys` + firewall, register as a **Windows service / scheduled task**, send heartbeats, apply SSH-key rotation. Source repo (PUBLIC): `github.com/moain2026/motech-platform`.

I need to: **build the exe (already easy on Linux), then RUN it on real Windows, watch its stdout/stderr/logs, confirm NetBird joins, SSH into the Windows box, test key rotation — fast, repeatable, and ideally without depending on the user's slow-internet physical PC + a flaky Cloudflare tunnel** (that combo already wasted hours: nested `ssh→PowerShell→cmd` quoting + tunnel buffering ate all output).

---

## THE HARD CONSTRAINT
- I **cannot run Windows locally** (no `/dev/kvm`; B-series = no nested virt; QEMU would be pure-software TCG = unusably slow; Docker `dockur/windows` also needs KVM; Wine can't do NetBird's TUN driver, Windows OpenSSH service, Authenticode, Session-0, or Task Scheduler XML).
- So local Windows emulation is OUT. I must find another path that leverages my **fast cloud connectivity** and **Azure access**.

---

## RESEARCH QUESTIONS (find the smartest path, including ones I haven't thought of)
1. **Azure-native Windows test targets.** Given I can `az login` to the same subscription: what is the fastest/cheapest repeatable way to spin up a throwaway Windows host I can SSH into and watch logs? Compare: Azure VM (B2s) vs **Azure Container Instances with Windows containers** vs **Azure Dev Box** vs **Windows 365 Cloud PC** vs a **spot VM**. Give exact `az` commands, cost, teardown, and how to capture exe stdout reliably (note: Windows OpenSSH default shell = PowerShell — set `DefaultShell=cmd.exe` to fix quoting). Can I make the Windows VM **join the same NetBird mesh** so I reach it over the mesh with no public RDP/SSH exposure?
2. **Could I enable nested virtualization at all?** Is there an Azure VM size in the same family/region (e.g. Dv5/Ev5/Dasv5) that supports nested virt, and could the user resize this very VM to it so I CAN run QEMU/Windows locally? Give the `az vm resize` path + which sizes support nested virt + caveats.
3. **CI-as-test-rig.** Use **GitHub Actions `windows-latest` runners** (the repo is on GitHub) as a free, fast, ephemeral Windows test environment: build + run the agent in a workflow, capture logs as artifacts, even start a NetBird peer inside the runner and exercise registration against my backend. Is this viable? Give a working `.github/workflows/win-e2e.yml` that runs the exe, captures output, and uploads logs. What are the limits (NetBird TUN in Actions? service install? admin rights?)?
4. **Remote Windows-as-a-service / sandbox APIs.** Any programmatic, API-driven ephemeral Windows (e.g. cloud sandbox providers, browserless Windows VMs, Win365 trial, AWS EC2 Windows spot, GCP) that a Linux agent can drive headlessly end-to-end via CLI/API? Compare speed, cost, and "can it load a kernel TUN driver + run a Windows service."
5. **Reduce dependence on a Windows box at all.** Which parts of the agent can I unit/integration-test on **Linux** (the agent already builds for linux and no-ops the Windows-only paths)? Propose a layered test strategy: (a) Linux for logic/register/heartbeat/rotation/HTTP, (b) a thin Windows smoke test only for the truly Windows-specific bits (TUN, OpenSSH ACL, service). Map exactly which source files/functions fall in each bucket (the repo has build-tagged `*_windows.go` / `*_other.go`).
6. **Clever / non-obvious ideas.** Anything I'm missing: e.g. mock the Windows-specific syscalls behind interfaces and run a Linux "fake Windows" harness; use `wine` only for the pure-CLI register/heartbeat path (no TUN) just to smoke-test argument parsing & stdout; ReactOS; pre-baked Windows cloud images with NetBird+OpenSSH already installed to cut setup to seconds; snapshot/restore for one-second resets.

---

## DELIVERABLE
1. Ranked recommendation: the single best path for MY exact constraints (B4as_v2, no KVM, Azure access, fast net) — with reasoning.
2. Exact copy-paste setup for the top 2 options (commands, costs, teardown, how to capture stdout, how to join NetBird mesh).
3. The GitHub Actions Windows-E2E workflow file (ready to commit).
4. The layered Linux-vs-Windows test-bucket map for this specific repo (file:line where possible).
5. Any clever approaches I clearly haven't considered, with honest pros/cons.
