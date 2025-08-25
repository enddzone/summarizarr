#!/usr/bin/env bash
set -euo pipefail

# Summarizarr SQLCipher key rotation helper
# - Fetches CSRF token, logs in, then POSTs /api/rotate-encryption-key
# - Works with cookie-based session and CSRF protection
#
# Usage:
#   ./rotate.sh -u http://localhost:8080 -e you@example.com -p 'your_password' [-c ./cookies.txt]
#   BASE, EMAIL, PASS, COOKIE can be provided via env vars instead of flags.
#
# Notes:
# - If -u/BASE is omitted, script tries http://localhost:8080 then http://localhost:8081
# - Requires curl and python3. jq is optional for pretty JSON output.

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

BASE="${BASE:-}"
EMAIL="${EMAIL:-}"
PASS="${PASS:-}"
COOKIE="${COOKIE:-$SCRIPT_DIR/cookies.txt}"

usage() {
  cat <<EOF
Usage: $0 [-u BASE_URL] -e EMAIL -p PASSWORD [-c COOKIE_FILE]

Options:
  -u  Base URL of API (e.g., http://localhost:8080). If omitted, script will probe 8080 then 8081.
  -e  Login email (or set EMAIL env var)
  -p  Login password (or set PASS env var)
  -c  Cookie file path (default: $COOKIE) (or set COOKIE env var)

Environment variables:
  BASE, EMAIL, PASS, COOKIE

Examples:
  BASE=http://localhost:8081 EMAIL=me@example.com PASS='secret' $0
  $0 -u http://localhost:8080 -e me@example.com -p 'secret' -c /tmp/summarizarr.cookies
EOF
}

while getopts ":u:e:p:c:h" opt; do
  case "$opt" in
    u) BASE="$OPTARG" ;;
    e) EMAIL="$OPTARG" ;;
    p) PASS="$OPTARG" ;;
    c) COOKIE="$OPTARG" ;;
    h) usage; exit 0 ;;
    :) echo "Missing argument for -$OPTARG" >&2; usage; exit 2 ;;
    \?) echo "Unknown option -$OPTARG" >&2; usage; exit 2 ;;
  esac
done

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: required command not found: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd python3

# Detect BASE if not supplied
probe_base() {
  local try_base="$1"
  local code
  code=$(curl -sS -o /dev/null -w "%{http_code}" "$try_base/health" || true)
  if [[ "$code" == "200" ]]; then
    echo "$try_base"
  else
    echo "" # not found
  fi
}

if [[ -z "${BASE}" ]]; then
  BASE=$(probe_base "http://localhost:8080") || true
fi
if [[ -z "${BASE}" ]]; then
  BASE=$(probe_base "http://localhost:8081") || true
fi
if [[ -z "${BASE}" ]]; then
  echo "Unable to detect API base. Provide -u or set BASE. Tried 8080 and 8081." >&2
  exit 1
fi

if [[ -z "${EMAIL}" || -z "${PASS}" ]]; then
  echo "Email and password are required." >&2
  usage
  exit 2
fi

pretty() {
  if command -v jq >/dev/null 2>&1; then
    jq .
  else
    python3 - <<'PY'
import json,sys
try:
    obj=json.load(sys.stdin)
    print(json.dumps(obj, indent=2))
except Exception:
    sys.stdout.write(sys.stdin.read())
PY
  fi
}

extract_json_field() {
  # $1 = JSON file path, $2 = key (top-level)
  local file="$1" key="$2"
  if command -v jq >/dev/null 2>&1; then
    jq -r ".${key}" "$file"
  else
    python3 - "$file" "$key" <<'PY'
import json,sys
f=sys.argv[1]; k=sys.argv[2]
with open(f) as fh:
    try:
        d=json.load(fh)
        v=d.get(k, '')
        print(v if isinstance(v,(str,int,float,bool)) or v is None else json.dumps(v))
    except Exception:
        print('')
PY
  fi
}

tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t rotate)
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

echo "Using BASE=$BASE"
echo "Using COOKIE=$COOKIE"

# 1) Get CSRF token (creates session cookie)
csrf_json="$tmpdir/csrf.json"
curl -sS -c "$COOKIE" "$BASE/api/auth/csrf-token" -o "$csrf_json"
CSRF=$(extract_json_field "$csrf_json" csrf_token)
if [[ -z "$CSRF" || "$CSRF" == "null" ]]; then
  echo "Failed to obtain CSRF token." >&2
  echo "Response:" >&2
  cat "$csrf_json" >&2 || true
  exit 1
fi
echo "Obtained CSRF token."

# 2) Login
login_json="$tmpdir/login.json"
login_code=$(curl -sS -b "$COOKIE" -c "$COOKIE" \
  -H "Content-Type: application/json" -H "X-CSRF-Token: $CSRF" \
  -X POST "$BASE/api/auth/login" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" \
  -w "%{http_code}" -o "$login_json")

if [[ "$login_code" != "200" ]]; then
  echo "Login failed (HTTP $login_code). Response:" >&2
  pretty < "$login_json" >&2 || true
  exit 1
fi
echo "Login successful."

# 3) Rotate encryption key
rotate_json="$tmpdir/rotate.json"
rotate_code=$(curl -sS -b "$COOKIE" -H "X-CSRF-Token: $CSRF" -X POST \
  "$BASE/api/rotate-encryption-key" -w "%{http_code}" -o "$rotate_json")

echo "Rotation response (HTTP $rotate_code):"
pretty < "$rotate_json" || true

status=$(extract_json_field "$rotate_json" status)
verification=$(extract_json_field "$rotate_json" verification_passed)

if [[ "$rotate_code" != "200" || "$status" != "success" || "$verification" != "true" ]]; then
  echo "Rotation reported failure or verification did not pass." >&2
  exit 1
fi

echo "Key rotation completed successfully."
