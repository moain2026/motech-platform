#!/usr/bin/env bash
# build.sh — cross-compile the three Motech Windows binaries and (optionally)
# Authenticode-sign them, then publish to the live /download dir.
#
# Usage:
#   ./build.sh            # build + sign (if cert present) + publish
#   ./build.sh --no-sign  # build + publish, skip signing
#   ./build.sh --no-publish
#
# Signing uses a self-signed Al-Abbasi Soft code-signing cert in signing/.
# A self-signed cert makes Windows show the real publisher "Al-Abbasi Soft"
# instead of "Unknown Publisher"; to fully clear SmartScreen you need an
# OV/EV cert from a CA (see signing/README.md) — drop it in as signing/cert.pfx
# and set MOTECH_PFX_PASS, this script will use it automatically.
set -euo pipefail

cd "$(dirname "$0")"
SIGN=1; PUBLISH=1
for a in "$@"; do
  case "$a" in
    --no-sign) SIGN=0 ;;
    --no-publish) PUBLISH=0 ;;
  esac
done

OUT=/tmp/motech-build
PUBDIR=/var/www/motech/download
mkdir -p "$OUT"

SIGNDIR="signing"
CRT="$SIGNDIR/motech-codesign.crt"
CHAIN="$SIGNDIR/motech-codesign-chain.crt"  # leaf + root (embed full chain)
KEY="$SIGNDIR/motech-codesign.key"
PFX="$SIGNDIR/cert.pfx"          # optional real CA cert (takes precedence)
# prefer the full chain bundle when present so the signature validates to root
[[ -f "$CHAIN" ]] && CRT="$CHAIN"
TS_URL="http://timestamp.digicert.com"

echo "==> building windows binaries"
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/motech-setup.exe" ./cmd/setup
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "$OUT/motech-connect-cli.exe" ./cmd/agent
CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 \
  go build -ldflags="-H windowsgui -s -w" -o "$OUT/motech-connect.exe" ./cmd/gui

sign_one() {
  local f="$1"
  local tmp="$f.signed"
  if [[ -f "$PFX" ]]; then
    osslsigncode sign -pkcs12 "$PFX" -pass "${MOTECH_PFX_PASS:-}" \
      -n "Motech Connect — Al-Abbasi Soft" -i "https://qfetmfdn.gensparkclaw.com" \
      -ts "$TS_URL" -in "$f" -out "$tmp"
  else
    osslsigncode sign -certs "$CRT" -key "$KEY" \
      -n "Motech Connect — Al-Abbasi Soft" -i "https://qfetmfdn.gensparkclaw.com" \
      -in "$f" -out "$tmp"   # no timestamp for self-signed (no TSA trust needed)
  fi
  mv "$tmp" "$f"
}

if [[ "$SIGN" == "1" && ( -f "$PFX" || ( -f "$CRT" && -f "$KEY" ) ) ]]; then
  echo "==> signing binaries"
  for f in motech-setup.exe motech-connect-cli.exe motech-connect.exe; do
    sign_one "$OUT/$f"
    osslsigncode verify "$OUT/$f" 2>/dev/null | grep -E "Signature verification|Subject" | head -2 || true
  done
else
  echo "==> signing skipped (no cert or --no-sign)"
fi

echo "==> md5"; md5sum "$OUT"/*.exe

if [[ "$PUBLISH" == "1" ]]; then
  echo "==> publishing to $PUBDIR"
  sudo cp "$OUT"/motech-setup.exe "$OUT"/motech-connect.exe "$OUT"/motech-connect-cli.exe "$PUBDIR/"
  sudo chmod 755 "$PUBDIR"/*.exe
  cp "$OUT/motech-connect.exe" "$OUT/Alabbasi.exe" 2>/dev/null || true
  echo "published:"; ls -la "$PUBDIR"/*.exe
fi
echo "==> done"
