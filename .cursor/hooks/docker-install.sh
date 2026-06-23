#!/usr/bin/env bash
set -euo pipefail

json_input="$(cat)"

status="$(printf '%s' "$json_input" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
if [[ -n "$status" && "$status" != "completed" ]]; then
  exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/../../installation/docker/install-or-update.sh"
