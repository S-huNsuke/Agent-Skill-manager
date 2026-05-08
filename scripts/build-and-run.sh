#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-run}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="agent-skills-manager"
BUNDLE_ID="com.wails.agent-skills-manager"
APP_BUNDLE="${ROOT_DIR}/build/bin/${APP_NAME}.app"
APP_BINARY="${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"

pkill -x "${APP_NAME}" >/dev/null 2>&1 || true

"${ROOT_DIR}/scripts/build.sh"

if [ ! -x "${APP_BINARY}" ]; then
  echo "missing app binary: ${APP_BINARY}" >&2
  exit 1
fi

open_app() {
  /usr/bin/open -n "${APP_BUNDLE}"
}

case "${MODE}" in
  run)
    open_app
    ;;
  --debug|debug)
    lldb -- "${APP_BINARY}"
    ;;
  --logs|logs)
    open_app
    /usr/bin/log stream --info --style compact --predicate "process == \"${APP_NAME}\""
    ;;
  --telemetry|telemetry)
    open_app
    /usr/bin/log stream --info --style compact --predicate "subsystem == \"${BUNDLE_ID}\""
    ;;
  --verify|verify)
    open_app
    sleep 1
    pgrep -x "${APP_NAME}" >/dev/null
    ;;
  *)
    echo "usage: $0 [run|--debug|--logs|--telemetry|--verify]" >&2
    exit 2
    ;;
esac
