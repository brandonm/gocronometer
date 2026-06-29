#!/usr/bin/env bash
#
# test-export.sh — probe the Cronometer CSV /export endpoint in isolation from
# the auto-journal app. Reports OK (export works) / 429 (rate limited) / error.
#
# Why: "the website works but the app 429s" can mean the export quota is spent
# from the *server's* IP. Run this from your Mac AND (via --remote) from the
# deploy box and compare. Browsing the site uses a different, unmetered API, so
# it tells you nothing about the export quota.
#
# Credentials, in priority order (the secret is never printed, logged, or
# placed in argv / shell history):
#   1) CRONOMETER_EMAIL / CRONOMETER_PASSWORD environment variables
#   2) macOS Keychain (recommended). Store once — hidden, NOT in shell history:
#        security add-generic-password -U -s cronometer-test -a 'you@example.com' -w
#      (-w as the last arg makes security PROMPT for the password.)
#   3) a creds file (default ./.cronometer-creds, gitignored) — plaintext, least secure
#   4) interactive prompt (password input hidden)
#
# Usage:
#   ./scripts/test-export.sh                       # one export, yesterday..today
#   ./scripts/test-export.sh -n 12                 # probe the daily cap (spends quota!)
#   ./scripts/test-export.sh 2026-06-20 2026-06-21 # explicit date range
#   ./scripts/test-export.sh --remote user@host    # run from the deploy box's IP
#   REMOTE_ARCH=arm64 ./scripts/test-export.sh --remote user@host
#
set -euo pipefail
cd "$(dirname "$0")/.."   # gocronometer repo root

REMOTE=""
ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --remote) REMOTE="${2:?--remote needs an ssh target, e.g. user@host}"; shift 2 ;;
    -h|--help) sed -n '2,28p' "$0"; exit 0 ;;
    *) ARGS+=("$1"); shift ;;
  esac
done

# --- credentials (priority: env > macOS Keychain > creds file > prompt) ---
# The secret never appears in argv, shell history, or this script's output —
# it flows Keychain -> env -> the tester's subprocess, and is never echoed.
KC_SERVICE="${CRONOMETER_KEYCHAIN_SERVICE:-cronometer-test}"

# macOS Keychain — store once (prompts, hidden, NOT saved to shell history):
#   security add-generic-password -U -s cronometer-test -a 'you@example.com' -w
# (password is the secret; the email is the item's account name.)
if [[ -z "${CRONOMETER_PASSWORD:-}" ]] && command -v security >/dev/null 2>&1; then
  if pw=$(security find-generic-password -s "$KC_SERVICE" -w 2>/dev/null); then
    CRONOMETER_PASSWORD="$pw"
    [[ -z "${CRONOMETER_EMAIL:-}" ]] && CRONOMETER_EMAIL=$(
      security find-generic-password -s "$KC_SERVICE" 2>/dev/null | awk -F'"' '/"acct"/{print $4}'
    )
  fi
fi

# Fallback: gitignored creds file (plaintext on disk — least secure).
CREDS_FILE="${CRONOMETER_CREDS_FILE:-.cronometer-creds}"
if { [[ -z "${CRONOMETER_EMAIL:-}" ]] || [[ -z "${CRONOMETER_PASSWORD:-}" ]]; } && [[ -f "$CREDS_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$CREDS_FILE"
fi

# Fallback: interactive prompt (password input hidden).
[[ -n "${CRONOMETER_EMAIL:-}" ]]    || read -r  -p "Cronometer email: " CRONOMETER_EMAIL
[[ -n "${CRONOMETER_PASSWORD:-}" ]] || { read -r -s -p "Cronometer password: " CRONOMETER_PASSWORD; echo; }
export CRONOMETER_EMAIL CRONOMETER_PASSWORD

# join pass-through args (bash 3.2 friendly — macOS default shell)
arg_str=""
[[ ${#ARGS[@]} -gt 0 ]] && arg_str="${ARGS[*]}"

if [[ -n "$REMOTE" ]]; then
  arch="${REMOTE_ARCH:-amd64}"
  echo "Building linux/${arch} and running on ${REMOTE} ..."
  bin="$(mktemp -t debug_export.XXXXXX)"
  GOOS=linux GOARCH="$arch" go build -o "$bin" ./cmd/debug_export
  scp -q "$bin" "${REMOTE}:/tmp/debug_export-linux"
  rm -f "$bin"
  # NOTE: creds are passed via the remote command env; they're briefly visible
  # in the remote process list. Fine for a one-off diagnostic on your own host.
  ssh "$REMOTE" "CRONOMETER_EMAIL='${CRONOMETER_EMAIL}' CRONOMETER_PASSWORD='${CRONOMETER_PASSWORD}' /tmp/debug_export-linux ${arg_str}; rm -f /tmp/debug_export-linux"
elif [[ ${#ARGS[@]} -gt 0 ]]; then
  go run ./cmd/debug_export "${ARGS[@]}"
else
  go run ./cmd/debug_export
fi
