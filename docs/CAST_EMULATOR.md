# Chromecast Device Emulator

The Cast receiver can be tested in a browser, but sender flows (device discovery,
session creation, media load) still need a Chromecast device. To test those
without hardware, run a Chromecast device emulator and point it at the local
receiver.

## Local Cast Device (No Hardware)

When running `make local-dev`, the Platform app exposes a **Local Cast (Emulator)**
device in the cast picker. Selecting it opens
`https://localhost/casting/receiver/` in a new window and uses `postMessage` to
sync playback state, queue, and room info.

This does not require a physical Chromecast or Chrome’s Cast device discovery.
It is enabled via `VITE_CAST_LOCAL_EMULATOR=true` in `make local-dev`.

## Optional: ajhsu/chromecast-device-emulator

1. Start the stack + emulator via Docker Compose:

```bash
make cast-dev
```

2. Configure it to use our receiver (override via env if needed):
   - App ID: `VITE_CAST_APP_ID` (default `1FAF5D9F` in `castManager.ts`)
   - Receiver URL: `VITE_CAST_RECEIVER_URL` (default `https://localhost/casting/receiver/`)
   - Emulator args: `CAST_EMULATOR_ARGS` (passed to the emulator CLI)

3. When prompted or configured by the emulator, use:
   - Device name: `Vibez Emulator`
   - App ID and receiver URL from step 2

## Local Dev (make local-dev)

`make local-dev` starts a host-based emulator via `npx` from
`emulation/run-cast-emulator.sh`. The CLI requires a recorded scenario file, so
set `CAST_EMULATOR_SCENARIO` if you want to run it.

Env overrides:
- `CAST_EMULATOR_CMD` (default `npx --yes chromecast-device-emulator`)
- `CAST_EMULATOR_ARGS` (extra CLI args)
- `CAST_EMULATOR_INTERFACE` (network interface; auto-detected if unset)
- `CAST_EMULATOR_SCENARIO` (path to scenario JSON)
- `CAST_APP_ID`, `CAST_RECEIVER_URL`, `CAST_DEVICE_NAME`

## Notes

- The emulator advertises via mDNS; ensure your machine allows local discovery.
- The sender and emulator must be on the same LAN segment.
- If Docker mDNS broadcasting is blocked on your host OS, run the emulator on the host instead of in Docker.
