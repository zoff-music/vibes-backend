#!/usr/bin/env sh
set -e

if [ -z "${CAST_EMULATOR_CMD+x}" ]; then
  if command -v npx >/dev/null 2>&1; then
    CAST_EMULATOR_CMD="npx --yes chromecast-device-emulator"
  else
    echo "Cast emulator: missing npx in PATH" >&2
    exit 1
  fi
fi

detect_interface() {
  if command -v ipconfig >/dev/null 2>&1; then
    if ipconfig getifaddr en0 >/dev/null 2>&1; then
      echo "en0"
      return
    fi
    if ipconfig getifaddr en1 >/dev/null 2>&1; then
      echo "en1"
      return
    fi
  fi
  if command -v route >/dev/null 2>&1; then
    route get default 2>/dev/null | awk '/interface:/{print $2; exit}'
    return
  fi
  if command -v ip >/dev/null 2>&1; then
    ip route get 1.1.1.1 2>/dev/null | awk '{for (i=1; i<=NF; i++) if ($i=="dev") {print $(i+1); exit}}'
  fi
}

: "${CAST_APP_ID:=1FAF5D9F}"
: "${CAST_RECEIVER_URL:=https://localhost/casting/receiver/}"
: "${CAST_DEVICE_NAME:=Vibez Emulator}"
: "${CAST_EMULATOR_INTERFACE:=$(detect_interface)}"
: "${CAST_EMULATOR_SCENARIO:=}"

export CAST_APP_ID
export CAST_RECEIVER_URL
export CAST_DEVICE_NAME

child_pid=""
cleanup() {
  if [ -n "$child_pid" ]; then
    kill "$child_pid" 2>/dev/null || true
  fi
}
trap cleanup INT TERM EXIT

echo "Starting cast emulator..."
echo "  app id: ${CAST_APP_ID}"
echo "  receiver: ${CAST_RECEIVER_URL}"
echo "  device: ${CAST_DEVICE_NAME}"
if [ -n "${CAST_EMULATOR_INTERFACE}" ]; then
  echo "  interface: ${CAST_EMULATOR_INTERFACE}"
fi
if [ -z "${CAST_EMULATOR_SCENARIO}" ]; then
  echo "Cast emulator: no scenario provided, skipping CLI start"
  exit 0
fi

base_cmd="${CAST_EMULATOR_CMD} start ${CAST_EMULATOR_SCENARIO}"

if [ -n "${CAST_EMULATOR_INTERFACE}" ] && \
  ! echo "${CAST_EMULATOR_ARGS:-}" | grep -q -- "--interface"; then
  set +e
  sh -c "${base_cmd} --interface ${CAST_EMULATOR_INTERFACE} ${CAST_EMULATOR_ARGS:-}" &
  child_pid=$!
  wait "$child_pid"
  status=$?
  set -e
  if [ $status -ne 0 ]; then
    echo "Cast emulator failed with interface; retrying without interface..." >&2
    sh -c "${base_cmd} ${CAST_EMULATOR_ARGS:-}" &
    child_pid=$!
    wait "$child_pid"
  fi
  exit 0
fi

sh -c "${base_cmd} ${CAST_EMULATOR_ARGS:-}" &
child_pid=$!
wait "$child_pid"
